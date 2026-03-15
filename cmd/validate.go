package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
)

var validateCmd = &cobra.Command{
	Use: "validate",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}
		warnings := config.Validate(cfg, configFile)
		if len(warnings) == 0 {
			fmt.Println(i18n.T("validate.config_valid", nil))
			return nil
		}
		for _, w := range warnings {
			fmt.Fprintln(os.Stderr, w)
		}
		return fmt.Errorf("%s", i18n.T("validate.warnings_found", map[string]any{"Count": len(warnings)}))
	},
}

func init() {
	validateCmd.Short = i18n.T("cmd.validate.short", nil)
	validateCmd.Long = i18n.T("cmd.validate.long", nil)
	rootCmd.AddCommand(validateCmd)
}
