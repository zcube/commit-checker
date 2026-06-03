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
	URL   string                 `yaml:"url"`
	Cache AllowedWordsCacheConfig `yaml:"cache"`
}

// ConfigCurrent: v1.2.0+ 설정 스키마.
// 변경 사항(v1.1.0 → v1.2.0):
//   - comment_language.required_language → comment_language.locale (필드 통합)
//   - commit_message.language_check.required_language → commit_message.language_check.locale
//   - file_languages[].language → file_languages[].locale
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
	Preset          PresetConfig               `yaml:"preset"`
	CommentLanguage CommentLanguageConfigV110  `yaml:"comment_language"`
	CommitMessage   CommitMessageConfigV110    `yaml:"commit_message"`
	BinaryFile      BinaryFileConfig           `yaml:"binary_file"`
	Lint            LintConfig                 `yaml:"lint"`
	Encoding        EncodingConfigCurrent      `yaml:"encoding"`
	EditorConfig    EditorConfigConfig         `yaml:"editorconfig"`
	Exceptions      ExceptionsConfig           `yaml:"exceptions"`
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
	Enabled              *bool                          `yaml:"enabled"`
	NoAICoauthor         *bool                          `yaml:"no_ai_coauthor"`
	CoauthorRemoveEmails []string                       `yaml:"coauthor_remove_emails"`
	NoUnicodeSpaces      *bool                          `yaml:"no_unicode_spaces"`
	NoAmbiguousChars     *bool                          `yaml:"no_ambiguous_chars"`
	NoBadRunes           *bool                          `yaml:"no_bad_runes"`
	NoEmoji              *bool                          `yaml:"no_emoji"`
	Locale               string                         `yaml:"locale"`
	LanguageCheck        CommitMessageLanguageConfig    `yaml:"language_check"`
	ConventionalCommit   ConventionalCommitConfig       `yaml:"conventional_commit"`
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
	// 구 버전 → 신 버전 순서. v1.2.0(Current) 를 v1.1.0 보다 먼저 시도하여
	// required_language 없는 신형 설정이 v1.1.0 으로 잘못 감지되지 않게 함.
	candidates := []candidate{
		{VersionV100, tryParseStrict[ConfigV100]},
		{VersionV101, tryParseStrict[ConfigV101]},
		{VersionV102, tryParseStrict[ConfigV102]},
		{VersionCurrent, tryParseStrict[ConfigCurrent]}, // v1.2.0 (required_language 없음)
		{VersionV110, tryParseStrict[ConfigV110]},       // v1.1.0 (required_language 있음)
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
		if hasV101Fields(data) {
			return VersionV101
		}
		return VersionV100
	case VersionV102:
		// v1.0.2 config는 v1.1.0/v1.2.0 모두의 부분집합.
		//   - allowed_words 등 v1.1.0+ 신규 필드가 있는지 확인
		//   - required_language 가 있으면 v1.1.0 (마이그레이션 대상)
		//   - 그 외에는 v1.0.2 또는 v1.2.0
		hasNew := hasV110Fields(data)
		hasReqLang := hasRequiredLanguageField(data)
		switch {
		case hasNew && hasReqLang:
			return VersionV110
		case hasNew && !hasReqLang:
			return VersionCurrent
		case !hasNew && hasReqLang:
			return VersionV102 // v1.0.2 라도 required_language 가 있으면 마이그레이션 필요
		default:
			return VersionV102
		}
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

// hasRequiredLanguageField: YAML 데이터에 v1.2.0 에서 제거된 required_language 키가
// (comment_language.required_language 또는 commit_message.language_check.required_language 또는
// comment_language.file_languages[].language 형태로) 존재하는지 확인.
// 이 필드가 있으면 마이그레이션이 필요합니다 (locale 로 변환).
func hasRequiredLanguageField(data []byte) bool {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false
	}
	if cl, ok := raw["comment_language"].(map[string]any); ok {
		if _, ok := cl["required_language"]; ok {
			return true
		}
		if fl, ok := cl["file_languages"].([]any); ok {
			for _, item := range fl {
				if m, ok := item.(map[string]any); ok {
					if _, ok := m["language"]; ok {
						return true
					}
				}
			}
		}
	}
	if cm, ok := raw["commit_message"].(map[string]any); ok {
		if lc, ok := cm["language_check"].(map[string]any); ok {
			if _, ok := lc["required_language"]; ok {
				return true
			}
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
		renameRequiredLanguageRule,
	},
	VersionV101: {
		{
			Description: "commit_message.no_coauthor → commit_message.no_ai_coauthor",
			Apply: func(data []byte) []byte {
				return renameYAMLKey(data, "no_coauthor", "no_ai_coauthor")
			},
		},
		renameRequiredLanguageRule,
	},
	VersionV102: {
		renameRequiredLanguageRule,
	},
	VersionV110: {
		renameRequiredLanguageRule,
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
