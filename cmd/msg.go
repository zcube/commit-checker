package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

var msgCmd = &cobra.Command{
	Use:   "msg <commit-msg-file>",
	Short: "Check commit message for violations",
	Long: `Reads the commit message file and checks for:
  - Co-authored-by: trailers (configurable)
  - Unicode space characters used instead of regular spaces (configurable)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		content, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read commit message file: %w", err)
		}

		errs := checker.CheckMsg(cfg, string(content))
		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintln(os.Stderr, e)
			}
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(msgCmd)
}
