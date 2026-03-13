package schema

// ConfigV100: v1.0.0 설정 스키마.
// commit_message.no_coauthor 사용, binary_file/lint/encoding/editorconfig 미지원.
type ConfigV100 struct {
	CommentLanguage CommentLanguageConfigV100 `yaml:"comment_language"`
	CommitMessage   CommitMessageConfigV100   `yaml:"commit_message"`
	Exceptions      ExceptionsConfig          `yaml:"exceptions"`
}

// CommentLanguageConfigV100: v1.0.0 주석 언어 검사 설정.
// no_emoji, allowed_words 미지원.
type CommentLanguageConfigV100 struct {
	Enabled              *bool               `yaml:"enabled"`
	RequiredLanguage     string              `yaml:"required_language"`
	Languages            []string            `yaml:"languages"`
	Extensions           []string            `yaml:"extensions"`
	MinLength            int                 `yaml:"min_length"`
	SkipDirectives       []string            `yaml:"skip_directives"`
	CheckMode            string              `yaml:"check_mode"`
	IgnoreFiles          []string            `yaml:"ignore_files"`
	Locale               string              `yaml:"locale"`
	FileLanguages        []FileLanguageRule  `yaml:"file_languages"`
	CheckStrings         *bool               `yaml:"check_strings"`
	SkipTechnicalStrings *bool               `yaml:"skip_technical_strings"`
}

// CommitMessageConfigV100: v1.0.0 커밋 메시지 검사 설정.
// no_coauthor 사용 (no_ai_coauthor 이전), enabled/no_emoji 미지원.
type CommitMessageConfigV100 struct {
	NoCoauthor           *bool                            `yaml:"no_coauthor"`
	CoauthorRemoveEmails []string                         `yaml:"coauthor_remove_emails"`
	NoUnicodeSpaces      *bool                            `yaml:"no_unicode_spaces"`
	NoAmbiguousChars     *bool                            `yaml:"no_ambiguous_chars"`
	NoBadRunes           *bool                            `yaml:"no_bad_runes"`
	Locale               string                           `yaml:"locale"`
	LanguageCheck        CommitMessageLanguageConfig       `yaml:"language_check"`
	ConventionalCommit   ConventionalCommitConfigV100      `yaml:"conventional_commit"`
}

// ConventionalCommitConfigV100: v1.0.0 컨벤셔널 커밋 설정.
// type_aliases, locale 미지원.
type ConventionalCommitConfigV100 struct {
	Enabled            *bool    `yaml:"enabled"`
	Types              []string `yaml:"types"`
	RequireScope       *bool    `yaml:"require_scope"`
	AllowMergeCommits  *bool    `yaml:"allow_merge_commits"`
	AllowRevertCommits *bool    `yaml:"allow_revert_commits"`
}
