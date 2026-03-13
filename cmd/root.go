package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/logger"
	"github.com/zcube/commit-checker/internal/version"
)

var configFile string
var globalQuiet bool
var globalNoColor bool

var rootCmd = &cobra.Command{
	Use:          "commit-checker",
	Short:        "Git commit checker for code quality and standards",
	Long:         "A tool to enforce commit message and code comment standards in git repositories.\nIntegrates with lefthook, husky, and other git hook managers.",
	SilenceUsage: true, // 에러 발생 시 Usage 출력 억제 (RunE 에러는 에러 메시지만 출력)
}

func Execute() {
	if err := fang.Execute(context.Background(), rootCmd,
		fang.WithVersion(version.Version),
		fang.WithCommit(version.Commit),
	); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", ".commit-checker.yml", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&globalQuiet, "quiet", "q", false, "suppress progress and log output")
	rootCmd.PersistentFlags().BoolVar(&globalNoColor, "no-color", false, "disable color output")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logger.SetQuiet(globalQuiet)
		if globalNoColor {
			logger.SetNoColor(true)
		}
		return nil
	}
	// 환경 변수에서 로케일을 감지하여 i18n 초기화
	i18n.Init("")
}
