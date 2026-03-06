package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// defaultConfig: 기본 설정 파일 내용 (주석 포함).
const defaultConfig = `# yaml-language-server: $schema=./.commit-checker.schema.json
# commit-checker 설정 파일
# 모든 필드는 선택 사항이며, 생략 시 아래 기본값이 적용됩니다.

comment_language:
  # enabled: false로 설정하면 diff 주석 검사를 완전히 건너뜁니다.
  # 기본값: true
  enabled: true

  # required_language: 주석에 사용해야 하는 자연어.
  # 값: korean | english | japanese | chinese | any
  # 기본값: korean
  required_language: korean

  # min_length: 언어 검사를 수행할 최소 글자 수.
  # 기본값: 5
  min_length: 5

  # check_mode: 검사할 주석 범위.
  #   diff — 추가된 줄의 주석만 검사 (기본값, 빠름)
  #   full — 스테이지된 파일의 모든 주석 검사
  check_mode: diff

  # extensions: 검사할 파일 확장자 목록.
  extensions:
    - .go
    - .ts
    - .tsx
    - .js
    - .jsx
    - .mjs
    - .java
    - .kt
    - .py
    - .c
    - .cpp
    - .cs
    - .swift
    - .rs

  # ignore_files: 주석 언어 검사를 건너뛸 파일 glob 패턴.
  # ignore_files:
  #   - "**/*_test.go"

commit_message:
  # no_coauthor: AI 도구의 Co-authored-by: 트레일러를 차단 (기본값: true).
  # 내장 AI 이메일 패턴(Copilot, Claude, Cursor, Codeium 등)과 일치하는 줄 거부.
  # 일반 사람 공동 작업자는 영향 없음.
  no_coauthor: true

  # coauthor_remove_emails: 내장 AI 패턴 외에 추가로 제거할 이메일 glob 패턴.
  # coauthor_remove_emails:
  #   - "*@myai.internal"

  # no_unicode_spaces: 비표준 유니코드 공백 문자 금지 (기본값: true).
  no_unicode_spaces: true

  # no_ambiguous_chars: ASCII와 유사하지만 다른 유니코드 문자 금지 (기본값: true).
  no_ambiguous_chars: true

  # no_bad_runes: 잘못된 UTF-8 바이트 시퀀스 금지 (기본값: true).
  no_bad_runes: true

  # locale: 모호한 문자 감지 로케일 (기본값: ko).
  # 값: ko | ja | zh-hans | zh-hant | ru | _default
  locale: ko

  # conventional_commit: Conventional Commits 형식 강제.
  # 형식: <type>[(<scope>)][!]: <description>
  conventional_commit:
    # enabled: true로 설정하면 커밋 메시지 형식을 검사합니다.
    # 기본값: false
    enabled: false

    # types: 허용된 커밋 타입 목록.
    # 기본값: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert
    # types:
    #   - feat
    #   - fix
    #   - docs
    #   - chore

    # require_scope: 스코프 필수 여부 (기본값: false).
    # require_scope: false

    # allow_merge_commits: "Merge ..." 커밋 건너뜀 (기본값: true).
    # allow_merge_commits: true

    # allow_revert_commits: "Revert ..." 커밋 건너뜀 (기본값: true).
    # allow_revert_commits: true

  # language_check: 커밋 메시지 본문 자연어 검사.
  language_check:
    # enabled: true로 설정하면 커밋 메시지 언어를 검사합니다.
    # 기본값: false
    enabled: false

    # required_language: 커밋 메시지에 사용해야 하는 언어.
    # 값: korean | english | japanese | chinese | any
    # 기본값: korean
    required_language: korean

    # min_length: 언어 검사를 수행할 최소 글자 수 (기본값: 5).
    min_length: 5

    # skip_prefixes: 언어 검사를 건너뛸 제목 줄 접두사 목록.
    skip_prefixes:
      - "Merge"
      - "Revert"
      - "fixup!"
      - "squash!"
`

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "기본 설정 파일 생성",
	Long: `현재 디렉토리에 기본 .commit-checker.yml 설정 파일을 생성합니다.

파일이 이미 존재하는 경우 --force 플래그를 사용하여 덮어쓸 수 있습니다.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := configFile
		if !initForce {
			if _, err := os.Stat(target); err == nil {
				return fmt.Errorf("%s 파일이 이미 존재합니다. 덮어쓰려면 --force 플래그를 사용하세요", target)
			}
		}
		if err := os.WriteFile(target, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("설정 파일 생성 실패: %w", err)
		}
		fmt.Printf("%s 파일이 생성되었습니다.\n", target)
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "기존 파일 덮어쓰기")
	rootCmd.AddCommand(initCmd)
}
