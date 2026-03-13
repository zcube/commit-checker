package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/progress"
)

var runFormat string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Check all tracked files for policy compliance",
	Long: `Reads all files tracked by git (git ls-files) and checks:
- binary file detection
- file encoding (UTF-8)
- data file lint (YAML, JSON/JSON5, XML)
- .editorconfig compliance
- comment language compliance

Unlike 'diff', this command checks all files regardless of staged state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		steps := []progress.Step{
			{Name: "Binary file detection", Category: "binary", Fn: func() ([]string, error) { return checker.RunBinaryFiles(cfg) }},
			{Name: "Encoding check (UTF-8)", Category: "encoding", Fn: func() ([]string, error) { return checker.RunEncoding(cfg) }},
			{Name: "Unicode character check", Category: "unicode", Fn: func() ([]string, error) { return checker.RunUnicode(cfg) }},
			{Name: "Data file lint (YAML, JSON, XML)", Category: "lint", Fn: func() ([]string, error) { return checker.RunLint(cfg) }},
			{Name: "EditorConfig compliance", Category: "editorconfig", Fn: func() ([]string, error) { return checker.RunEditorConfig(cfg) }},
			{Name: "Comment language check", Category: "comment_language", Fn: func() ([]string, error) { return checker.RunCommentLanguage(cfg) }},
		}

		result, err := progress.RunWithProgress(steps, progress.Options{
			Quiet:   globalQuiet || runFormat == "json",
			NoColor: globalNoColor,
		})
		if err != nil {
			return err
		}

		if runFormat == "json" {
			jsonBytes, jsonErr := progress.FormatJSON(result)
			if jsonErr != nil {
				return jsonErr
			}
			fmt.Println(string(jsonBytes))
			if len(result.AllErrors) > 0 {
				os.Exit(1)
			}
			return nil
		}

		// 텍스트 출력
		for _, e := range result.AllErrors {
			fmt.Fprintln(os.Stderr, e)
		}
		if len(result.AllErrors) > 0 {
			if summary := progress.SummaryLine(result.Steps); summary != "" {
				fmt.Fprintln(os.Stderr, summary)
			}
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	runCmd.Flags().StringVar(&runFormat, "format", "text", "output format: text or json")
	rootCmd.AddCommand(runCmd)
}
