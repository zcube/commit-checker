package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/i18n"
)

// langInfo: 감지된 프로그래밍 언어 정보.
type langInfo struct {
	Name  string
	Count int
}

// lintConfigCheck: 언어명과 알려진 lint 설정 파일 목록의 매핑.
type lintConfigCheck struct {
	Language    string
	ConfigFiles []string
}

var knownLintConfigs = []lintConfigCheck{
	{"Go", []string{".golangci.yml", ".golangci.yaml", ".golangci.toml", ".golangci.json"}},
	{"TypeScript/JavaScript", []string{
		".eslintrc", ".eslintrc.json", ".eslintrc.yml", ".eslintrc.yaml", ".eslintrc.js", ".eslintrc.cjs",
		"eslint.config.js", "eslint.config.mjs", "eslint.config.cjs", "eslint.config.ts",
		"biome.json", "biome.jsonc", ".prettierrc", ".prettierrc.json", ".prettierrc.yml",
	}},
	{"Python", []string{
		".pylintrc", ".flake8", ".ruff.toml", "ruff.toml",
		"pyproject.toml", "setup.cfg", ".pyre_configuration",
	}},
	{"Java", []string{"checkstyle.xml", "pmd.xml", ".checkstyle", "spotbugs.xml"}},
	{"Kotlin", []string{"detekt.yml", "detekt.yaml", ".detekt.yml"}},
	{"Rust", []string{"rustfmt.toml", ".rustfmt.toml", "clippy.toml"}},
	{"C/C++", []string{".clang-format", ".clang-tidy"}},
	{"Swift", []string{".swiftlint.yml", ".swiftformat"}},
	{"C#", []string{".editorconfig", "omnisharp.json", "Directory.Build.props"}},
}

// extensionToLanguage: 파일 확장자 → 언어명 매핑.
var extensionToLanguage = map[string]string{
	".go":    "Go",
	".ts":    "TypeScript/JavaScript",
	".tsx":   "TypeScript/JavaScript",
	".js":    "TypeScript/JavaScript",
	".jsx":   "TypeScript/JavaScript",
	".mjs":   "TypeScript/JavaScript",
	".cjs":   "TypeScript/JavaScript",
	".py":    "Python",
	".java":  "Java",
	".kt":    "Kotlin",
	".kts":   "Kotlin",
	".rs":    "Rust",
	".c":     "C/C++",
	".cpp":   "C/C++",
	".cc":    "C/C++",
	".h":     "C/C++",
	".hpp":   "C/C++",
	".swift": "Swift",
	".cs":    "C#",
	".rb":    "Ruby",
	".php":   "PHP",
	".yaml":  "YAML",
	".yml":   "YAML",
	".json":  "JSON",
	".xml":   "XML",
	".html":  "HTML",
	".css":   "CSS",
	".scss":  "CSS",
	".sh":    "Shell",
	".bash":  "Shell",
	".md":    "Markdown",
}

var analyzeCmd = &cobra.Command{
	Use: "analyze",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAnalyze()
	},
}

func init() {
	analyzeCmd.Short = i18n.T("cmd.analyze.short", nil)
	analyzeCmd.Long = i18n.T("cmd.analyze.long", nil)
	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze() error {
	// 1. 추적된 파일 목록 수집
	files, err := getTrackedFiles()
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	// 2. 언어별 파일 수 집계
	langCounts := make(map[string]int)
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if lang, ok := extensionToLanguage[ext]; ok {
			langCounts[lang]++
		}
	}

	// 파일 수 기준 내림차순 정렬
	var langs []langInfo
	for name, count := range langCounts {
		langs = append(langs, langInfo{Name: name, Count: count})
	}
	sort.Slice(langs, func(i, j int) bool { return langs[i].Count > langs[j].Count })

	// 3. 출력
	fmt.Println(i18n.T("analyze.header", nil))
	fmt.Println()

	// 감지된 언어 목록
	fmt.Println(i18n.T("analyze.detected_languages", nil))
	if len(langs) == 0 {
		fmt.Println(i18n.T("analyze.no_languages", nil))
	}
	for _, l := range langs {
		fmt.Println(i18n.T("analyze.lang_entry", map[string]any{"Name": l.Name, "Count": l.Count}))
	}
	fmt.Println()

	// 4. lint 설정 파일 존재 여부 확인
	fmt.Println(i18n.T("analyze.lint_config_status", nil))
	programmingLangs := filterProgrammingLangs(langs)
	if len(programmingLangs) == 0 {
		fmt.Println(i18n.T("analyze.no_programming_langs", nil))
	}
	for _, l := range programmingLangs {
		found, configName := checkLintConfig(l.Name)
		if found {
			fmt.Println(i18n.T("analyze.lint_found", map[string]any{"Language": l.Name, "Config": configName}))
		} else {
			fmt.Println(i18n.T("analyze.lint_not_found", map[string]any{"Language": l.Name}))
		}
	}
	fmt.Println()

	// 5. 일반 설정 파일 존재 여부 확인
	fmt.Println(i18n.T("analyze.project_config", nil))
	checkAndReport(".editorconfig")
	checkAndReport(".commit-checker.yml")
	checkAndReport(".gitattributes")
	checkAndReport(".gitignore")
	fmt.Println()

	// 6. 데이터 파일 수 출력
	dataLangs := filterDataLangs(langs)
	if len(dataLangs) > 0 {
		fmt.Println(i18n.T("analyze.data_files", nil))
		for _, l := range dataLangs {
			fmt.Println(i18n.T("analyze.lang_entry", map[string]any{"Name": l.Name, "Count": l.Count}))
		}
		fmt.Println()
	}

	return nil
}

func getTrackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

func filterProgrammingLangs(langs []langInfo) []langInfo {
	dataTypes := map[string]bool{
		"YAML": true, "JSON": true, "XML": true,
		"HTML": true, "CSS": true, "Markdown": true, "Shell": true,
	}
	var result []langInfo
	for _, l := range langs {
		if !dataTypes[l.Name] {
			result = append(result, l)
		}
	}
	return result
}

func filterDataLangs(langs []langInfo) []langInfo {
	dataTypes := map[string]bool{
		"YAML": true, "JSON": true, "XML": true,
	}
	var result []langInfo
	for _, l := range langs {
		if dataTypes[l.Name] {
			result = append(result, l)
		}
	}
	return result
}

func checkLintConfig(language string) (bool, string) {
	for _, lc := range knownLintConfigs {
		if lc.Language == language {
			for _, cf := range lc.ConfigFiles {
				if _, err := os.Stat(cf); err == nil {
					return true, cf
				}
			}
			return false, ""
		}
	}
	return false, ""
}

func checkAndReport(filename string) {
	if _, err := os.Stat(filename); err == nil {
		fmt.Println(i18n.T("analyze.file_found", map[string]any{"File": filename}))
	} else {
		fmt.Println(i18n.T("analyze.file_not_found", map[string]any{"File": filename}))
	}
}
