// defaults.go: 로드된 설정에 기본값을 적용하고 locale 필드를 정규화하는 로직.
package config

import (
	"github.com/zcube/commit-checker/internal/langdetect"
)

func applyDefaults(cfg *Config) {
	// CommentLanguage: Locale 과 RequiredLanguage 통일 (v1.2.0+ Locale 기준).
	// 우선순위: Locale > RequiredLanguage > "korean".
	// 정규화 후 두 필드를 모두 동일한 canonical 값으로 채워 호환성 유지.
	{
		lang := langdetect.NormalizeLocale(cfg.CommentLanguage.Locale)
		if lang == "" {
			lang = langdetect.NormalizeLocale(cfg.CommentLanguage.RequiredLanguage)
		}
		if lang == "" {
			lang = langdetect.Korean
		}
		cfg.CommentLanguage.Locale = lang
		cfg.CommentLanguage.RequiredLanguage = lang
	}

	// FileLanguages: 각 항목의 Locale 과 Language 통일.
	for i := range cfg.CommentLanguage.FileLanguages {
		r := &cfg.CommentLanguage.FileLanguages[i]
		lang := langdetect.NormalizeLocale(r.Locale)
		if lang == "" {
			lang = langdetect.NormalizeLocale(r.Language)
		}
		if lang != "" {
			r.Locale = lang
			r.Language = lang
		}
	}

	if len(cfg.CommentLanguage.Extensions) == 0 && len(cfg.CommentLanguage.Languages) == 0 {
		cfg.CommentLanguage.Extensions = []string{
			".go", ".ts", ".tsx", ".js", ".jsx", ".mjs",
			".java", ".kt", ".py", ".c", ".cpp", ".cs", ".swift", ".rs",
			".hcl", ".tf", ".tfvars",
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

	// CommitMessage.LanguageCheck: Locale > RequiredLanguage > CommitMessage.Locale 에서 유도 > "korean".
	{
		lang := langdetect.NormalizeLocale(cfg.CommitMessage.LanguageCheck.Locale)
		if lang == "" {
			lang = langdetect.NormalizeLocale(cfg.CommitMessage.LanguageCheck.RequiredLanguage)
		}
		if lang == "" {
			lang = langdetect.NormalizeLocale(cfg.CommitMessage.Locale)
		}
		if lang == "" {
			lang = langdetect.Korean
		}
		cfg.CommitMessage.LanguageCheck.Locale = lang
		cfg.CommitMessage.LanguageCheck.RequiredLanguage = lang
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
