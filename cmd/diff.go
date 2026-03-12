package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/progress"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Check staged diff for policy compliance",
	Long: `Reads the current staged changes (git diff --staged) and checks:
- binary file detection
- file encoding (UTF-8)
- data file lint (YAML, JSON/JSON5, XML)
- .editorconfig compliance
- comment language compliance`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		steps := []progress.Step{
			{Name: "Binary file detection", Fn: func() ([]string, error) { return checker.CheckBinaryFiles(cfg) }},
			{Name: "Encoding check (UTF-8)", Fn: func() ([]string, error) { return checker.CheckEncoding(cfg) }},
			{Name: "Unicode character check", Fn: func() ([]string, error) { return checker.CheckUnicode(cfg) }},
			{Name: "Data file lint (YAML, JSON, XML)", Fn: func() ([]string, error) { return checker.CheckLint(cfg) }},
			{Name: "EditorConfig compliance", Fn: func() ([]string, error) { return checker.CheckEditorConfig(cfg) }},
			{Name: "Comment language check", Fn: func() ([]string, error) { return checker.CheckDiff(cfg) }},
		}

		allErrs, err := progress.RunWithProgress(steps)
		if err != nil {
			return err
		}

		if len(allErrs) > 0 {
			for _, e := range allErrs {
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
