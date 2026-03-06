package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

var (
	fixRange  string
	fixMine   bool
	fixDryRun bool
)

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Rewrite git history to fix commit message violations",
	Long: `Scans commits for message policy violations and rewrites them automatically.

Auto-fixable violations:
  - Co-authored-by: trailers (removed)
  - Invisible / non-standard Unicode space characters (replaced or removed)
  - Ambiguous Unicode characters that look like ASCII (replaced with ASCII)
  - Invalid UTF-8 byte sequences (removed)

Language violations in commit bodies are reported but NOT auto-fixed.

Examples:
  commit-checker fix --dry-run                   # preview fixes for all commits
  commit-checker fix --range HEAD~5..HEAD        # fix last 5 commits
  commit-checker fix --mine --dry-run            # preview fixes for your commits
  commit-checker fix --range main..HEAD          # fix commits not yet in main`,
	RunE: runFix,
}

func init() {
	fixCmd.Flags().StringVar(&fixRange, "range", "", "git revision range to scan (e.g. HEAD~5..HEAD, main..HEAD)")
	fixCmd.Flags().BoolVar(&fixMine, "mine", false, "only scan commits authored by the current git user")
	fixCmd.Flags().BoolVar(&fixDryRun, "dry-run", false, "show what would be changed without modifying history")
	rootCmd.AddCommand(fixCmd)
}

func runFix(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Determine the revision range to scan.
	revRange, err := resolveRange(fixRange, fixMine)
	if err != nil {
		return err
	}

	// Collect commits (SHA + message) in the range.
	commits, err := listCommits(revRange)
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		fmt.Println("No commits found in range.")
		return nil
	}

	// Compute fixes.
	var fixes []commitFix
	var langIssues []string

	for _, c := range commits {
		result := checker.FixMsg(cfg, c.message)
		if result.NeedsFixing() {
			fixes = append(fixes, commitFix{sha: c.sha, result: result})
		}
		// Also report language violations (not auto-fixable).
		msgErrs := checker.CheckMsg(cfg, c.message)
		for _, e := range msgErrs {
			langIssues = append(langIssues, fmt.Sprintf("  %s: %s", c.sha[:12], e))
		}
	}

	// Report.
	if len(fixes) == 0 {
		fmt.Printf("Checked %d commits — no auto-fixable violations found.\n", len(commits))
	} else {
		fmt.Printf("Found %d commit(s) with auto-fixable violations:\n\n", len(fixes))
		for _, f := range fixes {
			sha := f.sha
		if len(sha) > 12 {
			sha = sha[:12]
		}
		fmt.Printf("commit %s\n", sha)
			for _, ch := range f.result.Changes {
				fmt.Printf("  - %s\n", ch)
			}
			if fixDryRun {
				fmt.Printf("  before: %s\n", firstLine(f.result.Original))
				fmt.Printf("  after:  %s\n", firstLine(f.result.Fixed))
			}
			fmt.Println()
		}
	}

	if len(langIssues) > 0 {
		fmt.Printf("Language violations (manual fix required):\n")
		for _, e := range langIssues {
			fmt.Println(e)
		}
		fmt.Println()
	}

	if fixDryRun || len(fixes) == 0 {
		if fixDryRun && len(fixes) > 0 {
			fmt.Println("(dry-run: no changes written)")
		}
		return nil
	}

	// Apply fixes using git filter-branch.
	return applyFixes(revRange, fixes)
}

type commitFix struct {
	sha    string
	result checker.FixResult
}

// applyFixes rewrites commit messages using git filter-branch.
// Fixed messages are stored as files in a temp directory; the filter script
// reads them by $GIT_COMMIT so special characters are handled safely.
func applyFixes(revRange string, fixes []commitFix) error {
	tmpDir, err := os.MkdirTemp("", "commit-checker-fix-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write each fixed message as a file named by SHA.
	for _, f := range fixes {
		msgPath := filepath.Join(tmpDir, f.sha)
		if err := os.WriteFile(msgPath, []byte(f.result.Fixed), 0600); err != nil {
			return fmt.Errorf("writing fixed message for %s: %w", f.sha[:12], err)
		}
	}

	// Shell script: if a fixed message file exists for this commit, use it; else pass through.
	script := fmt.Sprintf(`#!/bin/sh
fix="%s/$GIT_COMMIT"
if [ -f "$fix" ]; then
  cat "$fix"
else
  cat
fi
`, tmpDir)

	scriptPath := filepath.Join(tmpDir, "fix-msg.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0700); err != nil {
		return fmt.Errorf("writing filter script: %w", err)
	}

	fmt.Printf("Rewriting %d commit(s) with git filter-branch...\n", len(fixes))

	gitArgs := []string{"filter-branch", "-f", "--msg-filter", scriptPath, revRange}
	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git filter-branch failed: %w", err)
	}

	fmt.Println("Done. History rewritten successfully.")
	fmt.Println("Note: if you have already pushed these commits, you will need to force-push.")
	return nil
}

// resolveRange returns the effective git revision range.
func resolveRange(rangeStr string, mine bool) (string, error) {
	if rangeStr != "" && mine {
		return "", fmt.Errorf("--range and --mine cannot be used together")
	}
	if rangeStr != "" {
		return rangeStr, nil
	}
	if mine {
		email, err := gitConfig("user.email")
		if err != nil {
			return "", fmt.Errorf("getting git user.email: %w", err)
		}
		return fmt.Sprintf("--author=%s --all", email), nil
	}
	// Default: entire reachable history
	return "HEAD", nil
}

type commitInfo struct {
	sha     string
	message string
}

// listCommits returns commits reachable from revRange as (sha, full_message) pairs.
// It first fetches all SHAs, then retrieves each commit message individually to
// avoid null-byte or delimiter conflicts in format strings.
func listCommits(revRange string) ([]commitInfo, error) {
	// Step 1: get just the SHA list.
	shaArgs := append([]string{"log", "--format=%H"}, strings.Fields(revRange)...)
	shaOut, err := runGit(shaArgs...)
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	var commits []commitInfo
	for _, sha := range strings.Split(strings.TrimSpace(shaOut), "\n") {
		sha = strings.TrimSpace(sha)
		if sha == "" {
			continue
		}
		// Step 2: get the full commit message for this SHA.
		msg, err := runGit("log", "-1", "--format=%B", sha)
		if err != nil {
			return nil, fmt.Errorf("git log -1 %s: %w", sha[:12], err)
		}
		commits = append(commits, commitInfo{sha: sha, message: msg})
	}
	return commits, nil
}

func gitConfig(key string) (string, error) {
	out, err := runGit("config", "--get", key)
	return strings.TrimSpace(out), err
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	return string(out), err
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}
