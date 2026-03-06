package comment

import "path/filepath"

var globalParsers []Parser

func init() {
	globalParsers = []Parser{
		&GoParser{},
		// JS/TS: 백틱 템플릿 리터럴 처리를 위해 hasTemplate=true
		NewCStyleParser([]string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}, true),
		// C 계열 및 JVM 언어
		NewCStyleParser([]string{".java", ".kt", ".kts", ".c", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".cs", ".swift", ".rs"}, false),
		&PythonParser{},
	}
}

// languageExtensions: 언어 이름과 파일 확장자의 매핑.
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
}

// ExtensionsForLanguages: 주어진 언어 이름들에 대한 파일 확장자 합집합을 반환.
// 알 수 없는 언어 이름은 조용히 무시.
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

// GetParser: 주어진 파일 경로에 적합한 Parser를 반환.
// 지원하지 않는 확장자면 nil을 반환.
func GetParser(path string) Parser {
	ext := filepath.Ext(path)
	for _, p := range globalParsers {
		for _, e := range p.SupportedExtensions() {
			if ext == e {
				return p
			}
		}
	}
	return nil
}
