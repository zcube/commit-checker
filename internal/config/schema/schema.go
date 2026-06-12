// Package schema 는 설정 파일의 스키마 버전 감지와 현재 버전으로의
// 마이그레이션을 담당합니다.
//
// 모든 버전 정보는 versionChain (오래된 → 최신 순서) 하나에 선언적으로
// 모여 있으며, DetectVersion 과 Migrate 는 이 체인만 따라 동작합니다.
//
// 새 스키마 버전 추가 절차 (versionChain 에 엔트리 1개 추가로 끝):
//  1. 새 버전의 Config 구조체를 정의 (vX_Y_Z.go 또는 schema.go)
//  2. Version 상수를 추가하고 VersionCurrent 를 새 버전으로 올림
//  3. versionChain 끝에 versionSpec 엔트리 1개를 추가
//     - parse: 새 스키마의 strict 파싱 함수
//     - signature: 새 버전에서 도입된 마커 필드 경로
//     (필드 제거만 있는 버전은 직전 버전의 마커를 공유)
//     - 직전 버전 엔트리의 migrateUp 에 새 버전으로 가는 변환 규칙을 채움
//
// DetectVersion / Migrate 등 다른 코드는 수정할 필요가 없습니다.
package schema

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Version: 감지된 설정 파일의 스키마 버전.
type Version string

const (
	VersionCurrent Version = "v1.2.0" // v1.2.0+ (locale 통일: required_language → locale)
	VersionV110    Version = "v1.1.0" // v1.1.0~v1.1.x (allowed_words, encoding unicode, required_language 사용)
	VersionV102    Version = "v1.0.2" // v1.0.2~v1.0.4 (no_ai_coauthor, allowed_words 없음)
	VersionV101    Version = "v1.0.1" // v1.0.1 (no_coauthor, 전체 섹션)
	VersionV100    Version = "v1.0.0" // v1.0.0 (no_coauthor, 최소 섹션)
	VersionUnknown Version = "unknown"
)

// PresetConfig: 원격 URL에서 불러올 기본 설정 프리셋 (v1.1.0+).
type PresetConfig struct {
	URL   string                  `yaml:"url"`
	Cache AllowedWordsCacheConfig `yaml:"cache"`
}

// ConfigCurrent: v1.2.0+ 설정 스키마.
// 변경 사항(v1.1.0 → v1.2.0):
//   - comment_language.required_language → comment_language.locale (필드 통합)
//   - commit_message.language_check.required_language → commit_message.language_check.locale
//   - file_languages[].language → file_languages[].locale
//
// 단, strict 파싱에서는 unknown 필드를 거부하므로 v1.2.0 스키마는 새 이름만 허용합니다.
// 구 필드 사용 시에는 DetectVersion 이 v1.1.0 으로 판정하고 Migrate 가 자동 변환합니다.
type ConfigCurrent struct {
	Preset          PresetConfig                 `yaml:"preset"`
	CommentLanguage CommentLanguageConfigCurrent `yaml:"comment_language"`
	CommitMessage   CommitMessageConfigCurrent   `yaml:"commit_message"`
	BinaryFile      BinaryFileConfig             `yaml:"binary_file"`
	Lint            LintConfig                   `yaml:"lint"`
	Encoding        EncodingConfigCurrent        `yaml:"encoding"`
	EditorConfig    EditorConfigConfig           `yaml:"editorconfig"`
	Exceptions      ExceptionsConfig             `yaml:"exceptions"`
}

