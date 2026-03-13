package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/zcube/commit-checker/internal/config/schema"
	"github.com/zcube/commit-checker/internal/langdetect"
	"github.com/zcube/commit-checker/internal/logger"
	"gopkg.in/yaml.v3"
)

// Config: .commit-checker.yml에서 로드하는 최상위 설정 구조체.
type Config struct {
	CommentLanguage CommentLanguageConfig `yaml:"comment_language"`
	CommitMessage   CommitMessageConfig   `yaml:"commit_message"`
	BinaryFile      BinaryFileConfig      `yaml:"binary_file"`
	Lint            LintConfig            `yaml:"lint"`
	Encoding        EncodingConfig        `yaml:"encoding"`
	EditorConfig    EditorConfigConfig    `yaml:"editorconfig"`
	Exceptions      ExceptionsConfig      `yaml:"exceptions"`
}

// CommentLanguageConfig: 스테이지된 diff의 주석 언어 검사 설정.
type CommentLanguageConfig struct {
	// Enabled: 주석 언어 검사 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// RequiredLanguage: 주석을 작성해야 하는 자연어.
	// 지원값: korean, english, japanese, chinese, any
	// 기본값: korean
	RequiredLanguage string `yaml:"required_language"`

	// Languages: 파싱할 언어 이름 목록 (예: go, typescript, python).
	// 설정 시 해당 파서만 실행되며 Extensions보다 우선함.
	// 지원: go, typescript, javascript, java, kotlin, python, c, cpp, csharp, swift, rust
	Languages []string `yaml:"languages"`

	// Extensions: 검사할 파일 확장자 목록.
	// Languages가 설정되지 않은 경우 사용.
	Extensions []string `yaml:"extensions"`

	// MinLength: 언어 검사 전 주석이 가져야 할 최소 글자 수.
	// 짧거나 기술적인 주석은 건너뜀.
	// 기본값: 5
	MinLength int `yaml:"min_length"`

	// SkipDirectives: 항상 건너뛸 추가 주석 접두사 목록.
	// 내장 건너뜀: todo, fixme, nolint, noqa, go:generate 등.
	SkipDirectives []string `yaml:"skip_directives"`

	// CheckMode: 추가된 줄만("diff") 또는 전체 스테이지 파일("full")을 검사할지 제어.
	// 기본값: diff
	CheckMode string `yaml:"check_mode"`

	// IgnoreFiles: 주석 언어 검사에서 건너뛸 파일의 glob 패턴 목록.
	// exceptions.comment_language_ignore에 패턴을 추가하는 것과 동일.
	IgnoreFiles []string `yaml:"ignore_files"`

	// Locale: 필수 언어를 자동으로 설정하는 BCP-47 로케일 코드.
	// 지원: ko (korean), en (english), ja (japanese), zh / zh-hans / zh-hant (chinese).
	// 설정 시 RequiredLanguage를 재정의함.
	Locale string `yaml:"locale"`

	// FileLanguages: 순서대로 적용되는 파일별 언어 규칙.
	// 첫 번째 일치 패턴이 적용되며 전역 RequiredLanguage를 재정의함.
	// 모든 언어를 허용하려면 language를 "any"로 설정 (예: i18n/locale 파일).
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

	// AllowedWords: 언어 검사에서 무시할 영어 단어 목록.
	// 주석에서 해당 단어를 제거한 후 나머지 텍스트로 언어를 판별합니다.
	// 고유명사, 기술 용어 등에 활용합니다.
	// 예: TypeScript, JavaScript, API, URL
	AllowedWords []string `yaml:"allowed_words"`

	// AllowedWordsFile: 허용 단어가 한 줄에 하나씩 적힌 텍스트 파일 경로.
	// allowed_words와 병합됩니다. 줄 단위로 읽으며 # 로 시작하는 줄은 주석으로 무시합니다.
	AllowedWordsFile string `yaml:"allowed_words_file"`

	// AllowedWordsURL: 허용 단어 파일을 HTTP/HTTPS로 가져올 URL.
	// allowed_words, allowed_words_file과 병합됩니다.
	// 형식은 allowed_words_file과 동일합니다 (줄 단위, # 주석).
	AllowedWordsURL string `yaml:"allowed_words_url"`

	// AllowedWordsCache: URL에서 가져온 허용 단어의 로컬 캐싱 설정.
	AllowedWordsCache AllowedWordsCacheConfig `yaml:"allowed_words_cache"`
}

