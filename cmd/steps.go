package cmd

import (
	"context"
	"errors"
	"strings"

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

// stepDefsFor: 커맨드 종류(run/diff)에서 사용 가능한 step 정의 목록을 반환.
// only 가 비어있지 않으면 지정된 카테고리만 남긴다 (레지스트리 순서 유지).
// 해당 커맨드에서 유효하지 않은 카테고리가 있으면 유효 목록을 포함한 에러를 반환한다.
func stepDefsFor(diffMode bool, only []string) ([]checkStepDef, error) {
	available := make([]checkStepDef, 0, len(checkStepDefs))
	categories := make([]string, 0, len(checkStepDefs))
	for _, def := range checkStepDefs {
		if (diffMode && def.diffFn == nil) || (!diffMode && def.runFn == nil) {
			continue
		}
		available = append(available, def)
		categories = append(categories, def.category)
	}
	if len(only) == 0 {
		return available, nil
	}

	// 요청된 카테고리 검증: checkStepDefs 레지스트리 기반으로 동적 판별
	requested := make(map[string]bool, len(only))
	for _, cat := range only {
		cat = strings.TrimSpace(cat)
		if cat == "" {
			continue
		}
		valid := false
		for _, c := range categories {
			if c == cat {
				valid = true
				break
			}
		}
		if !valid {
			return nil, errors.New(i18n.T("flag.only_invalid", map[string]any{
				"Category": cat,
				"Valid":    strings.Join(categories, ", "),
			}))
		}
		requested[cat] = true
	}

	selected := make([]checkStepDef, 0, len(requested))
	for _, def := range available {
		if requested[def.category] {
			selected = append(selected, def)
		}
	}
	return selected, nil
}

// cfgWithOnlyEnabled: --only 로 선택된 검사의 enabled 설정을 강제로 켠 cfg 복제본을 반환.
// --only 는 "설정과 무관하게 지정 검사만 실행"이 목적이므로 enabled: false 설정을 덮어쓴다.
// internal/config 는 수정하지 않고 cmd 레이어에서 값 복제 후 enabled 필드만 교체한다.
func cfgWithOnlyEnabled(cfg *config.Config, defs []checkStepDef) *config.Config {
	c := *cfg // 얕은 복제: enabled 포인터 필드만 새 값으로 교체하므로 원본에 영향 없음
	on := true
	for _, def := range defs {
		switch def.category {
		case "binary":
			c.BinaryFile.Enabled = &on
		case "encoding":
			// 인코딩 검사는 enabled 와 require_utf8 둘 다 켜져야 동작하므로 함께 강제
			c.Encoding.Enabled = &on
			c.Encoding.RequireUTF8 = &on
		case "unicode":
			// 유니코드 검사의 enabled 게이트는 encoding.enabled 를 공유.
			// 세부 토글(no_invisible_chars 등)은 기능 선택이므로 설정값을 존중한다.
			c.Encoding.Enabled = &on
		case "lint":
			c.Lint.Enabled = &on
		case "editorconfig":
			c.EditorConfig.Enabled = &on
		case "comment_language":
			c.CommentLanguage.Enabled = &on
		case "cache_dir":
			c.CacheDir.Enabled = &on
		case "protected_paths":
			c.ProtectedPaths.Enabled = true
		case "append_only":
			c.AppendOnly.Enabled = true
		case "custom_rules":
			// custom_rules 는 enabled 토글이 없음 (규칙 목록 유무로 동작)
		}
	}
	return &c
}

// runSteps: run 커맨드용 progress.Step 목록을 생성.
// defs 는 stepDefsFor(false, …) 로 선별한 step 정의 목록.
// files 는 커맨드 진입 시 1회 조회한 추적 파일 목록을 각 step 에 주입한다.
func runSteps(defs []checkStepDef, cfg *config.Config, files []string) []progress.Step {
	steps := make([]progress.Step, 0, len(defs))
	for _, def := range defs {
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
// defs 는 stepDefsFor(true, …) 로 선별한 step 정의 목록.
// diffs 는 커맨드 진입 시 1회 조회한 staged diff 를 각 step 에 주입한다.
func diffSteps(defs []checkStepDef, cfg *config.Config, diffs []gitdiff.FileDiff) []progress.Step {
	steps := make([]progress.Step, 0, len(defs))
	for _, def := range defs {
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