// CommentLanguageConfigCurrent: v1.2.0+ 주석 언어 검사 설정.
// required_language 제거 (locale 로 통합).
type CommentLanguageConfigCurrent struct {
	Enabled              *bool                   `yaml:"enabled"`
	Languages            []string                `yaml:"languages"`
	Extensions           []string                `yaml:"extensions"`
	MinLength            int                     `yaml:"min_length"`
	SkipDirectives       []string                `yaml:"skip_directives"`
	CheckMode            string                  `yaml:"check_mode"`
	IgnoreFiles          []string                `yaml:"ignore_files"`
	Locale               string                  `yaml:"locale"`
	FileLanguages        []FileLanguageRuleV120  `yaml:"file_languages"`
	NoEmoji              *bool                   `yaml:"no_emoji"`
	CheckStrings         *bool                   `yaml:"check_strings"`
	SkipTechnicalStrings *bool                   `yaml:"skip_technical_strings"`
	AllowedWords         []string                `yaml:"allowed_words"`
	AllowedWordsFile     string                  `yaml:"allowed_words_file"`
	AllowedWordsURL      string                  `yaml:"allowed_words_url"`
	AllowedWordsCache    AllowedWordsCacheConfig `yaml:"allowed_words_cache"`
}

// FileLanguageRuleV120: v1.2.0+ 파일별 로케일 규칙. language → locale.
type FileLanguageRuleV120 struct {
	Pattern string `yaml:"pattern"`
	Locale  string `yaml:"locale"`
}

// ConfigV110: v1.1.0~v1.1.x 설정 스키마 (locale 통합 이전).
type ConfigV110 struct {
	Preset          PresetConfig              `yaml:"preset"`
	CommentLanguage CommentLanguageConfigV110 `yaml:"comment_language"`
	CommitMessage   CommitMessageConfigV110   `yaml:"commit_message"`
	BinaryFile      BinaryFileConfig          `yaml:"binary_file"`
	Lint            LintConfig                `yaml:"lint"`
	Encoding        EncodingConfigCurrent     `yaml:"encoding"`
	EditorConfig    EditorConfigConfig        `yaml:"editorconfig"`
	Exceptions      ExceptionsConfig          `yaml:"exceptions"`
}

// CommentLanguageConfigV110: v1.1.0 주석 언어 검사 설정.
// required_language 와 locale 동시 사용.
type CommentLanguageConfigV110 struct {
	Enabled              *bool                   `yaml:"enabled"`
	RequiredLanguage     string                  `yaml:"required_language"`
	Languages            []string                `yaml:"languages"`
	Extensions           []string                `yaml:"extensions"`
	MinLength            int                     `yaml:"min_length"`
	SkipDirectives       []string                `yaml:"skip_directives"`
	CheckMode            string                  `yaml:"check_mode"`
	IgnoreFiles          []string                `yaml:"ignore_files"`
	Locale               string                  `yaml:"locale"`
	FileLanguages        []FileLanguageRule      `yaml:"file_languages"`
	NoEmoji              *bool                   `yaml:"no_emoji"`
	CheckStrings         *bool                   `yaml:"check_strings"`
	SkipTechnicalStrings *bool                   `yaml:"skip_technical_strings"`
	AllowedWords         []string                `yaml:"allowed_words"`
	AllowedWordsFile     string                  `yaml:"allowed_words_file"`
	AllowedWordsURL      string                  `yaml:"allowed_words_url"`
	AllowedWordsCache    AllowedWordsCacheConfig `yaml:"allowed_words_cache"`
}

// CommitMessageConfigV110: v1.1.0 커밋 메시지 설정.
// language_check 가 required_language 사용.
type CommitMessageConfigV110 struct {
	Enabled              *bool                       `yaml:"enabled"`
	NoAICoauthor         *bool                       `yaml:"no_ai_coauthor"`
	CoauthorRemoveEmails []string                    `yaml:"coauthor_remove_emails"`
	NoUnicodeSpaces      *bool                       `yaml:"no_unicode_spaces"`
	NoAmbiguousChars     *bool                       `yaml:"no_ambiguous_chars"`
	NoBadRunes           *bool                       `yaml:"no_bad_runes"`
	NoEmoji              *bool                       `yaml:"no_emoji"`
	Locale               string                      `yaml:"locale"`
	LanguageCheck        CommitMessageLanguageConfig `yaml:"language_check"`
	ConventionalCommit   ConventionalCommitConfig    `yaml:"conventional_commit"`
}

// AllowedWordsCacheConfig: URL 캐싱 설정 (v1.1.0+).
type AllowedWordsCacheConfig struct {
	Enabled *bool  `yaml:"enabled"`
	TTL     string `yaml:"ttl"`
	Dir     string `yaml:"dir"`
}

