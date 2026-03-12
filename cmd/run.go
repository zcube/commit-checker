package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/progress"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Check all tracked files for policy compliance",
	Long: `Reads all files tracked by git (git ls-files) and checks:
- binary file detection
- file encoding (UTF-8)
- data file lint (YAML, JSON/JSON5, XML)
- .editorconfig compliance
- comment language compliance

Unlike 'diff', this command checks all files regardless of staged state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		steps := []progress.Step{
			{Name: "Binary file detection", Fn: func() ([]string, error) { return checker.RunBinaryFiles(cfg) }},
			{Name: "Encoding check (UTF-8)", Fn: func() ([]string, error) { return checker.RunEncoding(cfg) }},
			{Name: "Unicode character check", Fn: func() ([]string, error) { return checker.RunUnicode(cfg) }},
			{Name: "Data file lint (YAML, JSON, XML)", Fn: func() ([]string, error) { return checker.RunLint(cfg) }},
			{Name: "EditorConfig compliance", Fn: func() ([]string, error) { return checker.RunEditorConfig(cfg) }},
			{Name: "Comment language check", Fn: func() ([]string, error) { return checker.RunCommentLanguage(cfg) }},
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
	rootCmd.AddCommand(runCmd)
}
