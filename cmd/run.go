package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
)

var (
	runFormat string
	runOnly   []string
)

var runCmd = &cobra.Command{
	Use: "run",
	RunE: func(cmd *cobra.Command, args []string) error {
		// --require-config: 프로젝트 설정 파일이 없으면 아무 출력 없이 성공 종료 (전역 opt-in 설치)
		if requireConfigSkip() {
			return nil
		}

		// --only 카테고리 검증은 설정 로드보다 먼저 수행 (잘못된 입력 즉시 보고)
		defs, err := stepDefsFor(false, runOnly)
		if err != nil {
			return err
		}

		cfg, err := config.Load(resolveConfigFilePath(configFile))
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		// enabled: false — 리포 단위 opt-out: 모든 검사를 건너뛰고 성공 종료
		if !cfg.IsEnabled() {
			return nil
		}
		if len(runOnly) > 0 {
			// --only 는 설정의 enabled: false 를 덮어쓰고 지정 검사를 강제 실행
			cfg = cfgWithOnlyEnabled(cfg, defs)
		}

		// git ls-files 는 커맨드 진입 시 1회만 실행하고 각 검사 step 에 주입
		files, err := checker.GetTrackedFiles()
		if err != nil {
			return fmt.Errorf("failed to list tracked files: %w", err)
		}

		return runStepsAndReport(cmd.Context(), runSteps(defs, cfg, files), runFormat, guideEnabled(cfg))
	},
}

func init() {
	runCmd.Short = i18n.T("cmd.run.short", nil)
	runCmd.Long = i18n.T("cmd.run.long", nil)
	runCmd.Flags().StringVar(&runFormat, "format", "text", i18n.T("flag.format", nil))
	runCmd.Flags().StringSliceVar(&runOnly, "only", nil, i18n.T("flag.only", nil))
	rootCmd.AddCommand(runCmd)
}
