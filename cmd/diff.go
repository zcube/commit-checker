package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/progress"
)

var diffFormat string

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Check staged diff for policy compliance",
	Long: `Reads the current staged changes (git diff --staged) and checks:
- binary file detection
- file encoding (UTF-8)
- data file lint (YAML, JSON/JSON5, XML)
- .editorconfig compliance
- comment language compliance`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		steps := []progress.Step{
			{Name: "Binary file detection", Category: "binary", Fn: func() ([]string, error) { return checker.CheckBinaryFiles(cfg) }},
			{Name: "Encoding check (UTF-8)", Category: "encoding", Fn: func() ([]string, error) { return checker.CheckEncoding(cfg) }},
			{Name: "Unicode character check", Category: "unicode", Fn: func() ([]string, error) { return checker.CheckUnicode(cfg) }},
			{Name: "Data file lint (YAML, JSON, XML)", Category: "lint", Fn: func() ([]string, error) { return checker.CheckLint(cfg) }},
			{Name: "EditorConfig compliance", Category: "editorconfig", Fn: func() ([]string, error) { return checker.CheckEditorConfig(cfg) }},
			{Name: "Comment language check", Category: "comment_language", Fn: func() ([]string, error) { return checker.CheckDiff(cfg) }},
		}

		result, err := progress.RunWithProgress(steps, progress.Options{
			Quiet:   globalQuiet || diffFormat == "json",
			NoColor: globalNoColor,
		})
		if err != nil {
			return err
		}

		if diffFormat == "json" {
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
	diffCmd.Flags().StringVar(&diffFormat, "format", "text", "output format: text or json")
	rootCmd.AddCommand(diffCmd)
}
