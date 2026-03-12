package checker

import (
	"fmt"
	"os"
	"strings"

	"github.com/zcube/commit-checker/internal/charset"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/encoding"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckUnicode: 스테이지된 파일에서 비가시/모호한 유니코드 문자를 검사.
func CheckUnicode(cfg *config.Config) ([]string, error) {
	if !cfg.Encoding.IsEnabled() {
		return nil, nil
	}
	if !cfg.Encoding.IsNoInvisibleChars() && !cfg.Encoding.IsNoAmbiguousChars() {
		return nil, nil
	}

	files, err := getStagedFiles()
	if err != nil {
		return nil, err
	}

	return checkUnicodeFiles(cfg, files, func(path string) (string, error) {
		return gitdiff.GetStagedContent(path)
	})
}

// RunUnicode: 추적된 모든 파일에서 비가시/모호한 유니코드 문자를 검사.
func RunUnicode(cfg *config.Config) ([]string, error) {
	if !cfg.Encoding.IsEnabled() {
		return nil, nil
	}
	if !cfg.Encoding.IsNoInvisibleChars() && !cfg.Encoding.IsNoAmbiguousChars() {
		return nil, nil
	}

	files, err := getTrackedFiles()
	if err != nil {
		return nil, err
	}

	return checkUnicodeFiles(cfg, files, func(path string) (string, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	})
}

func checkUnicodeFiles(cfg *config.Config, files []string, readContent func(string) (string, error)) ([]string, error) {
	ignorePatterns := append(cfg.Exceptions.GlobalIgnore, cfg.Encoding.IgnoreFiles...)
	checkInvisible := cfg.Encoding.IsNoInvisibleChars()
	checkAmbiguous := cfg.Encoding.IsNoAmbiguousChars()

	var tables []*charset.AmbiguousTable
	if checkAmbiguous {
		tables = charset.TablesForLocale(cfg.Encoding.Locale)
	}

	var errs []string
	for _, path := range files {
		if pathutil.MatchesAny(path, ignorePatterns) {
			continue
		}

		content, err := readContent(path)
		if err != nil {
			continue
		}

		if encoding.IsBinary([]byte(content)) {
			continue
		}

		for lineNo, line := range strings.Split(content, "\n") {
			for _, r := range line {
				if checkInvisible && charset.IsInvisible(r) {
					name := charset.InvisibleName(r)
					if name == "" {
						name = fmt.Sprintf("U+%04X", r)
					}
					errs = append(errs, i18n.T("diff.file_invisible_char", map[string]any{
						"Path": path,
						"Line": lineNo + 1,
						"Char": fmt.Sprintf("U+%04X", r),
						"Name": name,
					}))
				}
				if checkAmbiguous {
					var confusedWith rune
					if charset.IsAmbiguous(r, &confusedWith, tables...) {
						errs = append(errs, i18n.T("diff.file_ambiguous_char", map[string]any{
							"Path":    path,
							"Line":    lineNo + 1,
							"Char":    fmt.Sprintf("U+%04X", r),
							"LooksAs": string(confusedWith),
						}))
					}
				}
			}
		}
	}
	return errs, nil
}
