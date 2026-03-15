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

var runFormat string

var runCmd = &cobra.Command{
	Use: "run",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		steps := []progress.Step{
			{Name: i18n.T("step.binary_detection", nil), Category: "binary", Fn: func() ([]string, error) { return checker.RunBinaryFiles(cfg) }},
			{Name: i18n.T("step.encoding_check", nil), Category: "encoding", Fn: func() ([]string, error) { return checker.RunEncoding(cfg) }},
			{Name: i18n.T("step.unicode_check", nil), Category: "unicode", Fn: func() ([]string, error) { return checker.RunUnicode(cfg) }},
			{Name: i18n.T("step.lint_check", nil), Category: "lint", Fn: func() ([]string, error) { return checker.RunLint(cfg) }},
			{Name: i18n.T("step.editorconfig_check", nil), Category: "editorconfig", Fn: func() ([]string, error) { return checker.RunEditorConfig(cfg) }},
			{Name: i18n.T("step.comment_language_check", nil), Category: "comment_language", Fn: func() ([]string, error) { return checker.RunCommentLanguage(cfg) }},
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
	runCmd.Short = i18n.T("cmd.run.short", nil)
	runCmd.Long = i18n.T("cmd.run.long", nil)
	runCmd.Flags().StringVar(&runFormat, "format", "text", i18n.T("flag.format", nil))
	rootCmd.AddCommand(runCmd)
}
