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
	diffOnly   []string
)

var diffCmd = &cobra.Command{
	Use:  "diff [<commit>] [<commit>]",
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// --require-config: 프로젝트 설정 파일이 없으면 아무 출력 없이 성공 종료 (전역 opt-in 설치)
		if requireConfigSkip() {
			return nil
		}

		// --only 카테고리 검증은 설정 로드보다 먼저 수행 (잘못된 입력 즉시 보고)
		defs, err := stepDefsFor(true, diffOnly)
		if err != nil {
			return err
		}

		spec, err := gitdiff.SpecFromArgs(args, diffStaged)
		if err != nil {
			return err
		}
		gitdiff.SetSpec(spec)

		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		// enabled: false — 리포 단위 opt-out: 모든 검사를 건너뛰고 성공 종료
		if !cfg.IsEnabled() {
			return nil
		}
		if len(diffOnly) > 0 {
			// --only 는 설정의 enabled: false 를 덮어쓰고 지정 검사를 강제 실행
			cfg = cfgWithOnlyEnabled(cfg, defs)
		}

		// git diff 는 커맨드 진입 시 1회만 실행하고 각 검사 step 에 주입
		// (SetSpec 이후에 호출해야 비교 대상이 올바르게 적용됨)
		diffs, err := gitdiff.GetStagedDiff()
		if err != nil {
			return err
		}

		return runStepsAndReport(cmd.Context(), diffSteps(defs, cfg, diffs), diffFormat, guideEnabled(cfg))
	},
}

func init() {
	diffCmd.Short = i18n.T("cmd.diff.short", nil)
	diffCmd.Long = i18n.T("cmd.diff.long", nil)
	diffCmd.Flags().StringVar(&diffFormat, "format", "text", i18n.T("flag.format", nil))
	diffCmd.Flags().BoolVar(&diffStaged, "staged", false, i18n.T("flag.diff_staged", nil))
	diffCmd.Flags().BoolVar(&diffStaged, "cached", false, i18n.T("flag.diff_staged", nil))
	diffCmd.Flags().StringSliceVar(&diffOnly, "only", nil, i18n.T("flag.only", nil))
	rootCmd.AddCommand(diffCmd)
}
