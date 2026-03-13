package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/config"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	Long:  `Check the configuration file for errors and potential issues.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}
		warnings := config.Validate(cfg, configFile)
		if len(warnings) == 0 {
			fmt.Println("설정 파일이 유효합니다.")
			return nil
		}
		for _, w := range warnings {
			fmt.Fprintln(os.Stderr, w)
		}
		return fmt.Errorf("%d개의 경고가 발견되었습니다", len(warnings))
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
