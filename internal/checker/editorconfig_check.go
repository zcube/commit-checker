package checker

import (
	"os"

	"github.com/zcube/commit-checker/internal/config"
	ecmod "github.com/zcube/commit-checker/internal/editorconfig"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckEditorConfig: 스테이지된 파일을 .editorconfig 규칙에 따라 검증.
// editorconfig-core-go/v2 패키지를 사용하여 정확한 규칙 매칭을 수행.
func CheckEditorConfig(cfg *config.Config) ([]string, error) {
	if !cfg.EditorConfig.IsEnabled() {
		return nil, nil
	}

	// .editorconfig가 없으면 건너뜀
	if _, err := os.Stat(".editorconfig"); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := getStagedFiles()
	if err != nil {
		return nil, err
	}

	var errs []string
	for _, path := range files {
		if pathutil.MatchesAny(path, cfg.Exceptions.GlobalIgnore) {
			continue
		}
		if pathutil.MatchesAny(path, cfg.EditorConfig.IgnoreFiles) {
			continue
		}

		def, err := ecmod.GetDefinition(path)
		if err != nil || def == nil {
			continue
		}

		content, err := gitdiff.GetStagedContent(path)
		if err != nil {
			continue
		}

		violations := ecmod.Check(path, []byte(content), def)
		for _, v := range violations {
			errs = append(errs, i18n.T("diff.editorconfig_error", map[string]interface{}{
				"Path":    v.File,
				"Line":    v.Line,
				"Message": v.Message,
			}))
		}
	}

	return errs, nil
}
