package checker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zcube/commit-checker/internal/comment"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/directive"
	ecmod "github.com/zcube/commit-checker/internal/editorconfig"
	"github.com/zcube/commit-checker/internal/emoji"
	"github.com/zcube/commit-checker/internal/encoding"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/langdetect"
	"github.com/zcube/commit-checker/internal/lint"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// getTrackedFiles: git ls-files로 추적된 전체 파일 목록 반환.
func getTrackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files")
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

// RunBinaryFiles: 추적된 모든 파일에서 바이너리 파일을 검사.
// 스테이지 상태에 관계없이 워킹 트리의 파일을 직접 읽어 검사.
func RunBinaryFiles(cfg *config.Config) ([]string, error) {
	if !cfg.BinaryFile.IsEnabled() {
		return nil, nil
	}

	files, err := getTrackedFiles()
	if err != nil {
		return nil, err
	}

	ignorePatterns := append(cfg.Exceptions.GlobalIgnore, cfg.BinaryFile.IgnoreFiles...)

	var errs []string
	for _, path := range files {
		if pathutil.MatchesAny(path, ignorePatterns) {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if encoding.IsBinary(content) {
			errs = append(errs, i18n.T("diff.binary_file_error", map[string]interface{}{
				"Path": path,
			}))
		}
	}
	return errs, nil
}

// RunEncoding: 추적된 모든 파일의 UTF-8 인코딩 유효성을 검사.
func RunEncoding(cfg *config.Config) ([]string, error) {
	if !cfg.Encoding.IsEnabled() || !cfg.Encoding.IsRequireUTF8() {
		return nil, nil
	}

	files, err := getTrackedFiles()
	if err != nil {
		return nil, err
	}

	ignorePatterns := append(cfg.Exceptions.GlobalIgnore, cfg.Encoding.IgnoreFiles...)

	var errs []string
	for _, path := range files {
		if pathutil.MatchesAny(path, ignorePatterns) {
			continue
		}

		// editorconfig에서 charset이 utf-8이 아닌 경우 건너뜀
		def, defErr := ecmod.GetDefinition(path)
		if defErr == nil && def != nil && def.Charset != "" &&
			def.Charset != "utf-8" && def.Charset != "utf-8-bom" {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		if encoding.IsBinary(content) {
			continue
		}

		result := encoding.CheckUTF8(content)
		if !result.Valid {
			errs = append(errs, i18n.T("diff.encoding_error", map[string]interface{}{
				"Path":    path,
				"Charset": result.DetectedCharset,
			}))
		}
	}
	return errs, nil
}

// RunLint: 추적된 모든 데이터 파일(YAML, JSON, XML)의 구문 오류를 검사.
func RunLint(cfg *config.Config) ([]string, error) {
	if !cfg.Lint.IsEnabled() {
		return nil, nil
	}

	files, err := getTrackedFiles()
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
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			validationErrs = lint.ValidateYAML(path, string(content))

		case ".json":
			if !cfg.Lint.JSON.IsEnabled() {
				continue
			}
			ignoreFiles := cfg.Lint.JSON.IgnoreFiles
			if len(ignoreFiles) == 0 {
				ignoreFiles = lint.DefaultJSONIgnoreFiles
			}
			if pathutil.MatchesAny(path, ignoreFiles) {
				continue
			}
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if cfg.Lint.JSON.IsAllowJSON5() {
				validationErrs = lint.ValidateJSON5(path, string(content))
			} else {
				validationErrs = lint.ValidateJSON(path, string(content))
			}

		case ".xml":
			if !cfg.Lint.XML.IsEnabled() {
				continue
			}
			if pathutil.MatchesAny(path, cfg.Lint.XML.IgnoreFiles) {
				continue
			}
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			validationErrs = lint.ValidateXML(path, string(content))
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

// RunEditorConfig: 추적된 모든 파일의 .editorconfig 규칙 준수 여부를 검사.
func RunEditorConfig(cfg *config.Config) ([]string, error) {
	if !cfg.EditorConfig.IsEnabled() {
		return nil, nil
	}

	if _, err := os.Stat(".editorconfig"); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := getTrackedFiles()
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

		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		violations := ecmod.Check(path, content, def)
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

// RunCommentLanguage: 추적된 모든 소스 파일의 주석 언어를 검사.
// check_mode 설정에 관계없이 항상 파일 전체를 검사.
func RunCommentLanguage(cfg *config.Config) ([]string, error) {
	if !cfg.CommentLanguage.IsEnabled() {
		return nil, nil
	}

	files, err := getTrackedFiles()
	if err != nil {
		return nil, err
	}

	extensions := cfg.CommentLanguage.Extensions
	if len(cfg.CommentLanguage.Languages) > 0 {
		extensions = comment.ExtensionsForLanguages(cfg.CommentLanguage.Languages)
	}

	minLength := cfg.CommentLanguage.MinLength
	skipDirectives := cfg.CommentLanguage.SkipDirectives
	checkStrings := cfg.CommentLanguage.IsCheckStrings()
	skipTechnical := cfg.CommentLanguage.IsSkipTechnicalStrings()
	noEmoji := cfg.CommentLanguage.IsNoEmoji()

	ignorePatterns := append(cfg.Exceptions.GlobalIgnore,
		cfg.Exceptions.CommentLanguageIgnore...)
	ignorePatterns = append(ignorePatterns, cfg.CommentLanguage.IgnoreFiles...)

	var errs []string

	for _, filePath := range files {
		if !gitdiff.HasExtension(filePath, extensions) {
			continue
		}
		if pathutil.MatchesAny(filePath, ignorePatterns) {
			continue
		}

		parser := comment.GetParser(filePath)
		if parser == nil {
			continue
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		comments, err := parser.ParseFile(string(content))
		if err != nil {
			fmt.Println(i18n.T("diff.parse_warning", map[string]interface{}{
				"Path":  filePath,
				"Error": err,
			}))
		}

		fileLang := resolveFileLang(filePath, cfg)
		states := directive.Analyze(comments, fileLang)

		for i, c := range comments {
			state := states[i]
			if state.Skip {
				continue
			}
			if c.Kind == comment.KindString && !checkStrings {
				continue
			}

			text := strings.TrimSpace(c.Text)
			if c.Kind == comment.KindString && skipTechnical && IsTechnicalString(text) {
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
					"Path":     filePath,
					"Line":     c.Line,
					"Kind":     i18n.T(kindID, nil),
					"Language": state.Language,
					"Detected": detected,
					"Text":     truncate(text, 80),
				}))
			}

			if noEmoji {
				emojis := emoji.FindEmojis(text)
				for _, e := range emojis {
					kindID := "diff.kind_comment"
					if c.Kind == comment.KindString {
						kindID = "diff.kind_string_literal"
					}
					errs = append(errs, i18n.T("diff.emoji_error", map[string]interface{}{
						"Path":     filePath,
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
