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
	// 환경 변수에서 로케일을 감지하여 i18n 초기화
	i18n.Init("")

	rootCmd.Short = i18n.T("cmd.root.short", nil)
	rootCmd.Long = i18n.T("cmd.root.long", nil)

	rootCmd.PersistentFlags().StringVar(&configFile, "config", ".commit-checker.yml", i18n.T("flag.config", nil))
	rootCmd.PersistentFlags().BoolVarP(&globalQuiet, "quiet", "q", false, i18n.T("flag.quiet", nil))
	rootCmd.PersistentFlags().BoolVar(&globalNoColor, "no-color", false, i18n.T("flag.no_color", nil))
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logger.SetQuiet(globalQuiet)
		if globalNoColor {
			logger.SetNoColor(true)
		}
		return nil
	}
}
