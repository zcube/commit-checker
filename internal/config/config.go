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
	Exceptions      ExceptionsConfig      `yaml:"exceptions"`
}

// ExceptionsConfig defines global and per-feature file exclusion patterns.
type ExceptionsConfig struct {
	// GlobalIgnore lists glob patterns for files to skip in ALL checks.
	GlobalIgnore []string `yaml:"global_ignore"`

	// CommentLanguageIgnore lists glob patterns for files to skip only in comment language checks.
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

	// Locale is the BCP 47 locale used for locale-specific ambiguous character detection.
	// Supported: ko, ja, zh-hans, zh-hant, ru, _default
	// Default: ko
	Locale string `yaml:"locale"`

	// LanguageCheck configures natural-language detection of the commit message body.
	LanguageCheck CommitMessageLanguageConfig `yaml:"language_check"`
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
	_ = boolPtr // suppress unused warning
}
