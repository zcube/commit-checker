// types.go: Config 의 하위 설정 구조체 정의와 내장 기본 데이터(타입/패턴/확장자) 목록.
package config

// CommentLanguageConfig: 스테이지된 diff의 주석 언어 검사 설정.
//
// 로케일/언어 필드 통일 정책:
//   - Locale 이 단일 진실의 소스입니다. BCP-47 코드(ko/en/ja/zh)와 legacy 언어명
//     (korean/english/japanese/chinese/any) 모두 허용됩니다.
//   - RequiredLanguage 는 v1.1.0 이전 설정과의 backward compatibility 를 위해
//     남아 있으며, Locale 이 비어있을 때만 fallback 으로 사용됩니다.
//   - applyDefaults 가 두 필드를 정규화하여 동일한 canonical 값을 갖도록 만듭니다.
type CommentLanguageConfig struct {
	// Enabled: 주석 언어 검사 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// RequiredLanguage: (legacy) 주석을 작성해야 하는 자연어. Locale 의 별칭.
	// 신규 설정에서는 Locale 사용을 권장합니다.
	RequiredLanguage string `yaml:"required_language,omitempty"`

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

	// Locale: 주석에 요구되는 자연어. BCP-47 코드 또는 legacy 언어명 모두 허용.
	// 지원값:
	//   - BCP-47:   ko, en, ja, zh (또는 zh-hans, zh-hant)
	//   - Legacy:   korean, english, japanese, chinese, any
	// 기본값: ko (korean)
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
	// Locale: Pattern 에 일치하는 파일에 필요한 자연어.
	// BCP-47 코드(ko, en, ja, zh) 또는 legacy 언어명(korean, english 등) 또는 "any".
	Locale string `yaml:"locale"`
	// Language: Locale 의 legacy 별칭 (구 설정 호환용). Locale 이 비어 있을 때만 사용됩니다.
	Language string `yaml:"language,omitempty"`
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

	// SubjectLimit: 커밋 메시지 제목(첫 번째 줄) 글자 수 제한.
	SubjectLimit SubjectLimitConfig `yaml:"subject_limit"`

	// BodyLineLimit: 커밋 메시지 본문 각 줄의 글자 수 제한.
	BodyLineLimit BodyLineLimitConfig `yaml:"body_line_limit"`
}

