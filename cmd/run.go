package cmd

import (
	"context"
	"fmt"

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
			{Name: i18n.T("step.binary_detection", nil), Category: "binary", Fn: func(ctx context.Context) ([]string, error) { return checker.RunBinaryFiles(ctx, cfg) }},
			{Name: i18n.T("step.encoding_check", nil), Category: "encoding", Fn: func(ctx context.Context) ([]string, error) { return checker.RunEncoding(ctx, cfg) }},
			{Name: i18n.T("step.unicode_check", nil), Category: "unicode", Fn: func(ctx context.Context) ([]string, error) { return checker.RunUnicode(ctx, cfg) }},
			{Name: i18n.T("step.lint_check", nil), Category: "lint", Fn: func(ctx context.Context) ([]string, error) { return checker.RunLint(ctx, cfg) }},
			{Name: i18n.T("step.editorconfig_check", nil), Category: "editorconfig", Fn: func(ctx context.Context) ([]string, error) { return checker.RunEditorConfig(ctx, cfg) }},
			{Name: i18n.T("step.comment_language_check", nil), Category: "comment_language", Fn: func(ctx context.Context) ([]string, error) { return checker.RunCommentLanguage(ctx, cfg) }},
			{Name: i18n.T("step.cache_dir_check", nil), Category: "cache_dir", Fn: func(ctx context.Context) ([]string, error) { return checker.CheckCacheDirCommitted(ctx, cfg) }},
		}

		return runStepsAndReport(cmd.Context(), steps, runFormat)
	},
}

func init() {
	runCmd.Short = i18n.T("cmd.run.short", nil)
	runCmd.Long = i18n.T("cmd.run.long", nil)
	runCmd.Flags().StringVar(&runFormat, "format", "text", i18n.T("flag.format", nil))
	rootCmd.AddCommand(runCmd)
}
