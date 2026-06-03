package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("테스트 데이터 읽기 실패 %s: %v", name, err)
	}
	return data
}

func TestDetectVersion(t *testing.T) {
	tests := []struct {
		file    string
		version Version
	}{
		{"current.yml", VersionCurrent},
		{"v1_1_0.yml", VersionV110},
		{"v1_0_2.yml", VersionV102},
		{"v1_0_1.yml", VersionV101},
		{"v1_0_0.yml", VersionV100},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data := readTestdata(t, tt.file)
			got := DetectVersion(data)
			if got != tt.version {
				t.Errorf("DetectVersion(%s) = %q, want %q", tt.file, got, tt.version)
			}
		})
	}
}

func TestDetectVersion_InvalidYAML(t *testing.T) {
	data := []byte(":::invalid yaml:::")
	got := DetectVersion(data)
	if got != VersionUnknown {
		t.Errorf("DetectVersion(invalid) = %q, want %q", got, VersionUnknown)
	}
}

func TestDetectVersion_UnknownField(t *testing.T) {
	data := []byte("completely_unknown_field: true\n")
	got := DetectVersion(data)
	if got != VersionUnknown {
		t.Errorf("DetectVersion(unknown field) = %q, want %q", got, VersionUnknown)
	}
}

func TestMigrate_Current(t *testing.T) {
	data := readTestdata(t, "current.yml")
	result, err := Migrate(data)
	if err != nil {
		t.Fatalf("Migrate(current) 실패: %v", err)
	}
	if result.DetectedVersion != VersionCurrent {
		t.Errorf("DetectedVersion = %q, want %q", result.DetectedVersion, VersionCurrent)
	}
	if len(result.Applied) != 0 {
		t.Errorf("Applied = %v, want empty", result.Applied)
	}
}

func TestMigrate_V102(t *testing.T) {
	data := readTestdata(t, "v1_0_2.yml")
	expected := readTestdata(t, "v1_0_2_migrated.yml")
	result, err := Migrate(data)
	if err != nil {
		t.Fatalf("Migrate(v1.0.2) 실패: %v", err)
	}
	if result.DetectedVersion != VersionV102 {
		t.Errorf("DetectedVersion = %q, want %q", result.DetectedVersion, VersionV102)
	}
	// v1.0.2 → v1.2.0: required_language → locale 통합 적용됨
	if len(result.Applied) == 0 {
		t.Error("Applied가 비어있으면 안 됨 (required_language → locale 마이그레이션 필요)")
	}
	if string(result.Data) != string(expected) {
		t.Errorf("마이그레이션 결과 불일치:\ngot:\n%s\nwant:\n%s", result.Data, expected)
	}
}

// TestMigrate_V110: v1.1.0 (required_language) → v1.2.0 (locale) 마이그레이션.
func TestMigrate_V110(t *testing.T) {
	data := readTestdata(t, "v1_1_0.yml")
	expected := readTestdata(t, "v1_1_0_migrated.yml")
	result, err := Migrate(data)
	if err != nil {
		t.Fatalf("Migrate(v1.1.0) 실패: %v", err)
	}
	if result.DetectedVersion != VersionV110 {
		t.Errorf("DetectedVersion = %q, want %q", result.DetectedVersion, VersionV110)
	}
	if len(result.Applied) == 0 {
		t.Error("Applied가 비어있으면 안 됨")
	}
	if string(result.Data) != string(expected) {
		t.Errorf("마이그레이션 결과 불일치:\ngot:\n%s\nwant:\n%s", result.Data, expected)
	}
}

func TestMigrate_V100(t *testing.T) {
	data := readTestdata(t, "v1_0_0.yml")
	expected := readTestdata(t, "v1_0_0_migrated.yml")

	result, err := Migrate(data)
	if err != nil {
		t.Fatalf("Migrate(v1.0.0) 실패: %v", err)
	}
	if result.DetectedVersion != VersionV100 {
		t.Errorf("DetectedVersion = %q, want %q", result.DetectedVersion, VersionV100)
	}
	if len(result.Applied) == 0 {
		t.Error("Applied가 비어있으면 안 됨")
	}
	if string(result.Data) != string(expected) {
		t.Errorf("마이그레이션 결과 불일치:\ngot:\n%s\nwant:\n%s", result.Data, expected)
	}
}

