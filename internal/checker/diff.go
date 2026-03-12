package checker

import (
	"fmt"
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

// CheckDiff 는 스테이지된 diff 에서 주석 언어 위반을 검사합니다.
// 사람이 읽을 수 있는 오류 문자열 목록을 반환합니다 (빈 목록 = 위반 없음).
func CheckDiff(cfg *config.Config) ([]string, error) {
	if !cfg.CommentLanguage.IsEnabled() {
		return nil, nil
	}

	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		return nil, err
	}

	// 유효 확장자 목록 결정: languages 가 extensions 보다 우선합니다.
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
	allowedWords := cfg.CommentLanguage.AllowedWords

	// 무시 패턴 수집: 전역 + comment_language 전용 + 인라인 ignore_files.
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
			// 파일이 인덱스에 없을 수 있음 (예: 서브모듈) — 조용히 건너뜁니다.
			continue
		}

		comments, err := parser.ParseFile(stagedContent)
		if err != nil {
			fmt.Println(i18n.T("diff.parse_warning", map[string]any{
				"Path":  diff.Path,
				"Error": err,
			}))
		}

		// 파일별 기본 언어 결정:
		// file_languages 규칙이 전역 required_language 를 재정의합니다.
		fileLang := resolveFileLang(diff.Path, cfg)

		// 인라인 지시자 처리 (commit-checker:disable / :ignore / :lang= 등).
		states := directive.Analyze(comments, fileLang)

		for _, u := range buildCommentUnits(comments, states, checkStrings) {
			// diff 모드에서는 해당 단위의 줄 범위가 추가된 줄과 겹쳐야 합니다.
			if !fullMode {
				overlaps := false
				for ln := u.line; ln <= u.endLine; ln++ {
					if diff.AddedLines[ln] {
						overlaps = true
						break
					}
				}
				if !overlaps {
					continue
				}
			}

			text := langdetect.StripAllowedWords(u.text, allowedWords)
			if u.kind == comment.KindString && skipTechnical && IsTechnicalString(text) {
				continue
			}
			ok, hasContent := langdetect.IsRequiredLanguage(text, u.lang, minLength, skipDirectives)
			if !hasContent {
				continue
			}
			if !ok {
				detected := langdetect.Dominant(text)
				kindID := "diff.kind_comment"
				if u.kind == comment.KindString {
					kindID = "diff.kind_string_literal"
				}
				errs = append(errs, i18n.T("diff.comment_language_error", map[string]any{
					"Path":     diff.Path,
					"Line":     u.line,
					"Kind":     i18n.T(kindID, nil),
					"Language": u.lang,
					"Detected": detected,
					"Text":     truncate(text, 80),
				}))
			}

			// 이모지 검사
			if noEmoji {
				emojis := emoji.FindEmojis(text)
				for _, e := range emojis {
					kindID := "diff.kind_comment"
					if u.kind == comment.KindString {
						kindID = "diff.kind_string_literal"
					}
					errs = append(errs, i18n.T("diff.emoji_error", map[string]any{
						"Path":     diff.Path,
						"Line":     u.line + e.Line - 1,
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

// resolveFileLang 는 file_languages 규칙을 순서대로 확인하여 파일 경로에 대한 필수 언어를 반환합니다.
// 첫 번째로 일치하는 규칙이 적용되며, 없으면 전역 required_language 로 폴백합니다.
func resolveFileLang(path string, cfg *config.Config) string {
	for _, rule := range cfg.CommentLanguage.FileLanguages {
		if pathutil.MatchesAny(path, []string{rule.Pattern}) {
			return normaliseLanguage(rule.Language)
		}
	}
	return cfg.CommentLanguage.RequiredLanguage
}

// normaliseLanguage 는 로케일 코드를 전체 언어 이름으로 매핑하고 소문자로 변환합니다.
func normaliseLanguage(lang string) string {
	if mapped := langdetect.LocaleToLanguage(strings.ToLower(lang)); mapped != "" {
		return mapped
	}
	return strings.ToLower(lang)
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
