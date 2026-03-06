package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
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

		var allErrs []string

		// 바이너리 파일 감지
		binErrs, err := checker.RunBinaryFiles(cfg)
		if err != nil {
			return fmt.Errorf("failed to check binary files: %w", err)
		}
		allErrs = append(allErrs, binErrs...)

		// 인코딩 검사 (UTF-8)
		encErrs, err := checker.RunEncoding(cfg)
		if err != nil {
			return fmt.Errorf("failed to check encoding: %w", err)
		}
		allErrs = append(allErrs, encErrs...)

		// 데이터 파일 lint (YAML, JSON, XML)
		lintErrs, err := checker.RunLint(cfg)
		if err != nil {
			return fmt.Errorf("failed to check lint: %w", err)
		}
		allErrs = append(allErrs, lintErrs...)

		// .editorconfig 검사
		ecErrs, err := checker.RunEditorConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to check editorconfig: %w", err)
		}
		allErrs = append(allErrs, ecErrs...)

		// 주석 언어 검사 (전체 파일)
		langErrs, err := checker.RunCommentLanguage(cfg)
		if err != nil {
			return fmt.Errorf("failed to check comment language: %w", err)
		}
		allErrs = append(allErrs, langErrs...)

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
