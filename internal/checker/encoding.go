package checker

import (
	ecmod "github.com/zcube/commit-checker/internal/editorconfig"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/encoding"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckEncoding: 스테이지된 텍스트 파일의 UTF-8 인코딩 유효성 검사.
// chardet으로 인코딩을 감지하며, .editorconfig의 charset 설정이 latin1 등이면 건너뜀.
func CheckEncoding(cfg *config.Config) ([]string, error) {
	if !cfg.Encoding.IsEnabled() || !cfg.Encoding.IsRequireUTF8() {
		return nil, nil
	}

	files, err := getStagedFiles()
	if err != nil {
		return nil, err
	}

	ignorePatterns := append(cfg.Exceptions.GlobalIgnore, cfg.Encoding.IgnoreFiles...)

	var errs []string
	for _, path := range files {
		if pathutil.MatchesAny(path, ignorePatterns) {
			continue
		}

		// editorconfig에서 charset이 utf-8이 아닌 경우 (latin1 등) 건너뜀
		def, defErr := ecmod.GetDefinition(path)
		if defErr == nil && def != nil && def.Charset != "" &&
			def.Charset != "utf-8" && def.Charset != "utf-8-bom" {
			continue
		}

		content, err := gitdiff.GetStagedContent(path)
		if err != nil {
			continue
		}

		raw := []byte(content)

		// 바이너리 파일은 건너뜀 (별도의 binary_file 검사에서 처리)
		if encoding.IsBinary(raw) {
			continue
		}

		result := encoding.CheckUTF8(raw)
		if !result.Valid {
			errs = append(errs, i18n.T("diff.encoding_error", map[string]interface{}{
				"Path":    path,
				"Charset": result.DetectedCharset,
			}))
		}
	}

	return errs, nil
}
