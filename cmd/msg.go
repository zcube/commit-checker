package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
)

var msgFix bool

var msgCmd = &cobra.Command{
	Use:  "msg <commit-msg-file>",
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

		msgContent := string(content)

		if msgFix {
			result := checker.FixMsg(cfg, msgContent)
			if result.NeedsFixing() {
				if err := os.WriteFile(args[0], []byte(result.Fixed), 0600); err != nil {
					return fmt.Errorf("failed to write fixed commit message: %w", err)
				}
				msgContent = result.Fixed
			}
		}

		errs := checker.CheckMsg(cfg, msgContent)
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
	msgCmd.Short = i18n.T("cmd.msg.short", nil)
	msgCmd.Long = i18n.T("cmd.msg.long", nil)
	msgCmd.Flags().BoolVar(&msgFix, "fix", false, i18n.T("flag.msg_fix", nil))
	rootCmd.AddCommand(msgCmd)
}