// CommitMessageConfigCurrent: v1.2.0+ 커밋 메시지 검사 설정.
// language_check 가 locale 사용으로 통일됨.
type CommitMessageConfigCurrent struct {
	Enabled              *bool                           `yaml:"enabled"`
	NoAICoauthor         *bool                           `yaml:"no_ai_coauthor"`
	CoauthorRemoveEmails []string                        `yaml:"coauthor_remove_emails"`
	NoUnicodeSpaces      *bool                           `yaml:"no_unicode_spaces"`
	NoAmbiguousChars     *bool                           `yaml:"no_ambiguous_chars"`
	NoBadRunes           *bool                           `yaml:"no_bad_runes"`
	NoEmoji              *bool                           `yaml:"no_emoji"`
	Locale               string                          `yaml:"locale"`
	LanguageCheck        CommitMessageLanguageConfigV120 `yaml:"language_check"`
	ConventionalCommit   ConventionalCommitConfig        `yaml:"conventional_commit"`
}

// CommitMessageLanguageConfigV120: v1.2.0+ 커밋 메시지 본문 언어 검사 설정.
// required_language 제거, locale 만 사용.
type CommitMessageLanguageConfigV120 struct {
	Enabled      *bool    `yaml:"enabled"`
	Locale       string   `yaml:"locale"`
	MinLength    int      `yaml:"min_length"`
	SkipPrefixes []string `yaml:"skip_prefixes"`
}

// EncodingConfigCurrent: v1.1.0+ 인코딩 설정.
type EncodingConfigCurrent struct {
	Enabled          *bool    `yaml:"enabled"`
	RequireUTF8      *bool    `yaml:"require_utf8"`
	NoInvisibleChars *bool    `yaml:"no_invisible_chars"`
	NoAmbiguousChars *bool    `yaml:"no_ambiguous_chars"`
	Locale           string   `yaml:"locale"`
	IgnoreFiles      []string `yaml:"ignore_files"`
}

// ConfigV102: v1.0.2~v1.0.4 설정 스키마.
// no_ai_coauthor 사용, allowed_words/encoding unicode 미지원.
// CommitMessage 는 v1.1.0 형태(language_check.required_language)를 사용.
type ConfigV102 struct {
	Preset          PresetConfig              `yaml:"preset"`
	CommentLanguage CommentLanguageConfigV102 `yaml:"comment_language"`
	CommitMessage   CommitMessageConfigV110   `yaml:"commit_message"`
	BinaryFile      BinaryFileConfig          `yaml:"binary_file"`
	Lint            LintConfig                `yaml:"lint"`
	Encoding        EncodingConfigV101        `yaml:"encoding"`
	EditorConfig    EditorConfigConfig        `yaml:"editorconfig"`
	Exceptions      ExceptionsConfig          `yaml:"exceptions"`
}

// CommentLanguageConfigV102: v1.0.2~v1.0.4 주석 언어 검사 설정.
// allowed_words 관련 필드 없음.
type CommentLanguageConfigV102 struct {
	Enabled              *bool              `yaml:"enabled"`
	RequiredLanguage     string             `yaml:"required_language"`
	Languages            []string           `yaml:"languages"`
	Extensions           []string           `yaml:"extensions"`
	MinLength            int                `yaml:"min_length"`
	SkipDirectives       []string           `yaml:"skip_directives"`
	CheckMode            string             `yaml:"check_mode"`
	IgnoreFiles          []string           `yaml:"ignore_files"`
	Locale               string             `yaml:"locale"`
	FileLanguages        []FileLanguageRule `yaml:"file_languages"`
	NoEmoji              *bool              `yaml:"no_emoji"`
	CheckStrings         *bool              `yaml:"check_strings"`
	SkipTechnicalStrings *bool              `yaml:"skip_technical_strings"`
}

// tryParseStrict: YAML 데이터를 주어진 타입으로 strict 모드 파싱 시도.
func tryParseStrict[T any](data []byte) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var cfg T
	return dec.Decode(&cfg)
}