func TestMigrate_V101(t *testing.T) {
	data := readTestdata(t, "v1_0_1.yml")
	expected := readTestdata(t, "v1_0_1_migrated.yml")

	result, err := Migrate(data)
	if err != nil {
		t.Fatalf("Migrate(v1.0.1) 실패: %v", err)
	}
	if result.DetectedVersion != VersionV101 {
		t.Errorf("DetectedVersion = %q, want %q", result.DetectedVersion, VersionV101)
	}
	if len(result.Applied) == 0 {
		t.Error("Applied가 비어있으면 안 됨")
	}
	if string(result.Data) != string(expected) {
		t.Errorf("마이그레이션 결과 불일치:\ngot:\n%s\nwant:\n%s", result.Data, expected)
	}
}

func TestMigrate_Unknown(t *testing.T) {
	data := []byte("unknown_field: true\n")
	_, err := Migrate(data)
	if err == nil {
		t.Error("Migrate(unknown) should return error")
	}
}

// TestDetectVersion_WithPreset: preset 필드가 있어도 기존 버전이 올바르게 감지되는지 확인.
func TestDetectVersion_WithPreset(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		version Version
	}{
		{
			name: "v1.0.0 + preset",
			yaml: `
preset:
  url: https://example.com/preset.yml
comment_language:
  required_language: korean
commit_message:
  no_coauthor: true
exceptions:
  global_ignore: []
`,
			version: VersionV100,
		},
		{
			name: "v1.0.1 + preset",
			yaml: `
preset:
  url: https://example.com/preset.yml
comment_language:
  required_language: korean
  no_emoji: false
commit_message:
  enabled: true
  no_coauthor: true
binary_file:
  enabled: true
`,
			version: VersionV101,
		},
		{
			name: "v1.0.2 + preset",
			yaml: `
preset:
  url: https://example.com/preset.yml
comment_language:
  required_language: korean
commit_message:
  no_ai_coauthor: true
binary_file:
  enabled: true
`,
			version: VersionV102,
		},
		{
			name: "current + preset",
			yaml: `
preset:
  url: https://example.com/preset.yml
comment_language:
  allowed_words: []
commit_message:
  no_ai_coauthor: true
`,
			version: VersionCurrent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectVersion([]byte(tt.yaml))
			if got != tt.version {
				t.Errorf("DetectVersion = %q, want %q", got, tt.version)
			}
		})
	}
}

// TestMigrate_WithPreset_V100: preset 필드가 포함된 v1.0.0 설정이 마이그레이션되는지 확인.
func TestMigrate_WithPreset_V100(t *testing.T) {
	data := []byte(`
preset:
  url: https://example.com/preset.yml
commit_message:
  no_coauthor: true
`)
	result, err := Migrate(data)
	if err != nil {
		t.Fatalf("Migrate 실패: %v", err)
	}
	if result.DetectedVersion != VersionV100 {
		t.Errorf("DetectedVersion = %q, want %q", result.DetectedVersion, VersionV100)
	}
	if len(result.Applied) == 0 {
		t.Error("마이그레이션 규칙이 적용되어야 함")
	}
	if !strings.Contains(string(result.Data), "no_ai_coauthor") {
		t.Error("no_coauthor가 no_ai_coauthor로 마이그레이션되어야 함")
	}
	if strings.Contains(string(result.Data), "no_coauthor:") {
		t.Error("no_coauthor가 여전히 존재함")
	}
	if !strings.Contains(string(result.Data), "https://example.com/preset.yml") {
		t.Error("preset.url이 보존되어야 함")
	}
}

func TestMigrate_PreservesComments(t *testing.T) {
	data := []byte(`# 설정 파일 주석
commit_message:
  # AI 공동 작성자 차단
  no_coauthor: true  # 이 옵션 활성화
  locale: ko
`)
	result, err := Migrate(data)
	if err != nil {
		t.Fatalf("Migrate 실패: %v", err)
	}
	got := string(result.Data)
	// 주석이 보존되는지 확인
	if !strings.Contains(got, "# 설정 파일 주석") {
		t.Error("파일 상단 주석이 보존되지 않음")
	}
	if !strings.Contains(got, "# AI 공동 작성자 차단") {
		t.Error("인라인 주석이 보존되지 않음")
	}
	if !strings.Contains(got, "no_ai_coauthor: true") {
		t.Error("no_coauthor가 no_ai_coauthor로 변경되지 않음")
	}
	if !strings.Contains(got, "# 이 옵션 활성화") {
		t.Error("trailing 주석이 보존되지 않음")
	}
	if strings.Contains(got, "no_coauthor:") {
		t.Error("no_coauthor가 여전히 존재함")
	}
}
