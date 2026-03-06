package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
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

		var allErrs []string

		// 바이너리 파일 감지
		binErrs, err := checker.CheckBinaryFiles(cfg)
		if err != nil {
			return fmt.Errorf("failed to check binary files: %w", err)
		}
		allErrs = append(allErrs, binErrs...)

		// 인코딩 검사 (UTF-8)
		encErrs, err := checker.CheckEncoding(cfg)
		if err != nil {
			return fmt.Errorf("failed to check encoding: %w", err)
		}
		allErrs = append(allErrs, encErrs...)

		// 데이터 파일 lint (YAML, JSON, XML)
		lintErrs, err := checker.CheckLint(cfg)
		if err != nil {
			return fmt.Errorf("failed to check lint: %w", err)
		}
		allErrs = append(allErrs, lintErrs...)

		// .editorconfig 검사
		ecErrs, err := checker.CheckEditorConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to check editorconfig: %w", err)
		}
		allErrs = append(allErrs, ecErrs...)

		// 주석 언어 검사
		errs, err := checker.CheckDiff(cfg)
		if err != nil {
			return fmt.Errorf("failed to check diff: %w", err)
		}
		allErrs = append(allErrs, errs...)

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
