// prepare_msg.go: git prepare-commit-msg 훅용 커맨드.
// 커밋 메시지 템플릿 끝에 설정 기반 정책 힌트를 # 주석 줄로 추가한다.
// git 이 커밋 시 # 으로 시작하는 줄을 제거하므로 실제 커밋 메시지에는 남지 않는다.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/langdetect"
)

var prepareMsgCmd = &cobra.Command{
	Use:  "prepare-msg <commit-msg-file> [source] [sha]",
	Args: cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := ""
		if len(args) >= 2 {
			source = args[1]
		}
		// 사용자가 이미 메시지를 가진 경우는 힌트를 추가하지 않고 정상 종료:
		//   message: -m/-F 로 메시지 지정, merge/squash: 자동 생성 메시지,
		//   commit: amend 등 기존 커밋 메시지 재사용
		switch source {
		case "message", "merge", "squash", "commit":
			return nil
		}

		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		hint := prepareMsgHint(cfg)
		if hint == "" {
			return nil
		}

		content, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read commit message file: %w", err)
		}

		// 이미 같은 힌트 블록(헤더 기준)이 있으면 중복 추가하지 않음
		if strings.Contains(string(content), prepareMsgHeaderLine()) {
			return nil
		}

		out := string(content)
		if out != "" && !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		out += "\n" + hint
		if err := os.WriteFile(args[0], []byte(out), 0600); err != nil {
			return fmt.Errorf("failed to write commit message file: %w", err)
		}
		return nil
	},
}

// prepareMsgHeaderLine: 힌트 블록의 첫 줄(중복 감지 기준)을 반환.
func prepareMsgHeaderLine() string {
	return "# " + i18n.T("cmd.prepare_msg.hint_header", nil)
}

// prepareMsgHint: 설정에서 활성화된 커밋 메시지 정책의 안내문을 # 주석 블록으로 생성.
// 활성화된 정책이 하나도 없으면 빈 문자열을 반환한다 (헤더만 단독 출력하지 않음).
func prepareMsgHint(cfg *config.Config) string {
	cm := &cfg.CommitMessage
	if !cm.IsEnabled() {
		return ""
	}

	var lines []string
	if cm.ConventionalCommit.IsEnabled() {
		lines = append(lines, i18n.T("cmd.prepare_msg.hint_format", nil))
		lines = append(lines, i18n.T("cmd.prepare_msg.hint_types", map[string]any{
			"Types": strings.Join(cm.ConventionalCommit.GetAllAllowedTypes(), ", "),
		}))
	}
	// language_check 가 켜져 있고 특정 언어를 요구하는 경우에만 안내 ("any" 는 제외)
	if cm.LanguageCheck.IsEnabled() {
		if lang := cm.LanguageCheck.GetLocale(); lang != langdetect.Any {
			lines = append(lines, i18n.T("cmd.prepare_msg.hint_language", map[string]any{
				"Language": languageDisplayName(lang),
			}))
		}
	}
	if cm.IsNoAICoauthor() {
		lines = append(lines, i18n.T("cmd.prepare_msg.hint_coauthor", nil))
	}
	if len(lines) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(prepareMsgHeaderLine() + "\n")
	for _, l := range lines {
		b.WriteString("# " + l + "\n")
	}
	return b.String()
}

// languageDisplayName: langdetect 언어 식별자(korean 등)를 현재 i18n 로케일의
// 표시 이름(한국어/Korean/韓国語/韩语)으로 변환. 번역 키가 없으면 식별자 그대로 반환.
func languageDisplayName(lang string) string {
	key := "lang." + lang
	name := i18n.T(key, nil)
	if name == "["+key+"]" {
		return lang
	}
	return name
}

func init() {
	prepareMsgCmd.Short = i18n.T("cmd.prepare_msg.short", nil)
	prepareMsgCmd.Long = i18n.T("cmd.prepare_msg.long", nil)
	rootCmd.AddCommand(prepareMsgCmd)
}
