package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/progress"
)

var diffFormat string

var diffCmd = &cobra.Command{
	Use: "diff",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		steps := []progress.Step{
			{Name: i18n.T("step.binary_detection", nil), Category: "binary", Fn: func() ([]string, error) { return checker.CheckBinaryFiles(cfg) }},
			{Name: i18n.T("step.encoding_check", nil), Category: "encoding", Fn: func() ([]string, error) { return checker.CheckEncoding(cfg) }},
			{Name: i18n.T("step.unicode_check", nil), Category: "unicode", Fn: func() ([]string, error) { return checker.CheckUnicode(cfg) }},
			{Name: i18n.T("step.lint_check", nil), Category: "lint", Fn: func() ([]string, error) { return checker.CheckLint(cfg) }},
			{Name: i18n.T("step.editorconfig_check", nil), Category: "editorconfig", Fn: func() ([]string, error) { return checker.CheckEditorConfig(cfg) }},
			{Name: i18n.T("step.comment_language_check", nil), Category: "comment_language", Fn: func() ([]string, error) { return checker.CheckDiff(cfg) }},
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
	diffCmd.Short = i18n.T("cmd.diff.short", nil)
	diffCmd.Long = i18n.T("cmd.diff.long", nil)
	diffCmd.Flags().StringVar(&diffFormat, "format", "text", i18n.T("flag.format", nil))
	rootCmd.AddCommand(diffCmd)
}
