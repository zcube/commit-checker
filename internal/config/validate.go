package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// validLanguages: required_language / language 필드에 허용되는 값 목록.
var validLanguages = map[string]bool{
	"korean": true, "english": true, "japanese": true, "chinese": true, "any": true,
	"ko": true, "en": true, "ja": true, "zh": true,
	"zh-hans": true, "zh-hant": true,
}

// Validate: 설정 값의 유효성을 검사하고 경고 메시지 목록을 반환.
// 오류가 아닌 경고이므로 Load()가 실패하지 않음.
func Validate(cfg *Config, cfgPath string) []string {
	var warns []string

	// required_language 값 검증
	if !validLanguages[strings.ToLower(cfg.CommentLanguage.RequiredLanguage)] {
		warns = append(warns, fmt.Sprintf(
			"%s: comment_language.required_language 알 수 없는 언어: %q (korean/english/japanese/chinese/any)",
			cfgPath, cfg.CommentLanguage.RequiredLanguage,
		))
	}
	if cfg.CommitMessage.LanguageCheck.IsEnabled() {
		if !validLanguages[strings.ToLower(cfg.CommitMessage.LanguageCheck.RequiredLanguage)] {
			warns = append(warns, fmt.Sprintf(
				"%s: commit_message.language_check.required_language 알 수 없는 언어: %q",
				cfgPath, cfg.CommitMessage.LanguageCheck.RequiredLanguage,
			))
		}
	}
	for i, fl := range cfg.CommentLanguage.FileLanguages {
		if !validLanguages[strings.ToLower(fl.Language)] {
			warns = append(warns, fmt.Sprintf(
				"%s: comment_language.file_languages[%d].language 알 수 없는 언어: %q",
				cfgPath, i, fl.Language,
			))
		}
		if _, err := filepath.Match(fl.Pattern, ""); err != nil {
			warns = append(warns, fmt.Sprintf(
				"%s: comment_language.file_languages[%d].pattern 잘못된 glob 패턴: %q (%v)",
				cfgPath, i, fl.Pattern, err,
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
