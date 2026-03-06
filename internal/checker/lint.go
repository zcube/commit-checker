package checker

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/lint"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckLint: 스테이지된 데이터 파일(YAML, JSON, XML)의 구문 오류를 검사.
func CheckLint(cfg *config.Config) ([]string, error) {
	if !cfg.Lint.IsEnabled() {
		return nil, nil
	}

	files, err := getStagedFiles()
	if err != nil {
		return nil, err
	}

	globalIgnore := cfg.Exceptions.GlobalIgnore
	var errs []string

	for _, path := range files {
		if pathutil.MatchesAny(path, globalIgnore) {
			continue
		}

		ext := strings.ToLower(filepath.Ext(path))
		var validationErrs []lint.ValidationError

		switch ext {
		case ".yaml", ".yml":
			if !cfg.Lint.YAML.IsEnabled() {
				continue
			}
			if pathutil.MatchesAny(path, cfg.Lint.YAML.IgnoreFiles) {
				continue
			}
			content, err := gitdiff.GetStagedContent(path)
			if err != nil {
				continue
			}
			validationErrs = lint.ValidateYAML(path, content)

		case ".json":
			if !cfg.Lint.JSON.IsEnabled() {
				continue
			}
			ignoreFiles := cfg.Lint.JSON.IgnoreFiles
			// 기본 제외 파일은 사용자 설정이 없을 때만 적용
			if len(ignoreFiles) == 0 {
				ignoreFiles = lint.DefaultJSONIgnoreFiles
			}
			if pathutil.MatchesAny(path, ignoreFiles) {
				continue
			}
			content, err := gitdiff.GetStagedContent(path)
			if err != nil {
				continue
			}
			if cfg.Lint.JSON.IsAllowJSON5() {
				validationErrs = lint.ValidateJSON5(path, content)
			} else {
				validationErrs = lint.ValidateJSON(path, content)
			}

		case ".xml":
			if !cfg.Lint.XML.IsEnabled() {
				continue
			}
			if pathutil.MatchesAny(path, cfg.Lint.XML.IgnoreFiles) {
				continue
			}
			content, err := gitdiff.GetStagedContent(path)
			if err != nil {
				continue
			}
			validationErrs = lint.ValidateXML(path, content)
		}

		for _, ve := range validationErrs {
			errs = append(errs, i18n.T("diff.lint_error", map[string]interface{}{
				"Path":    ve.File,
				"Line":    ve.Line,
				"Message": ve.Message,
			}))
		}
	}

	return errs, nil
}

// getStagedFiles: 스테이지된 파일 경로 목록 반환 (삭제된 파일 제외).
func getStagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--staged", "--name-only", "--diff-filter=d")
	out, err := cmd.Output()
	if err != nil {
		if len(out) == 0 {
			return nil, err
		}
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}
