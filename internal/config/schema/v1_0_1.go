package schema

// ConfigV101: v1.0.1 설정 스키마.
// commit_message.no_coauthor 사용, binary_file/lint/encoding/editorconfig 지원 추가.
// no_emoji 추가, type_aliases/locale 추가.
type ConfigV101 struct {
	CommentLanguage CommentLanguageConfigV101 `yaml:"comment_language"`
	CommitMessage   CommitMessageConfigV101   `yaml:"commit_message"`
	BinaryFile      BinaryFileConfig          `yaml:"binary_file"`
	Lint            LintConfig                `yaml:"lint"`
	Encoding        EncodingConfigV101        `yaml:"encoding"`
	EditorConfig    EditorConfigConfig        `yaml:"editorconfig"`
	Exceptions      ExceptionsConfig          `yaml:"exceptions"`
}

// CommentLanguageConfigV101: v1.0.1 주석 언어 검사 설정.
// no_emoji 추가, allowed_words 미지원.
type CommentLanguageConfigV101 struct {
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

// CommitMessageConfigV101: v1.0.1 커밋 메시지 검사 설정.
// no_coauthor 사용 (no_ai_coauthor 이전), enabled/no_emoji 추가.
type CommitMessageConfigV101 struct {
	Enabled              *bool                         `yaml:"enabled"`
	NoCoauthor           *bool                         `yaml:"no_coauthor"`
	CoauthorRemoveEmails []string                      `yaml:"coauthor_remove_emails"`
	NoUnicodeSpaces      *bool                         `yaml:"no_unicode_spaces"`
	NoAmbiguousChars     *bool                         `yaml:"no_ambiguous_chars"`
	NoBadRunes           *bool                         `yaml:"no_bad_runes"`
	NoEmoji              *bool                         `yaml:"no_emoji"`
	Locale               string                        `yaml:"locale"`
	LanguageCheck        CommitMessageLanguageConfig    `yaml:"language_check"`
	ConventionalCommit   ConventionalCommitConfig       `yaml:"conventional_commit"`
}

// EncodingConfigV101: v1.0.1 인코딩 설정.
// no_invisible_chars, no_ambiguous_chars, locale 미지원.
type EncodingConfigV101 struct {
	Enabled     *bool    `yaml:"enabled"`
	RequireUTF8 *bool    `yaml:"require_utf8"`
	IgnoreFiles []string `yaml:"ignore_files"`
}
