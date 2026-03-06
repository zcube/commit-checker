package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Check staged diff for comment language compliance",
	Long: `Reads the current staged changes (git diff --staged) and checks
that comments in supported source files are written in the required language.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		errs, err := checker.CheckDiff(cfg)
		if err != nil {
			return fmt.Errorf("failed to check diff: %w", err)
		}

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
	rootCmd.AddCommand(diffCmd)
}
