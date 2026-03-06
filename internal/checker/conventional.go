package checker

import (
	"regexp"
	"strings"

	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
)

// conventionalPattern: 컨벤셔널 커밋 제목 줄 패턴
// 형식: type[(scope)][!]: description
// 타입에 유니코드 문자도 허용 (로컬라이즈된 타입 지원: 기능, 修正 등)
var conventionalPattern = regexp.MustCompile(`^([^\s(:!]+)(\([^)]*\))?(!)?: .+`)

// checkConventional: 커밋 메시지 제목이 Conventional Commits 명세를 따르는지 검증.
func checkConventional(content string, cfg *config.ConventionalCommitConfig) []string {
	lines := strings.SplitN(strings.TrimRight(content, "\n"), "\n", 2)
	if len(lines) == 0 {
		return nil
	}

	subject := strings.TrimSpace(lines[0])
	if subject == "" {
		return nil
	}

	// Merge 커밋 건너뜀.
	if cfg.IsAllowMergeCommits() && strings.HasPrefix(subject, "Merge ") {
		return nil
	}

	// Revert 커밋 건너뜀.
	if cfg.IsAllowRevertCommits() && strings.HasPrefix(subject, "Revert ") {
		return nil
	}

	// fixup!/squash! 커밋 건너뜀.
	if strings.HasPrefix(subject, "fixup! ") || strings.HasPrefix(subject, "squash! ") {
		return nil
	}

	// 전체 형식 검사.
	m := conventionalPattern.FindStringSubmatch(subject)
	if m == nil {
		return []string{i18n.T("msg.conventional_format_error", map[string]interface{}{
			"Subject": truncate(subject, 80),
		})}
	}

	commitType := m[1]
	scope := m[2]

	// 허용된 타입 목록과 대조 (표준 타입 + 별칭 모두 확인).
	allTypes := cfg.GetAllAllowedTypes()
	typeAllowed := false
	for _, t := range allTypes {
		if strings.EqualFold(commitType, t) {
			typeAllowed = true
			break
		}
	}
	if !typeAllowed {
		return []string{i18n.T("msg.conventional_type_error", map[string]interface{}{
			"Type":  commitType,
			"Types": strings.Join(allTypes, ", "),
		})}
	}

	// 스코프 필수 여부 확인.
	if cfg.IsRequireScope() && (scope == "" || scope == "()") {
		return []string{i18n.T("msg.conventional_scope_required", map[string]interface{}{
			"Subject": truncate(subject, 80),
		})}
	}

	return nil
}
