package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
)

var pushRange string

var pushCmd = &cobra.Command{
	Use: "push",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		var commitRanges []string
		if pushRange != "" {
			commitRanges = []string{pushRange}
		} else {
			// stdin이 파이프/파일일 때만 읽기
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				commitRanges = parsePushRanges(os.Stdin)
			}
		}

		if len(commitRanges) == 0 {
			return nil
		}

		var allErrs []string
		for _, r := range commitRanges {
			hashes, listErr := listPushCommitHashes(r)
			if listErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not list commits for range %s, skipping commit check: %v\n", r, listErr)
				continue
			}
			for _, hash := range hashes {
				msg, msgErr := getPushCommitMessage(hash)
				if msgErr != nil {
					fmt.Fprintf(os.Stderr, "warning: could not read commit message for %s, skipping: %v\n", hash[:7], msgErr)
					continue
				}
				errs := checker.CheckMsg(cfg, msg)
				for _, e := range errs {
					allErrs = append(allErrs, fmt.Sprintf("[%s] %s", hash[:7], e))
				}
			}
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

// parsePushRanges: git pre-push 훅 stdin 형식을 파싱하여 커밋 범위 목록을 반환합니다.
// 각 줄 형식: <local ref> <local sha1> <remote ref> <remote sha1>
func parsePushRanges(r io.Reader) []string {
	var ranges []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 4 {
			continue
		}
		localSHA := parts[1]
		remoteSHA := parts[3]

		// 로컬 SHA가 0이면 브랜치 삭제 — 건너뜁니다
		if isPushZeroSHA(localSHA) {
			continue
		}

		if isPushZeroSHA(remoteSHA) {
			// 새 브랜치: 리모트 기본 브랜치에서 분기된 커밋을 검사합니다
			base := findPushRemoteBase()
			if base == "" {
				fmt.Fprintf(os.Stderr, "warning: could not find remote base branch for %s, skipping commit check\n", parts[0])
				continue
			}
			ranges = append(ranges, base+".."+localSHA)
		} else {
			ranges = append(ranges, remoteSHA+".."+localSHA)
		}
	}
	return ranges
}

// isPushZeroSHA: SHA가 40개의 '0'으로 구성된 빈 SHA인지 확인합니다.
func isPushZeroSHA(sha string) bool {
	if len(sha) != 40 {
		return false
	}
	for _, c := range sha {
		if c != '0' {
			return false
		}
	}
	return true
}

// findPushRemoteBase: 업스트림 추적 브랜치 또는 리모트 기본 브랜치를 찾아 반환합니다.
func findPushRemoteBase() string {
	// 현재 브랜치의 업스트림 추적 브랜치 시도
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	// origin HEAD로 기본 브랜치 탐지 (git remote set-head origin -a 필요)
	out, err = exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err == nil {
		// 예: refs/remotes/origin/main 에서 origin/main 추출
		ref := strings.TrimSpace(string(out))
		if after, ok := strings.CutPrefix(ref, "refs/remotes/"); ok {
			return after
		}
	}
	// 폴백: main, master 순으로 확인
	for _, branch := range []string{"origin/main", "origin/master"} {
		if _, err := exec.Command("git", "rev-parse", "--verify", branch).Output(); err == nil {
			return branch
		}
	}
	return ""
}

// listPushCommitHashes: 주어진 범위 내 커밋 해시 목록을 반환합니다.
func listPushCommitHashes(commitRange string) ([]string, error) {
	out, err := exec.Command("git", "log", "--format=%H", commitRange).Output()
	if err != nil {
		return nil, err
	}
	var hashes []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			hashes = append(hashes, line)
		}
	}
	return hashes, nil
}

// getPushCommitMessage: 주어진 커밋 해시의 커밋 메시지를 반환합니다.
func getPushCommitMessage(hash string) (string, error) {
	out, err := exec.Command("git", "show", "-s", "--format=%B", hash).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func init() {
	pushCmd.Short = i18n.T("cmd.push.short", nil)
	pushCmd.Long = i18n.T("cmd.push.long", nil)
	pushCmd.Flags().StringVar(&pushRange, "range", "", i18n.T("flag.push_range", nil))
	rootCmd.AddCommand(pushCmd)
}