// FileLanguageRule: glob 패턴을 필수 언어에 매핑하는 규칙.
type FileLanguageRule struct {
	// Pattern: 파일 경로에 대해 매칭되는 glob 패턴 (** 와일드카드 지원).
	Pattern string `yaml:"pattern"`
	// Language: Pattern에 일치하는 파일에 필요한 언어.
	// required_language와 동일한 값 및 로케일 코드 허용.
	Language string `yaml:"language"`
}

// IsEnabled: 주석 언어 검사 활성화 여부 반환 (기본값: true).
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

// IsFullMode: check_mode가 "full"일 때 true 반환 (전체 스테이지 파일 주석 검사).
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

// CommitMessageConfig: 커밋 메시지 검사 설정.
type CommitMessageConfig struct {
	// Enabled: 커밋 메시지 검사 전체 활성화 여부 (기본값: true).
	// false이면 no_ai_coauthor, no_unicode_spaces 등 모든 하위 검사를 건너뜀.
	Enabled *bool `yaml:"enabled"`

	// NoAICoauthor: AI 도구의 Co-authored-by: 트레일러를 차단 (기본값: true).
	// true이면 내장 AI 이메일 패턴 목록과 일치하는 Co-authored-by 줄을 거부/제거.
	// 일반 사람 공동 작업자는 영향을 받지 않음.
	NoAICoauthor *bool `yaml:"no_ai_coauthor"`

	// CoauthorRemoveEmails: 내장 AI 패턴에 추가로 제거할 이메일 주소 또는 glob 패턴 목록.
	// '*' 와일드카드 지원 (예: "*@myai.com"), 대소문자 무시.
	// 내장 패턴: *copilot*, noreply@anthropic.com, *@cursor.sh, *@codeium.com 등.
	CoauthorRemoveEmails []string `yaml:"coauthor_remove_emails"`

	// NoUnicodeSpaces: 보이지 않는/비표준 유니코드 공백 문자 금지 여부 (기본값: true).
	// Gitea와 동일한 InvisibleRanges 테이블 사용. BOM (U+FEFF)은 제외.
	NoUnicodeSpaces *bool `yaml:"no_unicode_spaces"`

	// NoAmbiguousChars: ASCII처럼 보이지만 아닌 유니코드 문자 금지 여부 (기본값: true).
	// Gitea와 동일한 AmbiguousCharacters 테이블 사용 (VSCode 유니코드 데이터 소스).
	// 예: 키릴 А (U+0410)는 라틴 A처럼; 그리스 ο는 라틴 o처럼 보임.
	NoAmbiguousChars *bool `yaml:"no_ambiguous_chars"`

	// NoBadRunes: 잘못된 UTF-8 바이트 시퀀스 금지 여부 (기본값: true).
	NoBadRunes *bool `yaml:"no_bad_runes"`

	// NoEmoji: 커밋 메시지에서 이모지 사용 금지 여부 (기본값: false).
	// true이면 커밋 메시지에 이모지 사용을 금지.
	NoEmoji *bool `yaml:"no_emoji"`

	// Locale: 로케일별 모호 문자 감지에 사용되는 BCP-47 로케일 코드.
	// 지원: ko, ja, zh-hans, zh-hant, ru, _default
	// 기본값: ko
	Locale string `yaml:"locale"`

	// LanguageCheck: 커밋 메시지 본문의 자연어 감지 설정.
	LanguageCheck CommitMessageLanguageConfig `yaml:"language_check"`

	// ConventionalCommit: Conventional Commits 형식 강제 설정.
	ConventionalCommit ConventionalCommitConfig `yaml:"conventional_commit"`
}

