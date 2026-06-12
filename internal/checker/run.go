package checker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

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
	"github.com/zcube/commit-checker/internal/logger"
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

// forEachFileConcurrent: files를 병렬로 순회하며 fn이 반환한 위반 메시지를 수집.
// 동시 실행 수는 runtime.NumCPU()*2로 제한하며, 메시지는 고루틴 완료 순서대로
// 수집된다(기존 동작과 동일하게 별도 정렬하지 않음).
func forEachFileConcurrent(files []string, fn func(path string) ([]string, error)) ([]string, error) {
	var (
		mu   sync.Mutex
		errs []string
	)
	g := new(errgroup.Group)
	sem := make(chan struct{}, runtime.NumCPU()*2)

	for _, path := range files {
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			msgs, err := fn(path)
			if err != nil {
				return err
			}
			if len(msgs) > 0 {
				mu.Lock()
				errs = append(errs, msgs...)
				mu.Unlock()
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return errs, nil
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

	return forEachFileConcurrent(files, func(path string) ([]string, error) {
		if pathutil.MatchesAny(path, ignorePatterns) {
			return nil, nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil, nil
		}
		if encoding.IsBinary(content) {
			if msg := evaluateBinaryPolicy(&cfg.BinaryFile, path); msg != "" {
				return []string{msg}, nil
			}
		}
		return nil, nil
	})
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

	return forEachFileConcurrent(files, func(path string) ([]string, error) {
		if pathutil.MatchesAny(path, ignorePatterns) {
			return nil, nil
		}

		// editorconfig에서 charset이 utf-8이 아닌 경우 건너뜀
		def, defErr := ecmod.GetDefinition(path)
		if defErr == nil && def != nil && def.Charset != "" &&
			def.Charset != "utf-8" && def.Charset != "utf-8-bom" {
			return nil, nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil, nil
		}

		if encoding.IsBinary(content) {
			return nil, nil
		}

		result := encoding.CheckUTF8(content)
		if !result.Valid {
			msg := i18n.T("diff.encoding_error", map[string]any{
				"Path":    path,
				"Charset": result.DetectedCharset,
			})
			return []string{msg}, nil
		}
		return nil, nil
	})
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

	return forEachFileConcurrent(files, func(path string) ([]string, error) {
		if pathutil.MatchesAny(path, globalIgnore) {
			return nil, nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		var validationErrs []lint.ValidationError

		switch ext {
		case ".yaml", ".yml":
			if !cfg.Lint.YAML.IsEnabled() {
				return nil, nil
			}
			if pathutil.MatchesAny(path, cfg.Lint.YAML.IgnoreFiles) {
				return nil, nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, nil
			}
			if cfg.Lint.YAML.IsCommentFilter() && lint.HasLintDisableComment(string(content), "#") {
				return nil, nil
			}
			validationErrs = lint.ValidateYAML(path, string(content))

		case ".jsonc":
			if !cfg.Lint.JSON.IsEnabled() {
				return nil, nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, nil
			}
			validationErrs = lint.ValidateJSON5(path, string(content))

		case ".json":
			if !cfg.Lint.JSON.IsEnabled() {
				return nil, nil
			}
			ignoreFiles := cfg.Lint.JSON.IgnoreFiles
			if len(ignoreFiles) == 0 {
				ignoreFiles = lint.DefaultJSONIgnoreFiles
			}
			if pathutil.MatchesAny(path, ignoreFiles) {
				return nil, nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, nil
			}
			if cfg.Lint.JSON.IsAllowJSON5() {
				validationErrs = lint.ValidateJSON5(path, string(content))
			} else if cfg.Lint.JSON.IsCommentFilter() {
				validationErrs = lint.ValidateJSONC(path, string(content))
			} else {
				validationErrs = lint.ValidateJSON(path, string(content))
			}

		case ".xml":
			if !cfg.Lint.XML.IsEnabled() {
				return nil, nil
			}
			if pathutil.MatchesAny(path, cfg.Lint.XML.IgnoreFiles) {
				return nil, nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, nil
			}
			validationErrs = lint.ValidateXML(path, string(content))

		case ".toml":
			if !cfg.Lint.TOML.IsEnabled() {
				return nil, nil
			}
			if pathutil.MatchesAny(path, cfg.Lint.TOML.IgnoreFiles) {
				return nil, nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, nil
			}
			validationErrs = lint.ValidateTOML(path, string(content))

		default:
			return nil, nil
		}

		if len(validationErrs) == 0 {
			return nil, nil
		}
		msgs := make([]string, 0, len(validationErrs))
		for _, ve := range validationErrs {
			msgs = append(msgs, i18n.T("diff.lint_error", map[string]any{
				"Path":    ve.File,
				"Line":    ve.Line,
				"Message": ve.Message,
			}))
		}
		return msgs, nil
	})
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

	return forEachFileConcurrent(files, func(path string) ([]string, error) {
		if pathutil.MatchesAny(path, cfg.Exceptions.GlobalIgnore) {
			return nil, nil
		}
		if pathutil.MatchesAny(path, cfg.EditorConfig.IgnoreFiles) {
			return nil, nil
		}

		def, err := ecmod.GetDefinition(path)
		if err != nil || def == nil {
			return nil, nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil, nil
		}

		violations := ecmod.Check(path, content, def)
		if len(violations) == 0 {
			return nil, nil
		}
		msgs := make([]string, 0, len(violations))
		for _, v := range violations {
			msgs = append(msgs, i18n.T("diff.editorconfig_error", map[string]any{
				"Path":    v.File,
				"Line":    v.Line,
				"Message": v.Message,
			}))
		}
		return msgs, nil
	})
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
	noEmoji := cfg.CommentLanguage.IsNoEmoji()
	allowedWords := cfg.CommentLanguage.AllowedWords

	ignorePatterns := append(cfg.Exceptions.GlobalIgnore,
		cfg.Exceptions.CommentLanguageIgnore...)
	ignorePatterns = append(ignorePatterns, cfg.CommentLanguage.IgnoreFiles...)

	return forEachFileConcurrent(files, func(filePath string) ([]string, error) {
		if !gitdiff.HasExtension(filePath, extensions) {
			return nil, nil
		}
		if pathutil.MatchesAny(filePath, ignorePatterns) {
			return nil, nil
		}

		parser := comment.GetParser(filePath)
		if parser == nil {
			return nil, nil
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, nil
		}

		comments, err := parser.ParseFile(string(content))
		if err != nil {
			logger.Warn("comment parse warning", "path", filePath, "error", err)
		}

		fileLang := resolveFileLang(filePath, cfg)
		states := directive.Analyze(comments, fileLang)

		var msgs []string
		for _, u := range buildCommentUnits(comments, states, checkStrings) {
			if u.kind == comment.KindString {
				// 문자열 리터럴: 언어 감지 제외 (유니코드 검사는 RunUnicode 에서 처리)
				continue
			}

			text := langdetect.StripAllowedWords(u.text, allowedWords)
			ok, hasContent := langdetect.IsRequiredLanguage(text, u.lang, minLength, skipDirectives)
			if !hasContent {
				continue
			}
			if !ok {
				detected := langdetect.Dominant(text)
				msgs = append(msgs, i18n.T("diff.comment_language_error", map[string]any{
					"Path":     filePath,
					"Line":     u.line,
					"Kind":     i18n.T("diff.kind_comment", nil),
					"Language": u.lang,
					"Detected": detected,
					"Text":     truncate(text, 80),
				}))
			}

			if noEmoji {
				emojis := emoji.FindEmojis(text)
				for _, e := range emojis {
					msgs = append(msgs, i18n.T("diff.emoji_error", map[string]any{
						"Path":     filePath,
						"Line":     u.line + e.Line - 1,
						"Kind":     i18n.T("diff.kind_comment", nil),
						"Char":     e.Char,
						"CharCode": fmt.Sprintf("%04X", e.Code),
					}))
				}
			}
		}
		return msgs, nil
	})
}