// versionSpec: 스키마 한 버전을 선언하는 단일 엔트리.
// 버전 감지·마이그레이션에 필요한 모든 정보를 이 구조체 하나에 모읍니다.
type versionSpec struct {
	// version: 이 엔트리가 나타내는 스키마 버전.
	version Version
	// parse: 이 버전 스키마로의 strict 파싱 시도 (모르는 필드가 있으면 실패).
	parse func(data []byte) error
	// signature: 이 버전에서 도입된 필드를 식별하는 YAML 경로 목록 (문서화·판별용).
	// 문법: "a.b" (중첩 맵 키), "a.b[].c" (시퀀스 원소의 키).
	// 구 버전 스키마가 신 버전의 부분집합이라 양쪽 strict 파싱이 모두 성공할 때,
	// 이 경로 중 하나라도 존재하면 신 버전으로 승격하는 근거가 됩니다.
	// nil 이면 마커 없음 (최초 버전).
	signature []string
	// migrateUp: 바로 다음 버전으로 가는 YAML 텍스트 변환 규칙 (주석 보존).
	// 필드 추가만 있는 버전 전환은 nil. 최신 버전은 항상 nil.
	migrateUp []MigrationRule
}

// v110MarkerFields: v1.1.0 에서 도입된 마커 필드 경로.
// v1.2.0 은 신규 필드 없이 required_language 류 구 필드만 제거했으므로
// 같은 마커를 공유합니다 (구 필드 유무는 strict 파싱이 구분).
var v110MarkerFields = []string{
	"comment_language.allowed_words",
	"comment_language.allowed_words_file",
	"comment_language.allowed_words_url",
	"comment_language.allowed_words_cache",
	"encoding.no_invisible_chars",
	"encoding.no_ambiguous_chars",
	"encoding.locale",
}

// versionChain: 지원하는 모든 스키마 버전 (오래된 → 최신 순서).
// 감지(DetectVersion)와 마이그레이션(Migrate)은 모두 이 체인 하나로 동작합니다.
var versionChain = []versionSpec{
	{
		version: VersionV100,
		parse:   tryParseStrict[ConfigV100],
		// 최초 버전: 마커 없음. v1.0.0 → v1.0.1 은 필드 추가만 있어 변환 불필요.
	},
	{
		version: VersionV101,
		parse:   tryParseStrict[ConfigV101],
		signature: []string{
			"binary_file",
			"lint",
			"encoding",
			"editorconfig",
			"commit_message.enabled",
			"commit_message.no_emoji",
			"comment_language.no_emoji",
		},
		// v1.0.1 → v1.0.2: no_coauthor 키 이름 변경.
		migrateUp: []MigrationRule{renameNoCoauthorRule},
	},
	{
		version:   VersionV102,
		parse:     tryParseStrict[ConfigV102],
		signature: []string{"commit_message.no_ai_coauthor"},
		// v1.0.2 → v1.1.0 은 필드 추가만 있어 변환 불필요.
	},
	{
		version:   VersionV110,
		parse:     tryParseStrict[ConfigV110],
		signature: v110MarkerFields,
		// v1.1.0 → v1.2.0: required_language 류 필드를 locale 로 통합.
		migrateUp: []MigrationRule{renameRequiredLanguageRule},
	},
	{
		version: VersionCurrent, // v1.2.0
		parse:   tryParseStrict[ConfigCurrent],
		// v1.2.0 마커는 v1.1.0 과 공유. required_language 류 구 필드가 있으면
		// strict 파싱이 실패하여 v1.1.0 으로 판별됩니다.
		signature: v110MarkerFields,
	},
}

// chainIndex: versionChain 에서 해당 버전의 인덱스를 반환. 없으면 -1.
func chainIndex(version Version) int {
	for i, spec := range versionChain {
		if spec.version == version {
			return i
		}
	}
	return -1
}

