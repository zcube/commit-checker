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
	Use:   "migrate",
	Short: "Migrate config file to the latest schema",
	Long: `Detect the schema version of the config file and migrate it to the latest format.

Old field names are automatically renamed and deprecated fields are removed.
Comments and formatting are preserved during migration.

Use --dry-run to preview changes without modifying the file.`,
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
			return fmt.Errorf("파일 저장 실패: %w", err)
		}
		fmt.Println(i18n.T("migrate.saved", map[string]any{
			"Path": configFile,
		}))
		return nil
	},
}

func init() {
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "preview changes without modifying the file")
	rootCmd.AddCommand(migrateCmd)
}
