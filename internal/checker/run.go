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

	var (
		mu   sync.Mutex
		errs []string
	)
	g := new(errgroup.Group)
	sem := make(chan struct{}, runtime.NumCPU()*2)

	for _, path := range files {

		if pathutil.MatchesAny(path, ignorePatterns) {
			continue
		}
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if encoding.IsBinary(content) {
				if msg := evaluateBinaryPolicy(&cfg.BinaryFile, path); msg != "" {
					mu.Lock()
					errs = append(errs, msg)
					mu.Unlock()
				}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
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

	var (
		mu   sync.Mutex
		errs []string
	)
	g := new(errgroup.Group)
	sem := make(chan struct{}, runtime.NumCPU()*2)

	for _, path := range files {

		if pathutil.MatchesAny(path, ignorePatterns) {
			continue
		}
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			// editorconfig에서 charset이 utf-8이 아닌 경우 건너뜀
			def, defErr := ecmod.GetDefinition(path)
			if defErr == nil && def != nil && def.Charset != "" &&
				def.Charset != "utf-8" && def.Charset != "utf-8-bom" {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			if encoding.IsBinary(content) {
				return nil
			}

			result := encoding.CheckUTF8(content)
			if !result.Valid {
				msg := i18n.T("diff.encoding_error", map[string]any{
					"Path":    path,
					"Charset": result.DetectedCharset,
				})
				mu.Lock()
				errs = append(errs, msg)
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

	var (
		mu   sync.Mutex
		errs []string
	)
	g := new(errgroup.Group)
	sem := make(chan struct{}, runtime.NumCPU()*2)

	for _, path := range files {

		if pathutil.MatchesAny(path, globalIgnore) {
			continue
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".yaml", ".yml", ".json", ".jsonc", ".xml", ".toml":
			// 처리 대상
		default:
			continue
		}

		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			ext := strings.ToLower(filepath.Ext(path))
			var validationErrs []lint.ValidationError

			switch ext {
			case ".yaml", ".yml":
				if !cfg.Lint.YAML.IsEnabled() {
					return nil
				}
				if pathutil.MatchesAny(path, cfg.Lint.YAML.IgnoreFiles) {
					return nil
				}
				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				if cfg.Lint.YAML.IsCommentFilter() && lint.HasLintDisableComment(string(content), "#") {
					return nil
				}
				validationErrs = lint.ValidateYAML(path, string(content))

			case ".jsonc":
				if !cfg.Lint.JSON.IsEnabled() {
					return nil
				}
				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				validationErrs = lint.ValidateJSON5(path, string(content))

			case ".json":
				if !cfg.Lint.JSON.IsEnabled() {
					return nil
				}
				ignoreFiles := cfg.Lint.JSON.IgnoreFiles
				if len(ignoreFiles) == 0 {
					ignoreFiles = lint.DefaultJSONIgnoreFiles
				}
				if pathutil.MatchesAny(path, ignoreFiles) {
					return nil
				}
				content, err := os.ReadFile(path)
				if err != nil {
					return nil
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
					return nil
				}
				if pathutil.MatchesAny(path, cfg.Lint.XML.IgnoreFiles) {
					return nil
				}
				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				validationErrs = lint.ValidateXML(path, string(content))

			case ".toml":
				if !cfg.Lint.TOML.IsEnabled() {
					return nil
				}
				if pathutil.MatchesAny(path, cfg.Lint.TOML.IgnoreFiles) {
					return nil
				}
				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				validationErrs = lint.ValidateTOML(path, string(content))
			}

			if len(validationErrs) > 0 {
				msgs := make([]string, 0, len(validationErrs))
				for _, ve := range validationErrs {
					msgs = append(msgs, i18n.T("diff.lint_error", map[string]any{
						"Path":    ve.File,
						"Line":    ve.Line,
						"Message": ve.Message,
					}))
				}
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

	var (
		mu   sync.Mutex
		errs []string
	)
	g := new(errgroup.Group)
	sem := make(chan struct{}, runtime.NumCPU()*2)

	for _, path := range files {

		if pathutil.MatchesAny(path, cfg.Exceptions.GlobalIgnore) {
			continue
		}
		if pathutil.MatchesAny(path, cfg.EditorConfig.IgnoreFiles) {
			continue
		}
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			def, err := ecmod.GetDefinition(path)
			if err != nil || def == nil {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			violations := ecmod.Check(path, content, def)
			if len(violations) > 0 {
				msgs := make([]string, 0, len(violations))
				for _, v := range violations {
					msgs = append(msgs, i18n.T("diff.editorconfig_error", map[string]any{
						"Path":    v.File,
						"Line":    v.Line,
						"Message": v.Message,
					}))
				}
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

	var (
		mu   sync.Mutex
		errs []string
	)
	g := new(errgroup.Group)
	sem := make(chan struct{}, runtime.NumCPU()*2)

	for _, filePath := range files {

		if !gitdiff.HasExtension(filePath, extensions) {
			continue
		}
		if pathutil.MatchesAny(filePath, ignorePatterns) {
			continue
		}

		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			parser := comment.GetParser(filePath)
			if parser == nil {
				return nil
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil
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
