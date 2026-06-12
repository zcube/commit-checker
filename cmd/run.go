package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
)

var runFormat string

var runCmd = &cobra.Command{
	Use: "run",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// git ls-files 는 커맨드 진입 시 1회만 실행하고 각 검사 step 에 주입
		files, err := checker.GetTrackedFiles()
		if err != nil {
			return fmt.Errorf("failed to list tracked files: %w", err)
		}

		return runStepsAndReport(cmd.Context(), runSteps(cfg, files), runFormat)
	},
}

func init() {
	runCmd.Short = i18n.T("cmd.run.short", nil)
	runCmd.Long = i18n.T("cmd.run.long", nil)
	runCmd.Flags().StringVar(&runFormat, "format", "text", i18n.T("flag.format", nil))
	rootCmd.AddCommand(runCmd)
}
