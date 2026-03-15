package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/version"
)

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("commit-checker %s\n", version.Version)
		fmt.Println(i18n.T("version.commit", map[string]any{"Value": version.Commit}))
		fmt.Println(i18n.T("version.build_time", map[string]any{"Value": version.BuildTime}))
	},
}

func init() {
	versionCmd.Short = i18n.T("cmd.version.short", nil)
	rootCmd.AddCommand(versionCmd)
}