// DetectVersion: YAML 데이터의 스키마 버전을 감지.
// versionChain 을 구 버전부터 순서대로 strict 파싱 시도하여 기준 버전을 정하고,
// 부분집합 관계로 모호한 경우 최신 쪽부터 시그니처 필드로 승격 판별.
func DetectVersion(data []byte) Version {
	// 1단계: 구 버전 → 신 버전 순서로 strict 파싱 시도.
	// YAML에 해당 스키마가 모르는 필드가 있으면 실패.
	// 첫 번째 성공 = 가장 오래된 호환 버전(base).
	base := -1
	for i, spec := range versionChain {
		if spec.parse(data) == nil {
			base = i
			break
		}
	}
	if base < 0 {
		return VersionUnknown
	}

	// 2단계: 승격 판별. 구 버전 스키마가 신 버전의 부분집합이면
	// (예: v1.0.0 ⊂ v1.0.1, v1.0.2 ⊂ v1.1.0)
	// base 파싱도 성공하므로, 최신 쪽부터 역순으로 시그니처 마커 필드가
	// 존재하고 strict 파싱도 통과하는 가장 새로운 버전으로 승격.
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return versionChain[base].version
	}
	for i := len(versionChain) - 1; i > base; i-- {
		spec := versionChain[i]
		if hasAnyYAMLPath(raw, spec.signature) && spec.parse(data) == nil {
			return spec.version
		}
	}
	return versionChain[base].version
}

// hasAnyYAMLPath: 파싱된 YAML 맵에 주어진 경로 중 하나라도 존재하는지 확인.
func hasAnyYAMLPath(raw map[string]any, paths []string) bool {
	for _, path := range paths {
		if yamlPathExists(raw, strings.Split(path, ".")) {
			return true
		}
	}
	return false
}

// yamlPathExists: 노드에서 경로 세그먼트를 따라 키가 존재하는지 확인.
// "key[]" 세그먼트는 시퀀스의 각 원소에 대해 나머지 경로를 검사.
func yamlPathExists(node any, segs []string) bool {
	if len(segs) == 0 {
		return true
	}
	m, ok := node.(map[string]any)
	if !ok {
		return false
	}
	if key, isSeq := strings.CutSuffix(segs[0], "[]"); isSeq {
		items, ok := m[key].([]any)
		if !ok {
			return false
		}
		for _, item := range items {
			if yamlPathExists(item, segs[1:]) {
				return true
			}
		}
		return false
	}
	value, ok := m[segs[0]]
	if !ok {
		return false
	}
	return yamlPathExists(value, segs[1:])
}

// MigrationRule: 단일 마이그레이션 규칙.
type MigrationRule struct {
	// Description: 사람이 읽을 수 있는 변경 사항 설명.
	Description string
	// Apply: YAML 텍스트에 변환을 적용하고 변경된 텍스트를 반환.
	Apply func(data []byte) []byte
}

// renameNoCoauthorRule: v1.0.2 에서 변경된 키 이름을 적용하는 마이그레이션 규칙.
var renameNoCoauthorRule = MigrationRule{
	Description: "commit_message.no_coauthor → commit_message.no_ai_coauthor",
	Apply: func(data []byte) []byte {
		return renameYAMLKey(data, "no_coauthor", "no_ai_coauthor")
	},
}

// renameRequiredLanguageRule: v1.2.0 통일을 위한 마이그레이션 규칙.
// required_language 와 file_languages[].language 를 모두 locale 로 통합.
// 두 키가 동시 존재하면 locale 을 우선시키고 required_language 줄을 제거합니다.
var renameRequiredLanguageRule = MigrationRule{
	Description: "required_language / language → locale 통합 (v1.2.0)",
	Apply: func(data []byte) []byte {
		data = removeKeyIfPeerExists(data, "required_language", "locale")
		data = removeKeyIfPeerExists(data, "language", "locale")
		data = renameYAMLKey(data, "required_language", "locale")
		data = renameYAMLKey(data, "language", "locale")
		return data
	},
}

// MigrateResult: 마이그레이션 결과.
type MigrateResult struct {
	// DetectedVersion: 감지된 설정 파일의 스키마 버전.
	DetectedVersion Version
	// Applied: 적용된 마이그레이션 규칙 설명 목록.
	Applied []string
	// Data: 마이그레이션된 YAML 데이터.
	Data []byte
}

