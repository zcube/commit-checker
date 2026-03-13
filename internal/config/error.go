package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/zcube/commit-checker/internal/i18n"
	"gopkg.in/yaml.v3"
)

// fieldHint: 특정 Go 타입에 대한 사용자 친화적 필드명과 올바른 YAML 예시.
type fieldHint struct {
	field   string
	example string
}

// typeHints: Go 타입명 → 필드 설명 및 올바른 형식 예시.
var typeHints = map[string]fieldHint{
	"config.LintRuleConfig": {
		field: "lint.yaml 또는 lint.xml",
		example: `lint:
  yaml:
    enabled: true
  xml:
    enabled: true`,
	},
	"config.JSONLintConfig": {
		field: "lint.json",
		example: `lint:
  json:
    enabled: true
    allow_json5: false`,
	},
	"config.LintConfig": {
		field: "lint",
		example: `lint:
  enabled: true
  yaml:
    enabled: true`,
	},
	"config.BinaryFileConfig": {
		field: "binary_file",
		example: `binary_file:
  enabled: true`,
	},
	"config.EncodingConfig": {
		field: "encoding",
		example: `encoding:
  enabled: true
  require_utf8: true`,
	},
	"config.EditorConfigConfig": {
		field: "editorconfig",
		example: `editorconfig:
  enabled: true`,
	},
	"config.CommentLanguageConfig": {
		field: "comment_language",
		example: `comment_language:
  enabled: true
  required_language: korean`,
	},
	"config.CommitMessageConfig": {
		field: "commit_message",
		example: `commit_message:
  enabled: true
  no_ai_coauthor: true`,
	},
	"config.ConventionalCommitConfig": {
		field: "commit_message.conventional_commit",
		example: `commit_message:
  conventional_commit:
    enabled: true`,
	},
	"config.LanguageCheckConfig": {
		field: "commit_message.language_check",
		example: `commit_message:
  language_check:
    enabled: true
    required_language: korean`,
	},
}

// reUnmarshalErr: "line N: cannot unmarshal !!TYPE `VALUE` into GOTYPE" 패턴.
var reUnmarshalErr = regexp.MustCompile("line (\\d+): cannot unmarshal !!([^ ]+) `([^`]*)` into (\\S+)")

// formatConfigError: yaml 언마샬 오류를 사람이 읽기 쉬운 메시지로 변환.
func formatConfigError(cfgPath string, err error) error {
	te, ok := err.(*yaml.TypeError)
	if !ok {
		// 구문 오류 등 기타 yaml 오류
		return fmt.Errorf("%s", i18n.T("config.syntax_error", map[string]any{
			"Path": cfgPath, "Error": err.Error(),
		}))
	}

	var sb strings.Builder
	sb.WriteString(i18n.T("config.type_error_header", map[string]any{"Path": cfgPath}))
	sb.WriteString("\n")

	for _, e := range te.Errors {
		m := reUnmarshalErr.FindStringSubmatch(e)
		if m == nil {
			fmt.Fprintf(&sb, "  - %s\n", e)
			continue
		}
		line, yamlType, value, goType := m[1], m[2], m[3], m[4]

		if hint, found := typeHints[goType]; found {
			sb.WriteString(i18n.T("config.type_error_object_required", map[string]any{
				"Line": line, "Field": hint.field, "Type": yamlType, "Value": value,
			}))
			sb.WriteString("\n")
			sb.WriteString(i18n.T("config.type_error_example_header", nil))
			sb.WriteString("\n")
			for exLine := range strings.SplitSeq(hint.example, "\n") {
				sb.WriteString(i18n.T("config.type_error_example_line", map[string]any{"Line": exLine}))
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString(i18n.T("config.type_error_generic", map[string]any{
				"Line": line, "GoType": goType, "Type": yamlType, "Value": value,
			}))
			sb.WriteString("\n")
		}
	}

	return fmt.Errorf("%s", strings.TrimRight(sb.String(), "\n"))
}
