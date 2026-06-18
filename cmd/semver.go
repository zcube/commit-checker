package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/gitversion-go/gitversion"
)

var semverShowVariable string
var semverOutput string

var semverCmd = &cobra.Command{
	Use:           "semver [path]",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		if abs, err := filepath.Abs(target); err == nil {
			target = abs
		}

		vars, err := gitversion.Compute(target)
		if err != nil {
			return fmt.Errorf("%s", i18n.T("cmd.semver.error_calc", map[string]any{"Err": err}))
		}

		if semverShowVariable != "" {
			out, err := vars.ShowVariable(semverShowVariable)
			if err != nil {
				return fmt.Errorf("%s", i18n.T("cmd.semver.error_variable", map[string]any{"Err": err}))
			}
			fmt.Println(out)
			return nil
		}

		switch strings.ToLower(semverOutput) {
		case "json":
			j, err := vars.ToJSON()
			if err != nil {
				return fmt.Errorf("%s", i18n.T("cmd.semver.error_variable", map[string]any{"Err": err}))
			}
			fmt.Println(j)
		case "dot-env", "dotenv":
			fmt.Print(vars.ToDotEnv())
		case "full-semver", "fullsemver":
			fmt.Println(vars.FullSemVer)
		default:
			fmt.Println(vars.SemVer)
		}
		return nil
	},
}

func init() {
	semverCmd.Short = i18n.T("cmd.semver.short", nil)
	semverCmd.Long = i18n.T("cmd.semver.long", nil)
	semverCmd.Flags().StringVarP(&semverShowVariable, "show-variable", "v", "", "단일 변수만 출력 (예: -v FullSemVer)")
	semverCmd.Flags().StringVarP(&semverOutput, "output", "o", "semver", "출력 형식: semver (기본), full-semver, json, dot-env")
	rootCmd.AddCommand(semverCmd)
}
