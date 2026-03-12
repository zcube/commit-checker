package config

import (
	"fmt"
	"regexp"
	"strings"

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
		return fmt.Errorf("설정 파일 구문 오류 (%s):\n  %s", cfgPath, err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "설정 파일 오류 (%s):\n", cfgPath)

	for _, e := range te.Errors {
		m := reUnmarshalErr.FindStringSubmatch(e)
		if m == nil {
			fmt.Fprintf(&sb, "  - %s\n", e)
			continue
		}
		line, yamlType, value, goType := m[1], m[2], m[3], m[4]

		if hint, found := typeHints[goType]; found {
			fmt.Fprintf(&sb, "  - %s행: '%s' 필드에 %s(%q) 대신 객체 형식이 필요합니다.\n",
				line, hint.field, yamlType, value)
			fmt.Fprintf(&sb, "    올바른 형식 예시:\n")
			for exLine := range strings.SplitSeq(hint.example, "\n") {
				fmt.Fprintf(&sb, "      %s\n", exLine)
			}
		} else {
			fmt.Fprintf(&sb, "  - %s행: %s 타입에 %s 값(%q)을 사용할 수 없습니다.\n",
				line, goType, yamlType, value)
		}
	}

	return fmt.Errorf("%s", strings.TrimRight(sb.String(), "\n"))
}
