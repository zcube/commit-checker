package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// validLanguages: locale 필드에 허용되는 값 (BCP-47 또는 legacy 언어명).
var validLanguages = map[string]bool{
	"korean": true, "english": true, "japanese": true, "chinese": true, "any": true,
	"ko": true, "en": true, "ja": true, "zh": true,
	"zh-hans": true, "zh-hant": true,
}

// checkLocale 는 값이 유효한 locale 인지 확인하고 그렇지 않으면 경고를 추가합니다.
// 빈 문자열은 무시합니다.
func checkLocale(warns []string, cfgPath, section, field, value string) []string {
	if value == "" {
		return warns
	}
	if !validLanguages[strings.ToLower(value)] {
		warns = append(warns, fmt.Sprintf(
			"%s: %s.%s 알 수 없는 값: %q (ko/en/ja/zh 또는 korean/english/japanese/chinese/any)",
			cfgPath, section, field, value,
		))
	}
	return warns
}

// Validate: 설정 값의 유효성을 검사하고 경고 메시지 목록을 반환.
// 오류가 아닌 경고이므로 Load()가 실패하지 않음.
func Validate(cfg *Config, cfgPath string) []string {
	var warns []string

	// comment_language: locale 과 (legacy) required_language 둘 다 검증
	warns = checkLocale(warns, cfgPath, "comment_language", "locale", cfg.CommentLanguage.Locale)
	warns = checkLocale(warns, cfgPath, "comment_language", "required_language", cfg.CommentLanguage.RequiredLanguage)

	if cfg.CommitMessage.LanguageCheck.IsEnabled() {
		warns = checkLocale(warns, cfgPath, "commit_message.language_check", "locale", cfg.CommitMessage.LanguageCheck.Locale)
		warns = checkLocale(warns, cfgPath, "commit_message.language_check", "required_language", cfg.CommitMessage.LanguageCheck.RequiredLanguage)
	}
	for i, fl := range cfg.CommentLanguage.FileLanguages {
		section := fmt.Sprintf("comment_language.file_languages[%d]", i)
		warns = checkLocale(warns, cfgPath, section, "locale", fl.Locale)
		warns = checkLocale(warns, cfgPath, section, "language", fl.Language)
		if _, err := filepath.Match(fl.Pattern, ""); err != nil {
			warns = append(warns, fmt.Sprintf(
				"%s: %s.pattern 잘못된 glob 패턴: %q (%v)",
				cfgPath, section, fl.Pattern, err,
			))
		}
	}

	// glob 패턴 검증 헬퍼
	checkGlobs := func(section string, patterns []string) {
		for _, p := range patterns {
			if _, err := filepath.Match(p, ""); err != nil {
				warns = append(warns, fmt.Sprintf(
					"%s: %s 잘못된 glob 패턴: %q (%v)", cfgPath, section, p, err,
				))
			}
		}
	}
	checkGlobs("comment_language.ignore_files", cfg.CommentLanguage.IgnoreFiles)
	checkGlobs("binary_file.ignore_files", cfg.BinaryFile.IgnoreFiles)
	checkGlobs("encoding.ignore_files", cfg.Encoding.IgnoreFiles)
	checkGlobs("editorconfig.ignore_files", cfg.EditorConfig.IgnoreFiles)
	checkGlobs("exceptions.global_ignore", cfg.Exceptions.GlobalIgnore)
	checkGlobs("exceptions.comment_language_ignore", cfg.Exceptions.CommentLanguageIgnore)
	checkGlobs("lint.yaml.ignore_files", cfg.Lint.YAML.IgnoreFiles)
	checkGlobs("lint.json.ignore_files", cfg.Lint.JSON.IgnoreFiles)
	checkGlobs("lint.xml.ignore_files", cfg.Lint.XML.IgnoreFiles)

	// coauthor_remove_emails glob 패턴 검증
	for _, p := range cfg.CommitMessage.CoauthorRemoveEmails {
		lowP := strings.ToLower(strings.TrimSpace(p))
		if _, err := path.Match(lowP, ""); err != nil {
			warns = append(warns, fmt.Sprintf(
				"%s: commit_message.coauthor_remove_emails 잘못된 glob 패턴: %q (%v)",
				cfgPath, p, err,
			))
		}
	}

	// allowed_words_file 존재 여부 확인
	if cfg.CommentLanguage.AllowedWordsFile != "" {
		if _, err := os.Stat(cfg.CommentLanguage.AllowedWordsFile); os.IsNotExist(err) {
			warns = append(warns, fmt.Sprintf(
				"%s: comment_language.allowed_words_file 파일을 찾을 수 없음: %q",
				cfgPath, cfg.CommentLanguage.AllowedWordsFile,
			))
		}
	}

	return warns
}
