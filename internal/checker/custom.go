package checker

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/pathutil"
)

// CheckMsgCustomRules: 커밋 메시지에 커스텀 정규식 규칙을 적용합니다.
// required 규칙은 전체 메시지에서 패턴을 찾지 못하면 오류.
// forbidden(기본) 규칙은 각 줄에서 패턴을 찾으면 오류.
func CheckMsgCustomRules(content string, rules []config.CustomRule) []string {
	var errs []string
	for _, rule := range rules {
		if rule.Pattern == "" {
			continue
		}
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			continue
		}
		msg := rule.Message
		if msg == "" {
			msg = fmt.Sprintf("pattern: %s", rule.Pattern)
		}

		if rule.Required {
			if !re.MatchString(content) {
				errs = append(errs, i18n.T("msg.custom_rule_required", map[string]interface{}{
					"Name":    rule.Name,
					"Message": msg,
				}))
			}
		} else {
			lines := strings.Split(content, "\n")
			for i, line := range lines {
				if re.MatchString(line) {
					errs = append(errs, i18n.T("msg.custom_rule_forbidden", map[string]interface{}{
						"Line":    i + 1,
						"Name":    rule.Name,
						"Message": msg,
					}))
				}
			}
		}
	}
	return errs
}

// CheckDiffCustomRules: 스테이지된 diff의 추가된 줄에 커스텀 정규식 규칙을 적용합니다.
// forbidden 규칙만 지원하며, 추가된 줄에서 패턴을 찾으면 오류를 반환합니다.
func CheckDiffCustomRules(ctx context.Context, cfg *config.Config) ([]string, error) {
	rules := cfg.CustomRules.Diff
	if len(rules) == 0 {
		return nil, nil
	}

	type compiledRule struct {
		rule config.CustomRule
		re   *regexp.Regexp
	}
	var compiled []compiledRule
	for _, rule := range rules {
		if rule.Pattern == "" || rule.Required {
			continue
		}
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			continue
		}
		compiled = append(compiled, compiledRule{rule: rule, re: re})
	}
	if len(compiled) == 0 {
		return nil, nil
	}

	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		return nil, err
	}

	ignorePatterns := cfg.Exceptions.GlobalIgnore

	var errs []string
	for _, diff := range diffs {
		// 취소 시 남은 파일 검사를 중단
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if diff.IsDeleted {
			continue
		}
		if pathutil.MatchesAny(diff.Path, ignorePatterns) {
			continue
		}
		if len(diff.AddedLines) == 0 {
			continue
		}

		stagedContent, err := gitdiff.GetStagedContent(diff.Path)
		if err != nil {
			continue
		}
		lines := strings.Split(stagedContent, "\n")

		for lineNum := range diff.AddedLines {
			if lineNum < 1 || lineNum > len(lines) {
				continue
			}
			line := lines[lineNum-1]
			for _, cr := range compiled {
				if cr.re.MatchString(line) {
					msg := cr.rule.Message
					if msg == "" {
						msg = fmt.Sprintf("pattern: %s", cr.rule.Pattern)
					}
					errs = append(errs, i18n.T("diff.custom_rule_forbidden", map[string]interface{}{
						"Path":    diff.Path,
						"Line":    lineNum,
						"Name":    cr.rule.Name,
						"Message": msg,
					}))
				}
			}
		}
	}
	return errs, nil
}
