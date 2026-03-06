package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "버전 정보 출력",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("commit-checker %s\n", version.Version)
		fmt.Printf("  커밋:      %s\n", version.Commit)
		fmt.Printf("  빌드 시각: %s\n", version.BuildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
