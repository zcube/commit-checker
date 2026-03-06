package config

import (
	"os"
	"path"
	"strings"

	"github.com/zcube/commit-checker/internal/langdetect"
	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration structure loaded from .commit-checker.yml
type Config struct {
	CommentLanguage CommentLanguageConfig `yaml:"comment_language"`
	CommitMessage   CommitMessageConfig   `yaml:"commit_message"`
	BinaryFile      BinaryFileConfig      `yaml:"binary_file"`
	Lint            LintConfig            `yaml:"lint"`
	Encoding        EncodingConfig        `yaml:"encoding"`
	EditorConfig    EditorConfigConfig    `yaml:"editorconfig"`
	Exceptions      ExceptionsConfig      `yaml:"exceptions"`
}

// LintConfig: 데이터 파일(YAML, JSON, XML) 구문 lint 검사 설정.
type LintConfig struct {
	// Enabled: lint 검사 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// YAML lint 설정
	YAML LintRuleConfig `yaml:"yaml"`

	// JSON lint 설정
	JSON JSONLintConfig `yaml:"json"`

	// XML lint 설정
	XML LintRuleConfig `yaml:"xml"`
}

// IsEnabled: lint 검사 활성화 여부 반환 (기본값: true).
func (c *LintConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// LintRuleConfig: 단일 lint 규칙 타입 설정.
type LintRuleConfig struct {
	// Enabled: 이 lint 규칙 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// IgnoreFiles: 건너뛸 파일의 glob 패턴 목록.
	IgnoreFiles []string `yaml:"ignore_files"`
}

// IsEnabled: 이 lint 규칙 활성화 여부 반환 (기본값: true).
func (c *LintRuleConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// JSONLintConfig: JSON lint 검사 설정.
type JSONLintConfig struct {
	// Enabled: JSON lint 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// AllowJSON5: JSON5 형식 허용 여부 (기본값: false).
	// true이면 // 및 /* */ 주석, trailing comma 허용.
	AllowJSON5 *bool `yaml:"allow_json5"`

	// IgnoreFiles: 건너뛸 파일의 glob 패턴 목록.
	// 기본 제외: package-lock.json, yarn.lock 등 auto-generated 파일.
	IgnoreFiles []string `yaml:"ignore_files"`
}

// IsEnabled: JSON lint 활성화 여부 반환 (기본값: true).
func (c *JSONLintConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsAllowJSON5: JSON5 형식 허용 여부 반환 (기본값: false).
func (c *JSONLintConfig) IsAllowJSON5() bool {
	if c.AllowJSON5 == nil {
		return false
	}
	return *c.AllowJSON5
}

// EncodingConfig: 파일 인코딩 검사 설정.
type EncodingConfig struct {
	// Enabled: 인코딩 검사 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// RequireUTF8: UTF-8 인코딩 필수 여부 (기본값: true).
	// true이면 UTF-8이 아닌 파일을 커밋할 수 없음.
	RequireUTF8 *bool `yaml:"require_utf8"`

	// IgnoreFiles: 인코딩 검사에서 제외할 파일의 glob 패턴 목록.
	IgnoreFiles []string `yaml:"ignore_files"`
}

// IsEnabled: 인코딩 검사 활성화 여부 반환 (기본값: true).
func (c *EncodingConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsRequireUTF8: UTF-8 인코딩 필수 여부 반환 (기본값: true).
func (c *EncodingConfig) IsRequireUTF8() bool {
	if c.RequireUTF8 == nil {
		return true
	}
	return *c.RequireUTF8
}

// EditorConfigConfig: .editorconfig 규칙 검증 설정.
type EditorConfigConfig struct {
	// Enabled: editorconfig 검사 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`
}

// IsEnabled: editorconfig 검사 활성화 여부 반환 (기본값: true).
func (c *EditorConfigConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// BinaryFileConfig: 스테이지된 diff에서 바이너리 파일 감지 설정.
// 컴파일된 실행파일 등 바이너리 파일이 커밋되는 것을 방지.
type BinaryFileConfig struct {
	// Enabled: 바이너리 파일 감지 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// IgnoreFiles: 허용할 바이너리 파일의 glob 패턴 목록.
	// 예: 이미지, 폰트 등 의도적으로 포함하는 바이너리 파일.
	IgnoreFiles []string `yaml:"ignore_files"`
}

// IsEnabled: 바이너리 파일 감지 활성화 여부 반환 (기본값: true).
func (c *BinaryFileConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// ExceptionsConfig: 전역 및 기능별 파일 제외 패턴 설정.
type ExceptionsConfig struct {
	// GlobalIgnore: 모든 검사에서 건너뛸 파일의 glob 패턴 목록.
	GlobalIgnore []string `yaml:"global_ignore"`

	// CommentLanguageIgnore: 주석 언어 검사에서만 건너뛸 파일의 glob 패턴 목록.
	CommentLanguageIgnore []string `yaml:"comment_language_ignore"`
}

// CommentLanguageConfig configures comment language checking in staged diffs
type CommentLanguageConfig struct {
	// Enabled controls whether comment language checking is active (default: true)
	Enabled *bool `yaml:"enabled"`

	// RequiredLanguage is the natural language comments must be written in.
	// Supported values: korean, english, japanese, chinese, any
	// Default: korean
	RequiredLanguage string `yaml:"required_language"`

	// Languages lists friendly language names to parse (e.g. go, typescript, python).
	// When set, only these parsers run; takes precedence over Extensions.
	// Supported: go, typescript, javascript, java, kotlin, python, c, cpp, csharp, swift, rust
	Languages []string `yaml:"languages"`

	// Extensions lists the file extensions to check.
	// Used when Languages is not set.
	Extensions []string `yaml:"extensions"`

	// MinLength is the minimum number of letter characters a comment must have
	// before its language is checked. Short/technical comments are skipped.
	// Default: 5
	MinLength int `yaml:"min_length"`

	// SkipDirectives lists additional comment prefixes to always skip.
	// Built-in skips include: todo, fixme, nolint, noqa, go:generate, etc.
	SkipDirectives []string `yaml:"skip_directives"`

	// CheckMode controls whether to check only added lines ("diff") or the full staged file ("full").
	// Default: diff
	CheckMode string `yaml:"check_mode"`

	// IgnoreFiles lists glob patterns for files to skip in comment language checks.
	// Equivalent to adding patterns to exceptions.comment_language_ignore.
	IgnoreFiles []string `yaml:"ignore_files"`

	// Locale is a BCP-47 locale code that sets the required language automatically.
	// Supported: ko (korean), en (english), ja (japanese), zh / zh-hans / zh-hant (chinese).
	// When set, overrides RequiredLanguage.
	Locale string `yaml:"locale"`

	// FileLanguages defines per-file language rules applied in order.
	// The first matching pattern wins; overrides the global RequiredLanguage.
	// Use language "any" to allow any language (e.g. for i18n/locale files).
	FileLanguages []FileLanguageRule `yaml:"file_languages"`

	// NoEmoji: 주석에서 이모지 사용 금지 여부 (기본값: false).
	// true이면 소스 코드 주석에 이모지 사용을 금지.
	NoEmoji *bool `yaml:"no_emoji"`

	// CheckStrings: 문자열 리터럴도 주석과 동일하게 언어 검사 여부 (기본값: false).
	// true 로 설정하면 소스 코드 내 문자열 리터럴에도 required_language 가 적용됨.
	CheckStrings *bool `yaml:"check_strings"`

	// SkipTechnicalStrings: check_strings=true 일 때 기술적 식별자로 판단되는 문자열을 건너뜀 (기본값: true).
	// 건너뜀 조건:
	//   - 슬래시(/) 포함 → 경로 또는 MIME 타입 (예: /api/v1, application/json)
	//   - 소문자 없는 순수 ASCII → 대문자 상수 (예: ERR_TOKEN, MAX_SIZE)
	// false 로 설정하면 모든 문자열을 언어 검사함.
	SkipTechnicalStrings *bool `yaml:"skip_technical_strings"`

}

// FileLanguageRule maps a glob pattern to a required language.
type FileLanguageRule struct {
	// Pattern is a glob pattern matched against the file path (supports **).
	Pattern string `yaml:"pattern"`
	// Language is the required language for files matching Pattern.
	// Accepts the same values as required_language, plus locale codes.
	Language string `yaml:"language"`
}

// IsEnabled returns true if comment language checking is enabled (default: true)
func (c *CommentLanguageConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsNoEmoji: 주석 이모지 검사 활성화 여부 반환 (기본값: false).
func (c *CommentLanguageConfig) IsNoEmoji() bool {
	if c.NoEmoji == nil {
		return false
	}
	return *c.NoEmoji
}

// IsFullMode returns true when check_mode is "full" (check all staged file comments).
func (c *CommentLanguageConfig) IsFullMode() bool {
	return c.CheckMode == "full"
}

// IsCheckStrings 는 문자열 리터럴 언어 검사가 활성화된 경우 true 반환 (기본값: false).
func (c *CommentLanguageConfig) IsCheckStrings() bool {
	if c.CheckStrings == nil {
		return false
	}
	return *c.CheckStrings
}

// IsSkipTechnicalStrings 는 기술적 식별자 문자열을 건너뛰는 경우 true 반환 (기본값: true).
func (c *CommentLanguageConfig) IsSkipTechnicalStrings() bool {
	if c.SkipTechnicalStrings == nil {
		return true
	}
	return *c.SkipTechnicalStrings
}

// CommitMessageLanguageConfig configures natural-language checking of the commit message body.
type CommitMessageLanguageConfig struct {
	// Enabled controls whether commit message language checking is active (default: false)
	Enabled *bool `yaml:"enabled"`

	// RequiredLanguage is the natural language the commit message body must be written in.
	// Supported values: korean, english, japanese, chinese, any
	// Default: korean
	RequiredLanguage string `yaml:"required_language"`

	// MinLength is the minimum number of letter characters before language is checked.
	// Default: 5
	MinLength int `yaml:"min_length"`

	// SkipPrefixes lists commit message subject prefixes that bypass language checking.
	// Default: Merge, Revert, fixup!, squash!
	SkipPrefixes []string `yaml:"skip_prefixes"`
}

// IsEnabled returns true if commit message language checking is enabled (default: false)
func (c *CommitMessageLanguageConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// ConventionalCommitConfig: Conventional Commits 형식 강제 설정.
// 명세: https://www.conventionalcommits.org/
type ConventionalCommitConfig struct {
	// Enabled: 컨벤셔널 커밋 형식 검사 활성화 여부 (기본값: false).
	Enabled *bool `yaml:"enabled"`

	// Types: 허용된 커밋 타입 목록.
	// 기본값: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert
	Types []string `yaml:"types"`

	// TypeAliases: 로컬라이즈된 타입 별칭 매핑 (로컬라이즈 -> 표준 타입).
	// 예: {"기능": "feat", "수정": "fix"} (한국어)
	// 설정하면 로컬라이즈된 타입도 허용됨.
	TypeAliases map[string]string `yaml:"type_aliases"`

	// Locale: 로컬라이즈된 타입 기본값을 적용할 언어.
	// type_aliases가 설정되지 않은 경우 내장 매핑을 사용.
	// 지원: ko, ja, zh
	Locale string `yaml:"locale"`

	// RequireScope: 스코프 필수 여부 (기본값: false).
	RequireScope *bool `yaml:"require_scope"`

	// AllowMergeCommits: "Merge "로 시작하는 커밋의 형식 검사 건너뜀 (기본값: true).
	AllowMergeCommits *bool `yaml:"allow_merge_commits"`

	// AllowRevertCommits: "Revert "로 시작하는 커밋의 형식 검사 건너뜀 (기본값: true).
	AllowRevertCommits *bool `yaml:"allow_revert_commits"`
}

// IsEnabled: 컨벤셔널 커밋 검사 활성화 여부 반환 (기본값: false).
func (c *ConventionalCommitConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// IsRequireScope: 스코프 필수 여부 반환 (기본값: false).
func (c *ConventionalCommitConfig) IsRequireScope() bool {
	if c.RequireScope == nil {
		return false
	}
	return *c.RequireScope
}

// IsAllowMergeCommits: Merge 커밋 형식 검사 건너뜀 여부 반환 (기본값: true).
func (c *ConventionalCommitConfig) IsAllowMergeCommits() bool {
	if c.AllowMergeCommits == nil {
		return true
	}
	return *c.AllowMergeCommits
}

// IsAllowRevertCommits: Revert 커밋 형식 검사 건너뜀 여부 반환 (기본값: true).
func (c *ConventionalCommitConfig) IsAllowRevertCommits() bool {
	if c.AllowRevertCommits == nil {
		return true
	}
	return *c.AllowRevertCommits
}

// DefaultConventionalTypes: 기본 허용 커밋 타입 목록.
var DefaultConventionalTypes = []string{
	"feat", "fix", "docs", "style", "refactor",
	"perf", "test", "build", "ci", "chore", "revert",
}

// LocalizedConventionalTypes: 언어별 컨벤셔널 커밋 타입 매핑.
// key: 로컬라이즈된 타입, value: 표준 타입.
var LocalizedConventionalTypes = map[string]map[string]string{
	"ko": {
		"기능":   "feat",
		"수정":   "fix",
		"문서":   "docs",
		"스타일":  "style",
		"리팩터":  "refactor",
		"리팩토링": "refactor",
		"성능":   "perf",
		"테스트":  "test",
		"빌드":   "build",
		"배포":   "ci",
		"잡일":   "chore",
		"되돌리기": "revert",
	},
	"ja": {
		"機能":       "feat",
		"修正":       "fix",
		"ドキュメント":   "docs",
		"スタイル":     "style",
		"リファクタリング": "refactor",
		"パフォーマンス":  "perf",
		"テスト":      "test",
		"ビルド":      "build",
		"デプロイ":     "ci",
		"雑務":       "chore",
		"リバート":     "revert",
	},
	"zh": {
		"功能": "feat",
		"修复": "fix",
		"文档": "docs",
		"样式": "style",
		"重构": "refactor",
		"性能": "perf",
		"测试": "test",
		"构建": "build",
		"部署": "ci",
		"杂务": "chore",
		"回退": "revert",
	},
}

// GetTypes: 설정된 타입 목록을 반환하며, 미설정 시 DefaultConventionalTypes 반환.
func (c *ConventionalCommitConfig) GetTypes() []string {
	if len(c.Types) > 0 {
		return c.Types
	}
	return DefaultConventionalTypes
}

// GetTypeAliases: 타입 별칭 매핑을 반환.
// 사용자 설정 > 로케일 내장 매핑 > 빈 map 순서로 적용.
func (c *ConventionalCommitConfig) GetTypeAliases() map[string]string {
	if len(c.TypeAliases) > 0 {
		return c.TypeAliases
	}
	if c.Locale != "" {
		if m, ok := LocalizedConventionalTypes[c.Locale]; ok {
			return m
		}
	}
	return nil
}

// GetAllAllowedTypes: 허용된 모든 타입 목록을 반환 (표준 + 별칭).
func (c *ConventionalCommitConfig) GetAllAllowedTypes() []string {
	types := c.GetTypes()
	aliases := c.GetTypeAliases()
	if len(aliases) == 0 {
		return types
	}
	all := make([]string, len(types))
	copy(all, types)
	for alias := range aliases {
		all = append(all, alias)
	}
	return all
}

// ResolveType: 타입 문자열을 표준 타입으로 해석.
// 별칭이면 표준 타입 반환, 아니면 입력 그대로 반환.
func (c *ConventionalCommitConfig) ResolveType(commitType string) string {
	aliases := c.GetTypeAliases()
	if aliases != nil {
		if standard, ok := aliases[commitType]; ok {
			return standard
		}
	}
	return commitType
}

// CommitMessageConfig configures commit message checking
type CommitMessageConfig struct {
	// NoCoauthor: AI 도구의 Co-authored-by: 트레일러를 차단 (기본값: true).
	// true이면 내장 AI 이메일 패턴 목록과 일치하는 Co-authored-by 줄을 거부/제거.
	// 일반 사람 공동 작업자는 영향을 받지 않음.
	NoCoauthor *bool `yaml:"no_coauthor"`

	// CoauthorRemoveEmails: 내장 AI 패턴에 추가로 제거할 이메일 주소 또는 glob 패턴 목록.
	// '*' 와일드카드 지원 (예: "*@myai.com"), 대소문자 무시.
	// 내장 패턴: *copilot*, noreply@anthropic.com, *@cursor.sh, *@codeium.com 등.
	CoauthorRemoveEmails []string `yaml:"coauthor_remove_emails"`

	// NoUnicodeSpaces disallows invisible/non-standard Unicode space characters (default: true)
	// Uses the same InvisibleRanges table as Gitea. BOM (U+FEFF) is excluded.
	NoUnicodeSpaces *bool `yaml:"no_unicode_spaces"`

	// NoAmbiguousChars disallows Unicode characters that look like ASCII but are not (default: true)
	// Uses the same AmbiguousCharacters tables as Gitea (VSCode unicode data source).
	// Examples: Cyrillic А (U+0410) looks like Latin A; Greek ο looks like Latin o.
	NoAmbiguousChars *bool `yaml:"no_ambiguous_chars"`

	// NoBadRunes disallows invalid UTF-8 byte sequences (default: true)
	NoBadRunes *bool `yaml:"no_bad_runes"`

	// NoEmoji: 커밋 메시지에서 이모지 사용 금지 여부 (기본값: false).
	// true이면 커밋 메시지에 이모지 사용을 금지.
	NoEmoji *bool `yaml:"no_emoji"`

	// Locale is the BCP 47 locale used for locale-specific ambiguous character detection.
	// Supported: ko, ja, zh-hans, zh-hant, ru, _default
	// Default: ko
	Locale string `yaml:"locale"`

	// LanguageCheck configures natural-language detection of the commit message body.
	LanguageCheck CommitMessageLanguageConfig `yaml:"language_check"`

	// ConventionalCommit: Conventional Commits 형식 강제 설정.
	ConventionalCommit ConventionalCommitConfig `yaml:"conventional_commit"`
}

func boolPtr(b bool) *bool { return &b }

// BuiltinAICoauthorPatterns: 내장 AI 도구 이메일 glob 패턴 목록.
// GitHub Copilot, Claude, Cursor, Codeium, Tabnine, Amazon Q 등 주요 AI 도구를 대상으로 함.
var BuiltinAICoauthorPatterns = []string{
	"*copilot*@*",
	"noreply@anthropic.com",
	"*@cursor.sh",
	"*@codeium.com",
	"*@tabnine.com",
	"*amazon-q*@*",
	"*@sourcegraph.com",
	"*gemini*@*",
}

// CoauthorShouldRemove: 주어진 이메일이 내장 AI 패턴 또는 사용자 정의 패턴과 일치하는지 확인.
// 일치하면 해당 Co-authored-by 줄을 제거해야 함을 의미.
func (c *CommitMessageConfig) CoauthorShouldRemove(email string) bool {
	emailLower := strings.ToLower(strings.TrimSpace(email))
	// 내장 AI 패턴 확인
	for _, pattern := range BuiltinAICoauthorPatterns {
		matched, err := path.Match(pattern, emailLower)
		if err == nil && matched {
			return true
		}
	}
	// 사용자 정의 추가 패턴 확인
	for _, pattern := range c.CoauthorRemoveEmails {
		matched, err := path.Match(strings.ToLower(strings.TrimSpace(pattern)), emailLower)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// ExtractCoauthorEmail: Co-authored-by: 트레일러에서 이메일 주소를 추출.
// 꺾쇠 괄호 없이 반환. 이메일이 없으면 빈 문자열 반환.
func ExtractCoauthorEmail(line string) string {
	lt := strings.LastIndex(line, "<")
	gt := strings.LastIndex(line, ">")
	if lt >= 0 && gt > lt {
		return line[lt+1 : gt]
	}
	return ""
}

// IsNoCoauthor returns true if co-author checking is enabled (default: true)
func (c *CommitMessageConfig) IsNoCoauthor() bool {
	if c.NoCoauthor == nil {
		return true
	}
	return *c.NoCoauthor
}

// IsNoUnicodeSpaces returns true if unicode space checking is enabled (default: true)
func (c *CommitMessageConfig) IsNoUnicodeSpaces() bool {
	if c.NoUnicodeSpaces == nil {
		return true
	}
	return *c.NoUnicodeSpaces
}

// IsNoAmbiguousChars returns true if ambiguous character checking is enabled (default: true)
func (c *CommitMessageConfig) IsNoAmbiguousChars() bool {
	if c.NoAmbiguousChars == nil {
		return true
	}
	return *c.NoAmbiguousChars
}

// IsNoBadRunes returns true if bad UTF-8 rune checking is enabled (default: true)
func (c *CommitMessageConfig) IsNoBadRunes() bool {
	if c.NoBadRunes == nil {
		return true
	}
	return *c.NoBadRunes
}

// IsNoEmoji: 커밋 메시지 이모지 검사 활성화 여부 반환 (기본값: false).
func (c *CommitMessageConfig) IsNoEmoji() bool {
	if c.NoEmoji == nil {
		return false
	}
	return *c.NoEmoji
}

// Load reads configuration from the given YAML file.
// If the file does not exist, default configuration is returned.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := &Config{}
			applyDefaults(cfg)
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	// locale takes priority over required_language for comment_language
	if cfg.CommentLanguage.Locale != "" {
		if lang := langdetect.LocaleToLanguage(cfg.CommentLanguage.Locale); lang != "" {
			cfg.CommentLanguage.RequiredLanguage = lang
		}
	}
	if cfg.CommentLanguage.RequiredLanguage == "" {
		cfg.CommentLanguage.RequiredLanguage = "korean"
	}
	if len(cfg.CommentLanguage.Extensions) == 0 && len(cfg.CommentLanguage.Languages) == 0 {
		cfg.CommentLanguage.Extensions = []string{
			".go", ".ts", ".tsx", ".js", ".jsx", ".mjs",
			".java", ".kt", ".py", ".c", ".cpp", ".cs", ".swift", ".rs",
		}
	}
	if cfg.CommentLanguage.MinLength == 0 {
		cfg.CommentLanguage.MinLength = 5
	}
	if cfg.CommentLanguage.CheckMode == "" {
		cfg.CommentLanguage.CheckMode = "diff"
	}
	if cfg.CommitMessage.Locale == "" {
		cfg.CommitMessage.Locale = "ko"
	}
	if cfg.CommitMessage.LanguageCheck.RequiredLanguage == "" {
		cfg.CommitMessage.LanguageCheck.RequiredLanguage = "korean"
	}
	if cfg.CommitMessage.LanguageCheck.MinLength == 0 {
		cfg.CommitMessage.LanguageCheck.MinLength = 5
	}
	if len(cfg.CommitMessage.LanguageCheck.SkipPrefixes) == 0 {
		cfg.CommitMessage.LanguageCheck.SkipPrefixes = []string{
			"Merge", "Revert", "fixup!", "squash!",
		}
	}
	_ = boolPtr // boolPtr 미사용 경고 억제
}
