package cmd

import (
	"context"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/progress"
)

// runCheckFn: run 커맨드 검사 함수 시그니처.
// files 는 커맨드 진입 시 1회 조회한 추적 파일 목록 (git ls-files).
type runCheckFn func(ctx context.Context, cfg *config.Config, files []string) ([]string, error)

// diffCheckFn: diff 커맨드 검사 함수 시그니처.
// diffs 는 커맨드 진입 시 1회 조회한 staged diff (gitdiff.GetStagedDiff).
type diffCheckFn func(ctx context.Context, cfg *config.Config, diffs []gitdiff.FileDiff) ([]string, error)

// checkStepDef: run/diff 커맨드가 공유하는 검사 step 정의.
// runFn 이 nil 이면 diff 전용, diffFn 이 nil 이면 run 전용 step 이다.
type checkStepDef struct {
	nameKey  string // step 이름 i18n 키
	category string // 기계가 읽을 수 있는 카테고리 (JSON 출력 등)
	runFn    runCheckFn
	diffFn   diffCheckFn
}

// checkStepDefs: 전체 검사 step 레지스트리.
// 새 검사를 여기에 추가하면 run/diff 커맨드 양쪽에 자동으로 반영된다.
var checkStepDefs = []checkStepDef{
	{
		nameKey:  "step.binary_detection",
		category: "binary",
		runFn:    checker.RunBinaryFiles,
		diffFn: func(ctx context.Context, cfg *config.Config, _ []gitdiff.FileDiff) ([]string, error) {
			// staged diff 텍스트 대신 git numstat 으로 바이너리를 판별하므로 diffs 를 받지 않음
			return checker.CheckBinaryFiles(ctx, cfg)
		},
	},
	{
		nameKey:  "step.encoding_check",
		category: "encoding",
		runFn:    checker.RunEncoding,
		diffFn: func(ctx context.Context, cfg *config.Config, _ []gitdiff.FileDiff) ([]string, error) {
			return checker.CheckEncoding(ctx, cfg)
		},
	},
	{
		nameKey:  "step.unicode_check",
		category: "unicode",
		runFn:    checker.RunUnicode,
		diffFn: func(ctx context.Context, cfg *config.Config, _ []gitdiff.FileDiff) ([]string, error) {
			return checker.CheckUnicode(ctx, cfg)
		},
	},
	{
		nameKey:  "step.lint_check",
		category: "lint",
		runFn:    checker.RunLint,
		diffFn: func(ctx context.Context, cfg *config.Config, _ []gitdiff.FileDiff) ([]string, error) {
			return checker.CheckLint(ctx, cfg)
		},
	},
	{
		nameKey:  "step.editorconfig_check",
		category: "editorconfig",
		runFn:    checker.RunEditorConfig,
		diffFn: func(ctx context.Context, cfg *config.Config, _ []gitdiff.FileDiff) ([]string, error) {
			return checker.CheckEditorConfig(ctx, cfg)
		},
	},
	{
		nameKey:  "step.comment_language_check",
		category: "comment_language",
		runFn:    checker.RunCommentLanguage,
		diffFn:   checker.CheckDiff,
	},
	{
		// diff 전용: 커스텀 규칙은 staged diff 의 추가된 줄에만 적용
		nameKey:  "step.custom_rules_check",
		category: "custom_rules",
		diffFn:   checker.CheckDiffCustomRules,
	},
	{
		// diff 전용: 보호 경로는 staged 변경(추가·수정·삭제)을 전부 차단
		nameKey:  "step.protected_paths_check",
		category: "protected_paths",
		diffFn:   checker.CheckProtectedPaths,
	},
	{
		// diff 전용: append-only 는 비교 기준(from) ↔ staged 비교가 필요
		nameKey:  "step.append_only_check",
		category: "append_only",
		diffFn:   checker.CheckAppendOnly,
	},
	{
		nameKey:  "step.cache_dir_check",
		category: "cache_dir",
		runFn: func(ctx context.Context, cfg *config.Config, _ []string) ([]string, error) {
			// 추적 파일 목록 대신 디렉터리 스캔으로 검사하므로 files 를 받지 않음
			return checker.CheckCacheDirCommitted(ctx, cfg)
		},
		diffFn: checker.CheckCacheDirStaged,
	},
}

// runSteps: run 커맨드용 progress.Step 목록을 생성.
// files 는 커맨드 진입 시 1회 조회한 추적 파일 목록을 각 step 에 주입한다.
func runSteps(cfg *config.Config, files []string) []progress.Step {
	steps := make([]progress.Step, 0, len(checkStepDefs))
	for _, def := range checkStepDefs {
		if def.runFn == nil {
			continue
		}
		fn := def.runFn
		steps = append(steps, progress.Step{
			Name:     i18n.T(def.nameKey, nil),
			Category: def.category,
			Fn: func(ctx context.Context) ([]string, error) {
				return fn(ctx, cfg, files)
			},
		})
	}
	return steps
}

// diffSteps: diff 커맨드용 progress.Step 목록을 생성.
// diffs 는 커맨드 진입 시 1회 조회한 staged diff 를 각 step 에 주입한다.
func diffSteps(cfg *config.Config, diffs []gitdiff.FileDiff) []progress.Step {
	steps := make([]progress.Step, 0, len(checkStepDefs))
	for _, def := range checkStepDefs {
		if def.diffFn == nil {
			continue
		}
		fn := def.diffFn
		steps = append(steps, progress.Step{
			Name:     i18n.T(def.nameKey, nil),
			Category: def.category,
			Fn: func(ctx context.Context) ([]string, error) {
				return fn(ctx, cfg, diffs)
			},
		})
	}
	return steps
}
