package comment

import (
	"path/filepath"
	"slices"
	"strings"
)

var globalParsers []Parser

func init() {
	globalParsers = []Parser{
		&GoParser{},
		// JS/TS: hasTemplate=true 로 백틱 템플릿 리터럴을 처리합니다.
		NewCStyleParser([]string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}, true),
		// C 계열 및 JVM 언어
		NewCStyleParser([]string{".java", ".kt", ".kts", ".c", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".cs", ".swift", ".rs"}, false),
		&PythonParser{},
		&DockerfileParser{},
		&MarkdownParser{},
	}
}

// isDockerfilePath 는 Dockerfile, Dockerfile.*, *.dockerfile 이름의 파일이면 true 를 반환합니다.
func isDockerfilePath(path string) bool {
	base := filepath.Base(path)
	lower := strings.ToLower(base)
	return lower == "dockerfile" ||
		strings.HasPrefix(lower, "dockerfile.") ||
		strings.HasSuffix(lower, ".dockerfile")
}

// languageExtensions 는 언어 이름을 파일 확장자 목록으로 매핑합니다.
var languageExtensions = map[string][]string{
	"go":         {".go"},
	"typescript": {".ts", ".tsx"},
	"javascript": {".js", ".jsx", ".mjs", ".cjs"},
	"java":       {".java"},
	"kotlin":     {".kt", ".kts"},
	"python":     {".py"},
	"c":          {".c", ".h"},
	"cpp":        {".cpp", ".cc", ".cxx", ".hpp"},
	"csharp":     {".cs"},
	"swift":      {".swift"},
	"rust":       {".rs"},
	"dockerfile": {"dockerfile"},
	"markdown":   {".md", ".markdown"},
}

// ExtensionsForLanguages 는 주어진 언어 이름들의 파일 확장자 합집합을 반환합니다.
// 알 수 없는 언어 이름은 무시됩니다.
func ExtensionsForLanguages(langs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, lang := range langs {
		for _, ext := range languageExtensions[lang] {
			if !seen[ext] {
				seen[ext] = true
				result = append(result, ext)
			}
		}
	}
	return result
}

// GetParser 는 path 에 맞는 Parser 를 반환합니다. 지원하지 않는 확장자면 nil 을 반환합니다.
func GetParser(path string) Parser {
	// Dockerfile* 파일명 패턴 우선 확인
	if isDockerfilePath(path) {
		return &DockerfileParser{}
	}
	ext := filepath.Ext(path)
	for _, p := range globalParsers {
		if slices.Contains(p.SupportedExtensions(), ext) {
			return p
		}
	}
	return nil
}
