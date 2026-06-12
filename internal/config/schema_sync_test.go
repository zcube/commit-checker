package config

// JSON 스키마(.commit-checker.schema.json)와 Config 구조체의 동기화 검증.
// 새 설정 필드를 추가하면서 스키마 갱신을 빠뜨리면 이 테스트가 실패한다.
// 이름이 아닌 경로 단위(예: lint.toml.enabled)로 대조해, 같은 이름의 키가
// 다른 위치에 존재할 때의 오탐을 방지한다.

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// collectConfigPaths 는 구조체의 yaml 태그를 재귀 순회하며 "a.b.c" 경로 집합을 만든다.
// 슬라이스/포인터는 요소 타입으로 내려가고, map 등 동적 키 타입에서는 중단한다.
func collectConfigPaths(t reflect.Type, prefix string, out map[string]bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("yaml")
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}
		if name == "" {
			// yaml.v3 기본 규칙: 태그가 없으면 필드명 소문자
			name = strings.ToLower(f.Name)
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		out[path] = true

		ft := f.Type
		for ft.Kind() == reflect.Pointer || ft.Kind() == reflect.Slice || ft.Kind() == reflect.Array {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			collectConfigPaths(ft, path, out)
		}
	}
}

// collectSchemaPaths 는 JSON 스키마의 properties 를 재귀 순회하며 경로 집합을 만든다.
// 배열은 items 로 내려가고(경로 동일), $ref 는 definitions 에서 해석한다.
func collectSchemaPaths(node map[string]any, prefix string, defs map[string]any, out map[string]bool) {
	if ref, ok := node["$ref"].(string); ok {
		const p = "#/definitions/"
		if name, found := strings.CutPrefix(ref, p); found {
			if def, ok := defs[name].(map[string]any); ok {
				collectSchemaPaths(def, prefix, defs, out)
			}
		}
		return
	}
	if items, ok := node["items"].(map[string]any); ok {
		collectSchemaPaths(items, prefix, defs, out)
	}
	props, ok := node["properties"].(map[string]any)
	if !ok {
		return
	}
	for key, v := range props {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		out[path] = true
		if child, ok := v.(map[string]any); ok {
			collectSchemaPaths(child, path, defs, out)
		}
	}
}

func loadSchema(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("../../.commit-checker.schema.json")
	if err != nil {
		t.Fatalf("스키마 파일 읽기 실패: %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("스키마 JSON 파싱 실패: %v", err)
	}
	return schema
}

func sortedDiff(a, b map[string]bool) []string {
	var diff []string
	for k := range a {
		if !b[k] {
			diff = append(diff, k)
		}
	}
	sort.Strings(diff)
	return diff
}

// TestSchemaSync_설정필드가스키마에존재: Config 구조체의 모든 yaml 경로가
// 스키마에 정의되어 있어야 한다 (스키마 누락 검출).
func TestSchemaSync_설정필드가스키마에존재(t *testing.T) {
	schema := loadSchema(t)
	defs, _ := schema["definitions"].(map[string]any)

	configPaths := map[string]bool{}
	collectConfigPaths(reflect.TypeOf(Config{}), "", configPaths)

	schemaPaths := map[string]bool{}
	collectSchemaPaths(schema, "", defs, schemaPaths)

	if missing := sortedDiff(configPaths, schemaPaths); len(missing) > 0 {
		t.Errorf("Config 에 있으나 스키마에 없는 경로 %d개 — .commit-checker.schema.json 갱신 필요:\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// TestSchemaSync_스키마키가설정에존재: 스키마의 모든 경로가 Config 구조체에
// 존재해야 한다 (제거된 필드의 스키마 잔존 검출).
func TestSchemaSync_스키마키가설정에존재(t *testing.T) {
	schema := loadSchema(t)
	defs, _ := schema["definitions"].(map[string]any)

	configPaths := map[string]bool{}
	collectConfigPaths(reflect.TypeOf(Config{}), "", configPaths)

	schemaPaths := map[string]bool{}
	collectSchemaPaths(schema, "", defs, schemaPaths)

	if stale := sortedDiff(schemaPaths, configPaths); len(stale) > 0 {
		t.Errorf("스키마에 있으나 Config 에 없는 경로 %d개 — 코드에서 제거된 필드인지 확인 필요:\n  %s",
			len(stale), strings.Join(stale, "\n  "))
	}
}
