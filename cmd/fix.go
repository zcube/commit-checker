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
	"github.com/zcube/commit-checker/internal/i18n"
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

	// 검사할 리비전 범위를 결정합니다.
	revRange, err := resolveRange(fixRange, fixMine)
	if err != nil {
		return err
	}

	// 범위 내의 커밋 목록(SHA + 메시지)을 수집합니다.
	commits, err := listCommits(revRange)
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		fmt.Println(i18n.T("cmd.no_commits_found", nil))
		return nil
	}

	// 수정 사항을 계산합니다.
	var fixes []commitFix
	var langIssues []string

	for _, c := range commits {
		result := checker.FixMsg(cfg, c.message)
		if result.NeedsFixing() {
			fixes = append(fixes, commitFix{sha: c.sha, result: result})
		}
		// 언어 위반 사항도 보고합니다 (자동 수정 불가).
		msgErrs := checker.CheckMsg(cfg, c.message)
		for _, e := range msgErrs {
			langIssues = append(langIssues, fmt.Sprintf("  %s: %s", c.sha[:12], e))
		}
	}

	// 결과를 출력합니다.
	if len(fixes) == 0 {
		fmt.Println(i18n.T("cmd.checked_no_fixes", map[string]any{"Count": len(commits)}))
	} else {
		fmt.Println(i18n.T("cmd.found_fixable", map[string]any{"Count": len(fixes)}))
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
		fmt.Println(i18n.T("cmd.language_violations_header", nil))
		for _, e := range langIssues {
			fmt.Println(e)
		}
		fmt.Println()
	}

	if fixDryRun || len(fixes) == 0 {
		if fixDryRun && len(fixes) > 0 {
			fmt.Println(i18n.T("cmd.dry_run_no_changes", nil))
		}
		return nil
	}

	// git filter-branch 로 수정 사항을 적용합니다.
	return applyFixes(revRange, fixes)
}

type commitFix struct {
	sha    string
	result checker.FixResult
}

// applyFixes 는 git filter-branch 를 사용하여 커밋 메시지를 재작성합니다.
// 수정된 메시지는 임시 디렉터리에 파일로 저장되며, 필터 스크립트가
// $GIT_COMMIT 으로 파일을 읽어 특수 문자를 안전하게 처리합니다.
func applyFixes(revRange string, fixes []commitFix) error {
	tmpDir, err := os.MkdirTemp("", "commit-checker-fix-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	// 수정된 메시지를 SHA 이름의 파일로 각각 저장합니다.
	for _, f := range fixes {
		msgPath := filepath.Join(tmpDir, f.sha)
		if err := os.WriteFile(msgPath, []byte(f.result.Fixed), 0600); err != nil {
			return fmt.Errorf("writing fixed message for %s: %w", f.sha[:12], err)
		}
	}

	// 셸 스크립트: 수정된 메시지 파일이 있으면 사용하고, 없으면 그대로 통과시킵니다.
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

	fmt.Println(i18n.T("cmd.rewriting_commits", map[string]any{"Count": len(fixes)}))

	gitArgs := []string{"filter-branch", "-f", "--msg-filter", scriptPath, revRange}
	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git filter-branch failed: %w", err)
	}

	fmt.Println(i18n.T("cmd.rewrite_done", nil))
	fmt.Println(i18n.T("cmd.force_push_note", nil))
	return nil
}

// resolveRange 는 유효한 git 리비전 범위를 반환합니다.
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
	// 기본값: 전체 도달 가능한 히스토리
	return "HEAD", nil
}

type commitInfo struct {
	sha     string
	message string
}

// listCommits 는 revRange 에서 도달 가능한 커밋을 (sha, 전체 메시지) 쌍으로 반환합니다.
// 먼저 SHA 목록을 가져온 다음 각 커밋 메시지를 개별적으로 조회하여
// 포맷 문자열의 null 바이트나 구분자 충돌을 방지합니다.
func listCommits(revRange string) ([]commitInfo, error) {
	// 1단계: SHA 목록만 가져옵니다.
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
		// 2단계: 해당 SHA 의 전체 커밋 메시지를 가져옵니다.
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
