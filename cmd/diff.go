package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
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

		// git diff 는 커맨드 진입 시 1회만 실행하고 각 검사 step 에 주입
		// (SetSpec 이후에 호출해야 비교 대상이 올바르게 적용됨)
		diffs, err := gitdiff.GetStagedDiff()
		if err != nil {
			return err
		}

		return runStepsAndReport(cmd.Context(), diffSteps(cfg, diffs), diffFormat, guideEnabled(cfg))
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
