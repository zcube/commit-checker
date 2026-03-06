package checker

import (
	"fmt"
	"strings"

	"github.com/zcube/commit-checker/internal/comment"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/directive"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/langdetect"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckDiff inspects the staged diff for comment language violations.
// It returns a list of human-readable error strings (empty = no violations).
func CheckDiff(cfg *config.Config) ([]string, error) {
	if !cfg.CommentLanguage.IsEnabled() {
		return nil, nil
	}

	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		return nil, err
	}

	// Resolve the effective extension list: languages takes priority over extensions.
	extensions := cfg.CommentLanguage.Extensions
	if len(cfg.CommentLanguage.Languages) > 0 {
		extensions = comment.ExtensionsForLanguages(cfg.CommentLanguage.Languages)
	}

	minLength := cfg.CommentLanguage.MinLength
	skipDirectives := cfg.CommentLanguage.SkipDirectives
	fullMode := cfg.CommentLanguage.IsFullMode()
	checkStrings := cfg.CommentLanguage.IsCheckStrings()
	skipTechnical := cfg.CommentLanguage.IsSkipTechnicalStrings()

	// Collect all ignore patterns: global + comment_language specific + inline ignore_files.
	ignorePatterns := append(cfg.Exceptions.GlobalIgnore,
		cfg.Exceptions.CommentLanguageIgnore...)
	ignorePatterns = append(ignorePatterns, cfg.CommentLanguage.IgnoreFiles...)

	var errs []string

	for _, diff := range diffs {
		if diff.IsDeleted {
			continue
		}
		if !fullMode && len(diff.AddedLines) == 0 {
			continue
		}
		if !gitdiff.HasExtension(diff.Path, extensions) {
			continue
		}
		if pathutil.MatchesAny(diff.Path, ignorePatterns) {
			continue
		}

		parser := comment.GetParser(diff.Path)
		if parser == nil {
			continue
		}

		stagedContent, err := gitdiff.GetStagedContent(diff.Path)
		if err != nil {
			// File may be absent from index (e.g. submodule) — skip silently.
			continue
		}

		comments, err := parser.ParseFile(stagedContent)
		if err != nil {
			fmt.Printf("warning: could not fully parse %s: %v\n", diff.Path, err)
		}

		// Resolve the base language for this file:
		// file_languages rules override the global required_language.
		fileLang := resolveFileLang(diff.Path, cfg)

		// Apply inline directives (commit-checker:disable / :ignore / :lang= etc.)
		states := directive.Analyze(comments, fileLang)

		for i, c := range comments {
			state := states[i]
			if state.Skip {
				continue
			}
			// 문자열 리터럴은 check_strings: true 일 때만 검사
			if c.Kind == comment.KindString && !checkStrings {
				continue
			}
			// In diff mode, skip comments that don't touch any added line.
			if !fullMode && !overlapsAddedLines(c, diff.AddedLines) {
				continue
			}

			text := strings.TrimSpace(c.Text)
			// 기술적 식별자 문자열 건너뜀 (skip_technical_strings: true, 기본값)
			if c.Kind == comment.KindString && skipTechnical && IsTechnicalString(text) {
				continue
			}
			ok, hasContent := langdetect.IsRequiredLanguage(text, state.Language, minLength, skipDirectives)
			if !hasContent {
				continue
			}
			if !ok {
				detected := langdetect.Dominant(text)
				kind := "comment"
				if c.Kind == comment.KindString {
					kind = "string literal"
				}
				errs = append(errs, fmt.Sprintf(
					"%s:%d: %s must be written in %s (detected: %s): %s",
					diff.Path, c.Line, kind, state.Language, detected,
					truncate(text, 80),
				))
			}
		}
	}

	return errs, nil
}

// resolveFileLang returns the required language for a given file path by
// checking file_languages rules in order. The first matching rule wins.
// Falls back to the global required_language.
func resolveFileLang(path string, cfg *config.Config) string {
	for _, rule := range cfg.CommentLanguage.FileLanguages {
		if pathutil.MatchesAny(path, []string{rule.Pattern}) {
			return normaliseLanguage(rule.Language)
		}
	}
	return cfg.CommentLanguage.RequiredLanguage
}

// normaliseLanguage maps locale codes to full language names and lowercases.
func normaliseLanguage(lang string) string {
	if mapped := langdetect.LocaleToLanguage(strings.ToLower(lang)); mapped != "" {
		return mapped
	}
	return strings.ToLower(lang)
}

// overlapsAddedLines reports whether any line of the comment was added in the diff.
func overlapsAddedLines(c comment.Comment, addedLines map[int]bool) bool {
	for line := c.Line; line <= c.EndLine; line++ {
		if addedLines[line] {
			return true
		}
	}
	return false
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

// IsPathLikeString: 슬래시(/) 포함 문자열은 경로 또는 MIME 타입으로 판단.
// 예: "/api/v1/users", "application/json", "text/html; charset=utf-8"
func IsPathLikeString(s string) bool {
	return strings.ContainsRune(s, '/')
}

// IsAllUppercaseASCII: 소문자와 비ASCII 문자 없는 순수 대문자 ASCII 문자열은 상수 식별자로 판단.
// 예: "ERR_TOKEN", "MAX_RETRY_COUNT", "STATUS_OK"
func IsAllUppercaseASCII(s string) bool {
	for _, r := range s {
		if r > 0x7F {
			return false // 비ASCII 문자 포함
		}
		if r >= 'a' && r <= 'z' {
			return false // 소문자 포함
		}
	}
	return true
}

// IsTechnicalString: 언어 검사 대상에서 제외할 기술적 문자열인지 판단.
// IsPathLikeString 또는 IsAllUppercaseASCII 조건 중 하나라도 해당하면 true.
func IsTechnicalString(s string) bool {
	return IsPathLikeString(s) || IsAllUppercaseASCII(s)
}