// CommitMessageLanguageConfig: 커밋 메시지 본문의 자연어 검사 설정.
//
// 로케일/언어 통일: Locale 이 단일 진실의 소스이며 BCP-47(ko/en/ja/zh) 와
// legacy 언어명(korean/english/...) 둘 다 허용합니다. RequiredLanguage 는
// v1.1.0 이전 설정 호환용 필드입니다.
type CommitMessageLanguageConfig struct {
	// Enabled: 커밋 메시지 언어 검사 활성화 여부 (기본값: false).
	Enabled *bool `yaml:"enabled"`

	// Locale: 커밋 메시지 본문에 요구되는 자연어 (BCP-47 또는 legacy 언어명).
	// 비어있으면 CommitMessageConfig.Locale 에서 자동 유도됩니다.
	Locale string `yaml:"locale,omitempty"`

	// RequiredLanguage: (legacy) Locale 의 별칭. 신규 설정에서는 Locale 사용을 권장.
	RequiredLanguage string `yaml:"required_language,omitempty"`

	// MinLength: 언어 검사 전 최소 글자 수.
	// 기본값: 5
	MinLength int `yaml:"min_length"`

	// SkipPrefixes: 언어 검사를 건너뛸 커밋 메시지 제목 접두사 목록.
	// 기본값: Merge, Revert, fixup!, squash!
	SkipPrefixes []string `yaml:"skip_prefixes"`
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

// SubjectLimitConfig: 커밋 메시지 제목 글자 수 제한 설정.
type SubjectLimitConfig struct {
	// Enabled: 제목 길이 검사 활성화 여부 (기본값: false).
	Enabled *bool `yaml:"enabled"`

	// MaxLength: 제목 최대 글자 수 (기본값: 72).
	MaxLength int `yaml:"max_length"`
}

// BodyLineLimitConfig: 커밋 메시지 본문 줄 길이 제한 설정.
type BodyLineLimitConfig struct {
	// Enabled: 본문 줄 길이 검사 활성화 여부 (기본값: false).
	Enabled *bool `yaml:"enabled"`

	// MaxLength: 본문 각 줄 최대 글자 수 (기본값: 100).
	MaxLength int `yaml:"max_length"`
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

// BinaryFileConfig: 스테이지된 diff에서 바이너리 파일 감지 설정.
// 컴파일된 실행파일 등 바이너리 파일이 커밋되는 것을 방지.
//
// 정책 종류 (block | allow | lfs):
//   - block: 차단 (에러)
//   - allow: 허가 (통과)
//   - lfs:   git LFS 로 추적되는 경우만 허가, 아니면 차단
//
// 우선순위: rules (확장자 매칭) > 내장 이미지 정책(이미지 확장자 → allow) > default_policy
type BinaryFileConfig struct {
	// Enabled: 바이너리 파일 감지 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// DefaultPolicy: 어느 규칙에도 매칭되지 않은 바이너리에 대한 정책 (기본값: block).
	DefaultPolicy string `yaml:"default_policy"`

	// Rules: 확장자별 정책 규칙. 첫 번째 매칭 규칙이 적용됩니다.
	Rules []BinaryFilePolicyRule `yaml:"rules"`

	// IgnoreFiles: 검사에서 완전히 제외할 파일 glob 패턴.
	IgnoreFiles []string `yaml:"ignore_files"`
}

// BinaryFilePolicyRule: 확장자별 바이너리 정책 규칙.
type BinaryFilePolicyRule struct {
	// Extensions: 정책을 적용할 파일 확장자 목록 (.png, .jpg 등). 대소문자 무관.
	Extensions []string `yaml:"extensions"`

	// Policy: 적용할 정책 (block | allow | lfs).
	Policy string `yaml:"policy"`
}

// BuiltinImageExtensions: 내장 이미지 확장자 목록.
// rules 에서 명시적으로 지정하지 않은 경우 이미지 확장자는 allow 가 기본입니다.
var BuiltinImageExtensions = []string{
	".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp",
	".ico", ".tiff", ".tif", ".heic", ".heif", ".avif",
}

// LintConfig: 데이터 파일(YAML, JSON, XML) 구문 lint 검사 설정.
type LintConfig struct {
	// Enabled: lint 검사 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// YAML lint 설정
	YAML YAMLLintConfig `yaml:"yaml"`

	// JSON lint 설정
	JSON JSONLintConfig `yaml:"json"`

	// XML lint 설정
	XML LintRuleConfig `yaml:"xml"`

	// TOML lint 설정
	TOML LintRuleConfig `yaml:"toml"`
}

// LintRuleConfig: 단일 lint 규칙 타입 설정.
type LintRuleConfig struct {
	// Enabled: 이 lint 규칙 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// IgnoreFiles: 건너뛸 파일의 glob 패턴 목록.
	IgnoreFiles []string `yaml:"ignore_files"`
}

// YAMLLintConfig: YAML lint 검사 설정.
type YAMLLintConfig struct {
	// Enabled: YAML lint 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// CommentFilter: true이면 파일 내 "# commit-checker: skip-lint" 주석으로 검사 비활성화 가능.
	CommentFilter *bool `yaml:"comment_filter"`

	// IgnoreFiles: 건너뛸 파일의 glob 패턴 목록.
	IgnoreFiles []string `yaml:"ignore_files"`
}

// JSONLintConfig: JSON lint 검사 설정.
type JSONLintConfig struct {
	// Enabled: JSON lint 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// AllowJSON5: JSON5 형식 허용 여부 (기본값: false).
	// true이면 // 및 /* */ 주석, trailing comma 허용.
	AllowJSON5 *bool `yaml:"allow_json5"`

	// CommentFilter: true이면 .json 파일에서 // 및 /* */ 주석을 제거 후 strict JSON 검사 (JSONC 모드).
	// .jsonc 파일은 설정과 무관하게 항상 JSON5로 검사.
	CommentFilter *bool `yaml:"comment_filter"`

	// IgnoreFiles: 건너뛸 파일의 glob 패턴 목록.
	// 기본 제외: package-lock.json, yarn.lock 등 auto-generated 파일.
	IgnoreFiles []string `yaml:"ignore_files"`
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

// EditorConfigConfig: .editorconfig 규칙 검증 설정.
type EditorConfigConfig struct {
	// Enabled: editorconfig 검사 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// IgnoreFiles: editorconfig 검사에서 제외할 파일의 glob 패턴 목록.
	IgnoreFiles []string `yaml:"ignore_files"`
}

// ExceptionsConfig: 전역 및 기능별 파일 제외 패턴 설정.
type ExceptionsConfig struct {
	// GlobalIgnore: 모든 검사에서 건너뛸 파일의 glob 패턴 목록.
	GlobalIgnore []string `yaml:"global_ignore"`

	// CommentLanguageIgnore: 주석 언어 검사에서만 건너뛸 파일의 glob 패턴 목록.
	CommentLanguageIgnore []string `yaml:"comment_language_ignore"`
}

// AppendOnlyConfig: 특정 경로에서 파일 삭제·내용 수정·중간 삽입을 금지하는 append-only 설정.
// DB 마이그레이션 디렉터리 등 한 번 커밋한 내용을 변경해서는 안 되는 경우에 사용합니다.
type AppendOnlyConfig struct {
	// Enabled: append-only 검사 활성화 여부 (기본값: false).
	Enabled bool `yaml:"enabled"`

	// Paths: append-only 규칙을 적용할 glob 패턴 목록.
	// 예: ["migrations/**", "db/migrations/**"]
	Paths []string `yaml:"paths"`

	// FilenameOrder: 새 파일 이름이 기존 파일보다 뒤에 와야 하는지 검사.
	// "numeric" 또는 "" (기본값): 자연수(numeric) 정렬 기준으로 기존 파일 중 최대값보다 뒤에 와야 함.
	// "none": 파일 이름 순서 검사 비활성화.
	FilenameOrder string `yaml:"filename_order"`
}

// CacheDirConfig: 빌드 산출물·캐시 디렉터리(node_modules, dist, build, target, __pycache__ 등)
// 안의 파일이 git 에 커밋되거나 스테이지되는지 검사하는 설정.
// pkg/cachedir 의 검증기를 사용하여 부모 디렉터리 인디케이터(go.mod, package.json 등) 기반으로 판별합니다.
type CacheDirConfig struct {
	// Enabled: 캐시/빌드 디렉터리 검사 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`

	// IgnoreDirs: 검사에서 제외할 디렉터리 이름 목록.
	// 예: 의도적으로 vendor/ 를 커밋하는 Go 프로젝트에서 ["vendor"] 지정.
	IgnoreDirs []string `yaml:"ignore_dirs"`
}

// GuideConfig: 검사 위반 시 출력하는 카테고리별 개선 가이드 설정.
// 위반 목록·요약 뒤에 수정 방법(구체적 명령어·행동)을 카테고리당 1회 출력합니다.
type GuideConfig struct {
	// Enabled: 개선 가이드 출력 활성화 여부 (기본값: true).
	Enabled *bool `yaml:"enabled"`
}

// CustomRulesConfig: 정규식 기반 커스텀 규칙 설정.
type CustomRulesConfig struct {
	// CommitMessage: 커밋 메시지에 적용할 커스텀 규칙 목록.
	CommitMessage []CustomRule `yaml:"commit_message"`

	// Diff: 스테이지된 diff의 추가된 줄에 적용할 커스텀 규칙 목록.
	Diff []CustomRule `yaml:"diff"`
}

// CustomRule: 정규식 기반 커스텀 검사 규칙.
type CustomRule struct {
	// Name: 규칙 이름 (오류 메시지에 표시됨).
	Name string `yaml:"name"`

	// Pattern: Go 정규식 패턴 (regexp.Compile 호환).
	Pattern string `yaml:"pattern"`

	// Message: 규칙 위반 시 표시할 사람이 읽기 쉬운 메시지.
	Message string `yaml:"message"`

	// Required: true이면 패턴이 반드시 일치해야 함 (불일치 시 오류).
	// false(기본값)이면 패턴이 일치하면 안 됨 (일치 시 오류, forbidden 규칙).
	Required bool `yaml:"required"`
}