// CommitMessageLanguageConfig: 커밋 메시지 본문의 자연어 검사 설정.
type CommitMessageLanguageConfig struct {
	// Enabled: 커밋 메시지 언어 검사 활성화 여부 (기본값: false).
	Enabled *bool `yaml:"enabled"`

	// RequiredLanguage: 커밋 메시지 본문을 작성해야 하는 자연어.
	// 지원값: korean, english, japanese, chinese, any
	// 기본값: korean
	RequiredLanguage string `yaml:"required_language"`

	// MinLength: 언어 검사 전 최소 글자 수.
	// 기본값: 5
	MinLength int `yaml:"min_length"`

	// SkipPrefixes: 언어 검사를 건너뛸 커밋 메시지 제목 접두사 목록.
	// 기본값: Merge, Revert, fixup!, squash!
	SkipPrefixes []string `yaml:"skip_prefixes"`
}

// IsEnabled: 커밋 메시지 언어 검사 활성화 여부 반환 (기본값: false).
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

// IsEnabled: 커밋 메시지 검사 전체 활성화 여부 반환 (기본값: true).
func (c *CommitMessageConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsNoAICoauthor: 공동 작성자 검사 활성화 여부 반환 (기본값: true).
func (c *CommitMessageConfig) IsNoAICoauthor() bool {
	if c.NoAICoauthor == nil {
		return true
	}
	return *c.NoAICoauthor
}

// IsNoUnicodeSpaces: 유니코드 공백 검사 활성화 여부 반환 (기본값: true).
func (c *CommitMessageConfig) IsNoUnicodeSpaces() bool {
	if c.NoUnicodeSpaces == nil {
		return true
	}
	return *c.NoUnicodeSpaces
}

// IsNoAmbiguousChars: 모호한 문자 검사 활성화 여부 반환 (기본값: true).
func (c *CommitMessageConfig) IsNoAmbiguousChars() bool {
	if c.NoAmbiguousChars == nil {
		return true
	}
	return *c.NoAmbiguousChars
}

// IsNoBadRunes: 잘못된 UTF-8 룬 검사 활성화 여부 반환 (기본값: true).
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
		lowPattern := strings.ToLower(strings.TrimSpace(pattern))
		matched, err := path.Match(lowPattern, emailLower)
		if err != nil {
			logger.Warn("invalid glob pattern in coauthor_remove_emails",
				"pattern", pattern, "error", err)
			continue
		}
		if matched {
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

	// NoInvisibleChars: 파일 내용에서 비가시 유니코드 문자 금지 여부 (기본값: false).
	// NBSP, ZWSP, BiDi 제어 문자 등을 감지.
	NoInvisibleChars *bool `yaml:"no_invisible_chars"`

	// NoAmbiguousChars: 파일 내용에서 ASCII와 혼동되는 유니코드 문자 금지 여부 (기본값: false).
	// 키릴 А (U+0410) vs 라틴 A (U+0041) 등을 감지.
	NoAmbiguousChars *bool `yaml:"no_ambiguous_chars"`

	// Locale: 모호한 문자 감지에 사용할 BCP-47 로케일 코드 (기본값: ko).
	Locale string `yaml:"locale"`

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

// IsNoInvisibleChars: 비가시 유니코드 문자 검사 활성화 여부 반환 (기본값: false).
func (c *EncodingConfig) IsNoInvisibleChars() bool {
	if c.NoInvisibleChars == nil {
		return false
	}
	return *c.NoInvisibleChars
}

// IsNoAmbiguousChars: 모호한 유니코드 문자 검사 활성화 여부 반환 (기본값: false).
func (c *EncodingConfig) IsNoAmbiguousChars() bool {
	if c.NoAmbiguousChars == nil {
		return false
	}
	return *c.NoAmbiguousChars
}

// EditorConfigConfig: .editorconfig 규칙 검증 설정.
type EditorConfigConfig struct {
	// Enabled: editorconfig 검사 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// IgnoreFiles: editorconfig 검사에서 제외할 파일의 glob 패턴 목록.
	IgnoreFiles []string `yaml:"ignore_files"`
}

// IsEnabled: editorconfig 검사 활성화 여부 반환 (기본값: true).
func (c *EditorConfigConfig) IsEnabled() bool {
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

// Load: 주어진 YAML 파일에서 설정을 읽음.
// 파일이 없으면 기본 설정을 반환.
func Load(cfgPath string) (*Config, error) {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := &Config{}
			applyDefaults(cfg)
			return cfg, nil
		}
		return nil, err
	}

	// 구 버전 스키마 감지: 현재 스키마로 파싱 실패 시 자동 마이그레이션 시도.
	ver := schema.DetectVersion(data)
	if ver != schema.VersionCurrent && ver != schema.VersionUnknown {
		result, migErr := schema.Migrate(data)
		if migErr == nil {
			data = result.Data
		} else {
			logger.Warn("config auto-migration failed, proceeding with original",
				"path", cfgPath, "error", migErr)
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, formatConfigError(cfgPath, err)
	}

	applyDefaults(&cfg)
	if err := resolveAllowedWords(&cfg); err != nil {
		return nil, err
	}
	for _, w := range Validate(&cfg, cfgPath) {
		logger.Warn(w)
	}
	return &cfg, nil
}

// resolveAllowedWords 는 allowed_words_file, allowed_words_url 에서 단어를 읽어
// allowed_words 목록에 병합합니다.
func resolveAllowedWords(cfg *Config) error {
	if cfg.CommentLanguage.AllowedWordsFile != "" {
		filePath := cfg.CommentLanguage.AllowedWordsFile
		// ~ 경로 확장
		if strings.HasPrefix(filePath, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				filePath = filepath.Join(home, filePath[2:])
				cfg.CommentLanguage.AllowedWordsFile = filePath
			}
		}
		words, err := loadWordsFromFile(filePath)
		if err != nil {
			return fmt.Errorf("allowed_words_file 읽기 실패: %w", err)
		}
		cfg.CommentLanguage.AllowedWords = append(cfg.CommentLanguage.AllowedWords, words...)
	}
	if cfg.CommentLanguage.AllowedWordsURL != "" {
		cache := &cfg.CommentLanguage.AllowedWordsCache
		if words, ok := loadCachedWords(cache, cfg.CommentLanguage.AllowedWordsURL); ok {
			cfg.CommentLanguage.AllowedWords = append(cfg.CommentLanguage.AllowedWords, words...)
		} else {
			words, body, err := loadWordsFromURLWithBody(cfg.CommentLanguage.AllowedWordsURL)
			if err != nil {
				return fmt.Errorf("allowed_words_url 가져오기 실패: %w", err)
			}
			cfg.CommentLanguage.AllowedWords = append(cfg.CommentLanguage.AllowedWords, words...)
			saveCachedWords(cache, cfg.CommentLanguage.AllowedWordsURL, body)
		}
	}
	return nil
}

// parseWordLines 는 텍스트를 줄 단위로 분리하여 단어 목록을 반환합니다.
// '#' 로 시작하는 줄과 빈 줄은 무시합니다.
func parseWordLines(text string) []string {
	var words []string
	for _, line := range strings.Split(text, "\n") {
		word := strings.TrimSpace(line)
		if word == "" || strings.HasPrefix(word, "#") {
			continue
		}
		words = append(words, word)
	}
	return words
}

func loadWordsFromFile(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return parseWordLines(string(data)), nil
}

// maxAllowedWordsSize: URL에서 가져오는 허용 단어 파일의 최대 크기 (10MB).
const maxAllowedWordsSize = 10 * 1024 * 1024

func loadWordsFromURLWithBody(rawURL string) ([]string, []byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(rawURL) //nolint:gosec
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, maxAllowedWordsSize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, nil, err
	}
	if int64(len(body)) > maxAllowedWordsSize {
		return nil, nil, fmt.Errorf("allowed_words_url response exceeds 10MB limit")
	}
	return parseWordLines(string(body)), body, nil
}

func applyDefaults(cfg *Config) {
	// locale이 comment_language의 required_language보다 우선함
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
			"dockerfile",
		}
	}
	if cfg.CommentLanguage.MinLength == 0 {
		cfg.CommentLanguage.MinLength = 5
	}
	if cfg.CommentLanguage.CheckMode == "" {
		cfg.CommentLanguage.CheckMode = "diff"
	}
	if cfg.Encoding.Locale == "" {
		cfg.Encoding.Locale = "ko"
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
}
