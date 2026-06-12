package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/progress"
)

var (
	diffFormat string
	diffStaged bool
)

var diffCmd = &cobra.Command{
	Use:  "diff [<commit>] [<commit>]",
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		spec, err := gitdiff.SpecFromArgs(args, diffStaged)
		if err != nil {
			return err
		}
		gitdiff.SetSpec(spec)

		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		steps := []progress.Step{
			{Name: i18n.T("step.binary_detection", nil), Category: "binary", Fn: func(ctx context.Context) ([]string, error) { return checker.CheckBinaryFiles(ctx, cfg) }},
			{Name: i18n.T("step.encoding_check", nil), Category: "encoding", Fn: func(ctx context.Context) ([]string, error) { return checker.CheckEncoding(ctx, cfg) }},
			{Name: i18n.T("step.unicode_check", nil), Category: "unicode", Fn: func(ctx context.Context) ([]string, error) { return checker.CheckUnicode(ctx, cfg) }},
			{Name: i18n.T("step.lint_check", nil), Category: "lint", Fn: func(ctx context.Context) ([]string, error) { return checker.CheckLint(ctx, cfg) }},
			{Name: i18n.T("step.editorconfig_check", nil), Category: "editorconfig", Fn: func(ctx context.Context) ([]string, error) { return checker.CheckEditorConfig(ctx, cfg) }},
			{Name: i18n.T("step.comment_language_check", nil), Category: "comment_language", Fn: func(ctx context.Context) ([]string, error) { return checker.CheckDiff(ctx, cfg) }},
			{Name: i18n.T("step.custom_rules_check", nil), Category: "custom_rules", Fn: func(ctx context.Context) ([]string, error) { return checker.CheckDiffCustomRules(ctx, cfg) }},
			{Name: i18n.T("step.append_only_check", nil), Category: "append_only", Fn: func(ctx context.Context) ([]string, error) { return checker.CheckAppendOnly(ctx, cfg) }},
			{Name: i18n.T("step.cache_dir_check", nil), Category: "cache_dir", Fn: func(ctx context.Context) ([]string, error) { return checker.CheckCacheDirStaged(ctx, cfg) }},
		}

		return runStepsAndReport(cmd.Context(), steps, diffFormat)
	},
}

func init() {
	diffCmd.Short = i18n.T("cmd.diff.short", nil)
	diffCmd.Long = i18n.T("cmd.diff.long", nil)
	diffCmd.Flags().StringVar(&diffFormat, "format", "text", i18n.T("flag.format", nil))
	diffCmd.Flags().BoolVar(&diffStaged, "staged", false, i18n.T("flag.diff_staged", nil))
	diffCmd.Flags().BoolVar(&diffStaged, "cached", false, i18n.T("flag.diff_staged", nil))
	rootCmd.AddCommand(diffCmd)
}
