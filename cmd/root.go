package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/i18n"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:          "commit-checker",
	Short:        "Git commit checker for code quality and standards",
	Long:         "A tool to enforce commit message and code comment standards in git repositories.\nIntegrates with lefthook, husky, and other git hook managers.",
	SilenceUsage: true, // 에러 발생 시 Usage 출력 억제 (RunE 에러는 에러 메시지만 출력)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// cobra 가 이미 stderr 에 에러를 출력하므로 중복 출력하지 않음
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", ".commit-checker.yml", "config file path")
	// 환경 변수에서 로케일을 감지하여 i18n 초기화
	i18n.Init("")
}
