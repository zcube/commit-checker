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
	VersionCurrent Version = "v1.1.0" // v1.1.0+ (현재: allowed_words, encoding unicode)
	VersionV102    Version = "v1.0.2" // v1.0.2~v1.0.4 (no_ai_coauthor, allowed_words 없음)
	VersionV101    Version = "v1.0.1" // v1.0.1 (no_coauthor, 전체 섹션)
	VersionV100    Version = "v1.0.0" // v1.0.0 (no_coauthor, 최소 섹션)
	VersionUnknown Version = "unknown"
)

// ConfigCurrent: v1.1.0+ 설정 스키마.
type ConfigCurrent struct {
	CommentLanguage CommentLanguageConfigCurrent `yaml:"comment_language"`
	CommitMessage   CommitMessageConfigCurrent   `yaml:"commit_message"`
	BinaryFile      BinaryFileConfig             `yaml:"binary_file"`
	Lint            LintConfig                   `yaml:"lint"`
	Encoding        EncodingConfigCurrent        `yaml:"encoding"`
	EditorConfig    EditorConfigConfig           `yaml:"editorconfig"`
	Exceptions      ExceptionsConfig             `yaml:"exceptions"`
}

// CommentLanguageConfigCurrent: v1.1.0+ 주석 언어 검사 설정.
type CommentLanguageConfigCurrent struct {
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

// AllowedWordsCacheConfig: URL 캐싱 설정 (v1.1.0+).
type AllowedWordsCacheConfig struct {
	Enabled *bool  `yaml:"enabled"`
	TTL     string `yaml:"ttl"`
	Dir     string `yaml:"dir"`
}

// CommitMessageConfigCurrent: v1.0.2+ 커밋 메시지 검사 설정.
type CommitMessageConfigCurrent struct {
	Enabled              *bool                      `yaml:"enabled"`
	NoAICoauthor         *bool                      `yaml:"no_ai_coauthor"`
	CoauthorRemoveEmails []string                   `yaml:"coauthor_remove_emails"`
	NoUnicodeSpaces      *bool                      `yaml:"no_unicode_spaces"`
	NoAmbiguousChars     *bool                      `yaml:"no_ambiguous_chars"`
	NoBadRunes           *bool                      `yaml:"no_bad_runes"`
	NoEmoji              *bool                      `yaml:"no_emoji"`
	Locale               string                     `yaml:"locale"`
	LanguageCheck        CommitMessageLanguageConfig `yaml:"language_check"`
	ConventionalCommit   ConventionalCommitConfig    `yaml:"conventional_commit"`
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
type ConfigV102 struct {
	CommentLanguage CommentLanguageConfigV102 `yaml:"comment_language"`
	CommitMessage   CommitMessageConfigCurrent `yaml:"commit_message"`
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

// DetectVersion: YAML 데이터의 스키마 버전을 감지.
// 구 버전부터 순서대로 strict 파싱을 시도하여 첫 번째 성공한 버전을 반환.
// 상위 호환(superset) 관계에서는 YAML 필드 존재 여부로 추가 판별.
func DetectVersion(data []byte) Version {
	// strict 파싱: 구 버전 → 신 버전 순서로 시도.
	// YAML에 해당 스키마가 모르는 필드가 있으면 실패.
	// 첫 번째 성공 = 가장 오래된 호환 버전.
	//
	// 단, 구 버전 스키마가 신 버전의 부분집합이면 둘 다 성공하므로
	// 추가 판별이 필요한 경우가 있음:
	//   - v1.0.0 ⊂ v1.0.1 (v1.0.0은 v1.0.1의 부분집합)
	//   - v1.0.2 ⊂ v1.1.0 (v1.0.2는 v1.1.0의 부분집합)

	type candidate struct {
		version  Version
		tryParse func([]byte) error
	}
	// 구 버전 → 신 버전 순서
	candidates := []candidate{
		{VersionV100, tryParseStrict[ConfigV100]},
		{VersionV101, tryParseStrict[ConfigV101]},
		{VersionV102, tryParseStrict[ConfigV102]},
		{VersionCurrent, tryParseStrict[ConfigCurrent]},
	}

	var matched Version
	for _, c := range candidates {
		if c.tryParse(data) == nil {
			matched = c.version
			break
		}
	}

	if matched == "" {
		return VersionUnknown
	}

	// 부분집합 관계 추가 판별
	switch matched {
	case VersionV100:
		// v1.0.0 config는 v1.0.1에서도 파싱됨 (부분집합).
		// v1.0.1 고유 필드(binary_file 등)가 있으면 v1.0.1.
		if hasV101Fields(data) {
			return VersionV101
		}
		return VersionV100
	case VersionV102:
		// v1.0.2 config는 v1.1.0에서도 파싱됨 (부분집합).
		// v1.1.0 고유 필드(allowed_words 등)가 있으면 v1.1.0.
		if hasV110Fields(data) {
			return VersionCurrent
		}
		return VersionV102
	}

	return matched
}

// hasV101Fields: v1.0.1에서 추가된 필드가 존재하는지 확인.
func hasV101Fields(data []byte) bool {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false
	}
	for _, key := range []string{"binary_file", "lint", "encoding", "editorconfig"} {
		if _, ok := raw[key]; ok {
			return true
		}
	}
	if cm, ok := raw["commit_message"].(map[string]any); ok {
		for _, key := range []string{"enabled", "no_emoji"} {
			if _, ok := cm[key]; ok {
				return true
			}
		}
	}
	if cl, ok := raw["comment_language"].(map[string]any); ok {
		if _, ok := cl["no_emoji"]; ok {
			return true
		}
	}
	return false
}

// hasV110Fields: v1.1.0에서 추가된 필드가 존재하는지 확인.
func hasV110Fields(data []byte) bool {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false
	}
	// comment_language 에 allowed_words 관련 필드
	if cl, ok := raw["comment_language"].(map[string]any); ok {
		for _, key := range []string{"allowed_words", "allowed_words_file", "allowed_words_url", "allowed_words_cache"} {
			if _, ok := cl[key]; ok {
				return true
			}
		}
	}
	// encoding 에 no_invisible_chars, no_ambiguous_chars, locale
	if enc, ok := raw["encoding"].(map[string]any); ok {
		for _, key := range []string{"no_invisible_chars", "no_ambiguous_chars", "locale"} {
			if _, ok := enc[key]; ok {
				return true
			}
		}
	}
	return false
}

// MigrationRule: 단일 마이그레이션 규칙.
type MigrationRule struct {
	// Description: 사람이 읽을 수 있는 변경 사항 설명.
	Description string
	// Apply: YAML 텍스트에 변환을 적용하고 변경된 텍스트를 반환.
	Apply func(data []byte) []byte
}

// migrationRules: 버전별 마이그레이션 규칙 매핑.
// key는 감지된 (구) 버전.
var migrationRules = map[Version][]MigrationRule{
	VersionV100: {
		{
			Description: "commit_message.no_coauthor → commit_message.no_ai_coauthor",
			Apply: func(data []byte) []byte {
				return renameYAMLKey(data, "no_coauthor", "no_ai_coauthor")
			},
		},
	},
	VersionV101: {
		{
			Description: "commit_message.no_coauthor → commit_message.no_ai_coauthor",
			Apply: func(data []byte) []byte {
				return renameYAMLKey(data, "no_coauthor", "no_ai_coauthor")
			},
		},
	},
	// VersionV102: 마이그레이션 불필요 (v1.1.0은 필드 추가만, 제거/이름 변경 없음)
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
// 이미 최신이면 Applied가 빈 슬라이스.
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

	rules, ok := migrationRules[version]
	if !ok {
		// 마이그레이션 규칙 없음 (예: v1.0.2 → 추가만 있어 변환 불필요)
		return result, nil
	}

	migrated := make([]byte, len(data))
	copy(migrated, data)

	for _, rule := range rules {
		migrated = rule.Apply(migrated)
		result.Applied = append(result.Applied, rule.Description)
	}

	// 마이그레이션 후 현재 스키마로 파싱 가능한지 검증.
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
