package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

var msgFix bool

var msgCmd = &cobra.Command{
	Use:   "msg <commit-msg-file>",
	Short: "Check commit message for violations",
	Long: `Reads the commit message file and checks for:
  - Co-authored-by: trailers (AI tools, configurable)
  - Invisible / non-standard Unicode space characters
  - Ambiguous Unicode characters that look like ASCII
  - Invalid UTF-8 byte sequences
  - Conventional commit format (when enabled)

With --fix, auto-fixable violations (all except language checks) are
corrected in place before the commit proceeds.`,
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
	msgCmd.Flags().BoolVar(&msgFix, "fix", false, "auto-fix violations in place before checking")
	rootCmd.AddCommand(msgCmd)
}
