package checker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/zcube/commit-checker/internal/comment"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/directive"
	"github.com/zcube/commit-checker/internal/emoji"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/langdetect"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckDiff: 스테이징된 diff에서 주석 언어 위반 사항을 검사.
// 사람이 읽을 수 있는 에러 문자열 목록을 반환 (비어 있으면 위반 없음).
func CheckDiff(cfg *config.Config) ([]string, error) {
	if !cfg.CommentLanguage.IsEnabled() {
		return nil, nil
	}

	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		return nil, err
	}

	// 유효한 확장자 목록 결정: languages가 extensions보다 우선.
	extensions := cfg.CommentLanguage.Extensions
	if len(cfg.CommentLanguage.Languages) > 0 {
		extensions = comment.ExtensionsForLanguages(cfg.CommentLanguage.Languages)
	}

	minLength := cfg.CommentLanguage.MinLength
	skipDirectives := cfg.CommentLanguage.SkipDirectives
	fullMode := cfg.CommentLanguage.IsFullMode()
	checkStrings := cfg.CommentLanguage.IsCheckStrings()
	skipTechnical := cfg.CommentLanguage.IsSkipTechnicalStrings()
	noEmoji := cfg.CommentLanguage.IsNoEmoji()
	skipPatterns := compileSkipStringPatterns(cfg.CommentLanguage.SkipStringPatterns)

	// 모든 무시 패턴 수집: 전역 + comment_language 전용 + 인라인 ignore_files.
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
			// 인덱스에 파일이 없을 수 있음 (예: 서브모듈) — 조용히 건너뜀.
			continue
		}

		comments, err := parser.ParseFile(stagedContent)
		if err != nil {
			fmt.Println(i18n.T("diff.parse_warning", map[string]interface{}{
				"Path":  diff.Path,
				"Error": err,
			}))
		}

		// 이 파일의 기본 언어 결정:
		// file_languages 규칙이 전역 required_language를 재정의.
		fileLang := resolveFileLang(diff.Path, cfg)

		// 인라인 디렉티브 적용 (commit-checker:disable / :ignore / :lang= 등)
		states := directive.Analyze(comments, fileLang)

		for i, c := range comments {
			state := states[i]
			if state.Skip {
				continue
			}
			// import/include 문자열은 항상 건너뜀
			if c.Kind == comment.KindImportString {
				continue
			}
			// 문자열 리터럴은 check_strings: true 일 때만 검사
			if c.Kind == comment.KindString && !checkStrings {
				continue
			}
			// diff 모드에서 추가된 줄에 해당하지 않는 주석은 건너뜀.
			if !fullMode && !overlapsAddedLines(c, diff.AddedLines) {
				continue
			}

			text := strings.TrimSpace(c.Text)
			// 기술적 식별자 문자열 건너뜀 (skip_technical_strings: true, 기본값)
			if c.Kind == comment.KindString && skipTechnical && IsTechnicalString(text) {
				continue
			}
			// skip_string_patterns 정규표현식 패턴 매칭으로 문자열 건너뜀
			if c.Kind == comment.KindString && len(skipPatterns) > 0 && matchesSkipPattern(text, skipPatterns) {
				continue
			}
			ok, hasContent := langdetect.IsRequiredLanguage(text, state.Language, minLength, skipDirectives)
			if !hasContent {
				continue
			}
			if !ok {
				detected := langdetect.Dominant(text)
				kindID := "diff.kind_comment"
				if c.Kind == comment.KindString {
					kindID = "diff.kind_string_literal"
				}
				errs = append(errs, i18n.T("diff.comment_language_error", map[string]interface{}{
					"Path":     diff.Path,
					"Line":     c.Line,
					"Kind":     i18n.T(kindID, nil),
					"Language": state.Language,
					"Detected": detected,
					"Text":     truncate(text, 80),
				}))
			}

			// 이모지 검사
			if noEmoji {
				emojis := emoji.FindEmojis(text)
				for _, e := range emojis {
					kindID := "diff.kind_comment"
					if c.Kind == comment.KindString {
						kindID = "diff.kind_string_literal"
					}
					errs = append(errs, i18n.T("diff.emoji_error", map[string]interface{}{
						"Path":     diff.Path,
						"Line":     c.Line + e.Line - 1,
						"Kind":     i18n.T(kindID, nil),
						"Char":     e.Char,
						"CharCode": fmt.Sprintf("%04X", e.Code),
					}))
				}
			}
		}
	}

	return errs, nil
}

// resolveFileLang: 주어진 파일 경로에 대한 필수 언어를 반환.
// file_languages 규칙을 순서대로 확인하여 첫 번째 일치하는 규칙을 적용.
// 일치하는 규칙이 없으면 전역 required_language로 폴백.
func resolveFileLang(path string, cfg *config.Config) string {
	for _, rule := range cfg.CommentLanguage.FileLanguages {
		if pathutil.MatchesAny(path, []string{rule.Pattern}) {
			return normaliseLanguage(rule.Language)
		}
	}
	return cfg.CommentLanguage.RequiredLanguage
}

// normaliseLanguage: 로케일 코드를 전체 언어 이름으로 매핑하고 소문자화.
func normaliseLanguage(lang string) string {
	if mapped := langdetect.LocaleToLanguage(strings.ToLower(lang)); mapped != "" {
		return mapped
	}
	return strings.ToLower(lang)
}

// overlapsAddedLines: 주석의 줄 중 diff에서 추가된 줄이 있는지 확인.
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

// compileSkipStringPatterns: 정규표현식 패턴 문자열을 컴파일하여 반환.
// 전체 문자열 매칭을 위해 ^...$가 없으면 자동으로 ^(?:...)$로 감쌈.
// 잘못된 패턴은 경고를 출력하고 건너뜀.
func compileSkipStringPatterns(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		// 전체 문자열 매칭: ^...$가 없으면 자동으로 감쌈
		anchored := p
		if len(p) == 0 || p[0] != '^' {
			anchored = "^(?:" + anchored
		} else {
			anchored = "(?:" + anchored[1:]
			anchored = "^" + anchored
		}
		if p[len(p)-1] != '$' {
			anchored = anchored + ")$"
		} else {
			anchored = anchored[:len(anchored)-1] + ")$"
		}
		re, err := regexp.Compile(anchored)
		if err != nil {
			fmt.Println(i18n.T("diff.parse_warning", map[string]interface{}{
				"Path":  "skip_string_patterns",
				"Error": fmt.Sprintf("invalid regex %q: %v", p, err),
			}))
			continue
		}
		compiled = append(compiled, re)
	}
	return compiled
}

// matchesSkipPattern: 문자열이 건너뜀 패턴 중 하나와 일치하는지 확인.
func matchesSkipPattern(s string, patterns []*regexp.Regexp) bool {
	for _, re := range patterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// IsTechnicalString: 언어 검사 대상에서 제외할 기술적 문자열인지 판단.
// IsPathLikeString 또는 IsAllUppercaseASCII 조건 중 하나라도 해당하면 true.
func IsTechnicalString(s string) bool {
	return IsPathLikeString(s) || IsAllUppercaseASCII(s)
}
