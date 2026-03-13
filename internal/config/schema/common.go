package schema

// 버전 간 공유되는 공통 타입 정의.
// 이 타입들은 모든 버전에서 동일한 구조를 가짐.

// FileLanguageRule: 파일별 언어 규칙 (glob 패턴 → 언어 매핑).
type FileLanguageRule struct {
	Pattern  string `yaml:"pattern"`
	Language string `yaml:"language"`
}

// ExceptionsConfig: 전역 및 기능별 파일 제외 패턴.
type ExceptionsConfig struct {
	GlobalIgnore          []string `yaml:"global_ignore"`
	CommentLanguageIgnore []string `yaml:"comment_language_ignore"`
}

// CommitMessageLanguageConfig: 커밋 메시지 본문 자연어 검사 설정.
type CommitMessageLanguageConfig struct {
	Enabled          *bool    `yaml:"enabled"`
	RequiredLanguage string   `yaml:"required_language"`
	MinLength        int      `yaml:"min_length"`
	SkipPrefixes     []string `yaml:"skip_prefixes"`
}

// ConventionalCommitConfig: 컨벤셔널 커밋 형식 설정 (v1.0.1+).
type ConventionalCommitConfig struct {
	Enabled            *bool             `yaml:"enabled"`
	Types              []string          `yaml:"types"`
	TypeAliases        map[string]string `yaml:"type_aliases"`
	Locale             string            `yaml:"locale"`
	RequireScope       *bool             `yaml:"require_scope"`
	AllowMergeCommits  *bool             `yaml:"allow_merge_commits"`
	AllowRevertCommits *bool             `yaml:"allow_revert_commits"`
}

// BinaryFileConfig: 바이너리 파일 감지 설정 (v1.0.1+).
type BinaryFileConfig struct {
	Enabled     *bool    `yaml:"enabled"`
	IgnoreFiles []string `yaml:"ignore_files"`
}

// LintConfig: 데이터 파일 lint 설정 (v1.0.1+).
type LintConfig struct {
	Enabled *bool          `yaml:"enabled"`
	YAML    LintRuleConfig `yaml:"yaml"`
	JSON    JSONLintConfig `yaml:"json"`
	XML     LintRuleConfig `yaml:"xml"`
}

// LintRuleConfig: 단일 lint 규칙 설정.
type LintRuleConfig struct {
	Enabled     *bool    `yaml:"enabled"`
	IgnoreFiles []string `yaml:"ignore_files"`
}

// JSONLintConfig: JSON lint 설정.
type JSONLintConfig struct {
	Enabled     *bool    `yaml:"enabled"`
	AllowJSON5  *bool    `yaml:"allow_json5"`
	IgnoreFiles []string `yaml:"ignore_files"`
}

// EditorConfigConfig: EditorConfig 검사 설정 (v1.0.1+).
type EditorConfigConfig struct {
	Enabled     *bool    `yaml:"enabled"`
	IgnoreFiles []string `yaml:"ignore_files"`
}
