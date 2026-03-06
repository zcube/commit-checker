package comment

import "path/filepath"

var globalParsers []Parser

func init() {
	globalParsers = []Parser{
		&GoParser{},
		// JS/TS: hasTemplate=true to handle backtick template literals
		NewCStyleParser([]string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}, true),
		// C-family and JVM languages
		NewCStyleParser([]string{".java", ".kt", ".kts", ".c", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".cs", ".swift", ".rs"}, false),
		&PythonParser{},
	}
}

// languageExtensions maps friendly language names to their file extensions.
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

// ExtensionsForLanguages returns the union of file extensions for the given language names.
// Unknown language names are silently ignored.
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

// GetParser returns the appropriate Parser for the file at path,
// or nil if the extension is not supported.
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