// Migrate: YAML 데이터를 현재 스키마로 마이그레이션.
// 감지된 버전부터 versionChain 을 따라 migrateUp 을 단계적으로 적용
// (v1.0.0 → v1.0.1 → ... → 최신). 이미 최신이면 Applied가 빈 슬라이스.
func Migrate(data []byte) (*MigrateResult, error) {
	version := DetectVersion(data)
	if version == VersionUnknown {
		return nil, fmt.Errorf("인식할 수 없는 설정 파일 형식입니다")
	}

	result := &MigrateResult{
		DetectedVersion: version,
		Data:            data,
	}

	if version == VersionCurrent {
		return result, nil
	}

	migrated := make([]byte, len(data))
	copy(migrated, data)

	// 감지된 버전부터 체인을 따라 한 단계씩 최신 버전으로 변환.
	// (DetectVersion 이 unknown 이 아닌 버전을 반환했으므로 체인에 반드시 존재)
	for _, spec := range versionChain[chainIndex(version):] {
		for _, rule := range spec.migrateUp {
			migrated = rule.Apply(migrated)
			result.Applied = append(result.Applied, rule.Description)
		}
	}

	// 마이그레이션 후 현재(최신) 스키마로 파싱 가능한지 검증.
	if err := tryParseStrict[ConfigCurrent](migrated); err != nil {
		return nil, fmt.Errorf("마이그레이션 후 검증 실패: %w", err)
	}

	result.Data = migrated
	return result, nil
}

// renameYAMLKey: YAML 텍스트에서 키 이름을 변경 (주석 보존).
func renameYAMLKey(data []byte, oldKey, newKey string) []byte {
	lines := strings.Split(string(data), "\n")
	oldPrefix := oldKey + ":"
	newPrefix := newKey + ":"
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, oldPrefix) {
			lines[i] = strings.Replace(line, oldPrefix, newPrefix, 1)
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// removeKeyIfPeerExists 는 동일 블록 내에 peerKey 와 staleKey 가 같이 존재할 때
// staleKey 줄을 제거합니다 (YAML 중복 키 충돌 방지).
// 블록 판별은 들여쓰기 기준입니다.
func removeKeyIfPeerExists(data []byte, staleKey, peerKey string) []byte {
	lines := strings.Split(string(data), "\n")
	stalePrefix := staleKey + ":"
	peerPrefix := peerKey + ":"

	indentOf := func(s string) int {
		n := 0
		for _, ch := range s {
			if ch == ' ' || ch == '\t' {
				n++
				continue
			}
			break
		}
		return n
	}
	keyAt := func(line string) string {
		trimmed := strings.TrimLeft(line, " \t")
		idx := strings.Index(trimmed, ":")
		if idx < 0 {
			return ""
		}
		return trimmed[:idx+1]
	}

	// 같은 들여쓰기·같은 블록 내에 peer 가 있는 stale 라인 인덱스 수집.
	toRemove := make(map[int]bool)
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(trimmed, stalePrefix) {
			continue
		}
		indent := indentOf(line)
		// 같은 indent 의 형제 키들 중 peer 가 있는지 확인 (앞·뒤 양방향).
		hasPeer := false
		scan := func(start, step int) {
			for j := start; j >= 0 && j < len(lines); j += step {
				if j == i {
					continue
				}
				ln := lines[j]
				if strings.TrimSpace(ln) == "" || strings.HasPrefix(strings.TrimLeft(ln, " \t"), "#") {
					continue
				}
				ind := indentOf(ln)
				if ind < indent {
					return // 블록 경계
				}
				if ind > indent {
					continue
				}
				if keyAt(ln) == peerPrefix {
					hasPeer = true
					return
				}
			}
		}
		scan(i-1, -1)
		if !hasPeer {
			scan(i+1, 1)
		}
		if hasPeer {
			toRemove[i] = true
		}
	}

	if len(toRemove) == 0 {
		return data
	}
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		if !toRemove[i] {
			out = append(out, line)
		}
	}
	return []byte(strings.Join(out, "\n"))
}
