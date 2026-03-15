package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/config/schema"
	"github.com/zcube/commit-checker/internal/i18n"
)

var migrateDryRun bool

var migrateCmd = &cobra.Command{
	Use: "migrate",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(configFile)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s", i18n.T("migrate.file_not_found", map[string]any{
					"Path": configFile,
				}))
			}
			return err
		}

		result, err := schema.Migrate(data)
		if err != nil {
			return fmt.Errorf("%s", i18n.T("migrate.failed", map[string]any{
				"Error": err.Error(),
			}))
		}

		if result.DetectedVersion == schema.VersionCurrent {
			fmt.Println(i18n.T("migrate.already_current", map[string]any{
				"Path": configFile,
			}))
			return nil
		}

		fmt.Println(i18n.T("migrate.detected_version", map[string]any{
			"Version": result.DetectedVersion,
		}))
		for _, desc := range result.Applied {
			fmt.Println(i18n.T("migrate.change", map[string]any{
				"Desc": desc,
			}))
		}

		if migrateDryRun {
			fmt.Println(i18n.T("migrate.dry_run_header", nil))
			fmt.Print(string(result.Data))
			return nil
		}

		if err := os.WriteFile(configFile, result.Data, 0644); err != nil { //nolint:gosec
			return fmt.Errorf("%s", i18n.T("migrate.save_failed", map[string]any{"Error": err.Error()}))
		}
		fmt.Println(i18n.T("migrate.saved", map[string]any{
			"Path": configFile,
		}))
		return nil
	},
}

func init() {
	migrateCmd.Short = i18n.T("cmd.migrate.short", nil)
	migrateCmd.Long = i18n.T("cmd.migrate.long", nil)
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, i18n.T("flag.migrate_dry_run", nil))
	rootCmd.AddCommand(migrateCmd)
}
