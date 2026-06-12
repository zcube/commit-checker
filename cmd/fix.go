package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/encoding"
	"github.com/zcube/commit-checker/internal/i18n"
)

var fixDryRun bool

var fixCmd = &cobra.Command{
	Use:  "fix",
	RunE: runFix,
}

func init() {
	fixCmd.Short = i18n.T("cmd.fix.short", nil)
	fixCmd.Long = i18n.T("cmd.fix.long", nil)
	fixCmd.Flags().BoolVar(&fixDryRun, "dry-run", false, i18n.T("flag.fix_dry_run", nil))
	rootCmd.AddCommand(fixCmd)
}

func runFix(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	files, err := getStagedFilesForFix()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Println(i18n.T("cmd.fix.no_staged_files", nil))
		return nil
	}

	var fixedCount int
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, i18n.T("cmd.fix.warn_read_failed", map[string]any{"Path": path, "Error": err.Error()}))
			continue
		}
		if encoding.IsBinary(content) {
			continue
		}

		result := checker.FixFileContent(cfg, string(content))
		if !result.NeedsFixing() {
			continue
		}

		fmt.Printf("%s:\n", path)
		for _, ch := range result.Changes {
			fmt.Printf("  - %s\n", ch)
		}

		if !fixDryRun {
			info, statErr := os.Stat(path)
			perm := os.FileMode(0644)
			if statErr == nil {
				perm = info.Mode().Perm()
			}
			if err := os.WriteFile(path, []byte(result.Fixed), perm); err != nil {
				return fmt.Errorf("writing %s: %w", path, err)
			}
			if err := runGitAdd(path); err != nil {
				return fmt.Errorf("git add %s: %w", path, err)
			}
		}
		fixedCount++
	}

	if fixedCount == 0 {
		fmt.Println(i18n.T("cmd.fix.no_issues", nil))
	} else if fixDryRun {
		fmt.Println(i18n.T("cmd.fix.dry_run_summary", map[string]any{"Count": fixedCount}))
	} else {
		fmt.Println(i18n.T("cmd.fix.fixed_summary", map[string]any{"Count": fixedCount}))
	}
	return nil
}

func getStagedFilesForFix() ([]string, error) {
	out, err := exec.Command("git", "diff", "--staged", "--name-only", "--diff-filter=ACM").Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --staged: %w", err)
	}
	var files []string
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f = strings.TrimSpace(f)
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

func runGitAdd(path string) error {
	return exec.Command("git", "add", path).Run()
}
