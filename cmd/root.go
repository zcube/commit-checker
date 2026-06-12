package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/logger"
	"github.com/zcube/commit-checker/internal/version"
)

var configFile string
var globalQuiet bool
var globalNoColor bool
var globalNoGuide bool
var globalRequireConfig bool

var rootCmd = &cobra.Command{
	Use:          "commit-checker",
	SilenceUsage: true, // 에러 발생 시 Usage 출력 억제 (RunE 에러는 에러 메시지만 출력)
}

func Execute() {
	// SIGINT(Ctrl-C)/SIGTERM 시 진행 중인 검사를 취소할 수 있도록 시그널 연동 context 생성
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// os.Exit 은 여기 한 곳으로 중앙화: RunE 는 errSilentExit 등 에러를 반환만 하고,
	// 종료 코드 결정은 Execute 가 담당한다.
	if err := fang.Execute(ctx, rootCmd,
		fang.WithVersion(version.Version),
		fang.WithCommit(version.Commit),
		fang.WithErrorHandler(silentErrorHandler),
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
	rootCmd.PersistentFlags().BoolVar(&globalNoGuide, "no-guide", false, i18n.T("flag.no_guide", nil))
	rootCmd.PersistentFlags().BoolVar(&globalRequireConfig, "require-config", false, i18n.T("flag.require_config", nil))
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logger.SetQuiet(globalQuiet)
		if globalNoColor {
			logger.SetNoColor(true)
		}
		return nil
	}
}

// requireConfigSkip: --require-config 가 켜져 있고 프로젝트 설정 파일(configFile)이
// 존재하지 않으면 true 를 반환합니다. 훅 진입 커맨드(run/diff/msg/push)는 이 경우
// 아무 출력 없이 성공 종료하여, 전역 훅 설치 시 설정 파일이 있는 리포만 검사(opt-in)합니다.
func requireConfigSkip() bool {
	if !globalRequireConfig {
		return false
	}
	_, err := os.Stat(configFile)
	return err != nil
}
