package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/progress"
)

// guideEnabled: 개선 가이드 출력 활성화 여부 결정.
// --no-guide 플래그가 켜져 있으면 설정과 무관하게 비활성화.
func guideEnabled(cfg *config.Config) bool {
	return !globalNoGuide && cfg.Guide.IsEnabled()
}

// guideText: 카테고리의 개선 가이드 텍스트를 반환.
// 해당 i18n 키(guide.<category>)가 없으면 빈 문자열을 반환해 그 카테고리는 생략.
func guideText(category string) string {
	key := "guide." + category
	text := i18n.T(key, nil)
	if text == "["+key+"]" {
		return ""
	}
	return text
}

// failedGuides: 위반이 발생한 step 의 카테고리별 가이드 텍스트를 수집.
// step 순서를 유지하고 중복 카테고리는 1회만 포함하며, 가이드 키가 없는 카테고리는 생략.
func failedGuides(steps []progress.StepResult) ([]string, map[string]string) {
	var categories []string
	guides := map[string]string{}
	for _, s := range steps {
		if len(s.Errors) == 0 {
			continue
		}
		cat := s.Category
		if cat == "" {
			continue
		}
		if _, seen := guides[cat]; seen {
			continue
		}
		text := guideText(cat)
		if text == "" {
			continue
		}
		categories = append(categories, cat)
		guides[cat] = text
	}
	return categories, guides
}

// printGuides: 가이드 헤더와 카테고리별 가이드 텍스트를 stderr 로 출력.
func printGuides(categories []string, guides map[string]string) {
	if len(categories) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, i18n.T("guide.header", nil))
	for _, cat := range categories {
		fmt.Fprintf(os.Stderr, "  [%s] %s\n", cat, guides[cat])
	}
}

// printCommitMessageGuide: 커밋 메시지 위반 시 commit_message 가이드를 1회 출력.
// msg/push 커맨드에서 위반 목록 출력 뒤에 사용.
func printCommitMessageGuide() {
	const cat = "commit_message"
	if text := guideText(cat); text != "" {
		printGuides([]string{cat}, map[string]string{cat: text})
	}
}

// runStepsAndReport: 검사 step 들을 실행하고 결과를 format 에 맞게 출력.
// json 이면 JSON 출력 후 오류가 있으면 errSilentExit 반환, 아니면 오류를 stderr 로 출력하고
// 요약 라인과 함께 errSilentExit 반환. ctx 취소 시 실행을 중단하고 에러를 반환.
// withGuide 가 true 면 위반 시 카테고리별 개선 가이드를 함께 출력.
func runStepsAndReport(ctx context.Context, steps []progress.Step, format string, withGuide bool) error {
	result, err := progress.RunWithProgress(ctx, steps, progress.Options{
		Quiet:   globalQuiet || format == "json",
		NoColor: globalNoColor,
	})
	if err != nil {
		return err
	}

	if format == "json" {
		// 가이드는 위반이 있고 활성화된 경우에만 포함 (기존 JSON 소비자 호환)
		var guides map[string]string
		if withGuide && len(result.AllErrors) > 0 {
			_, guides = failedGuides(result.Steps)
		}
		jsonBytes, jsonErr := progress.FormatJSON(result, guides)
		if jsonErr != nil {
			return jsonErr
		}
		fmt.Println(string(jsonBytes))
		if len(result.AllErrors) > 0 {
			return errSilentExit
		}
		return nil
	}

	// 텍스트 출력
	for _, e := range result.AllErrors {
		fmt.Fprintln(os.Stderr, e)
	}
	if len(result.AllErrors) > 0 {
		if total, checks := progress.Summary(result.Steps); total > 0 {
			fmt.Fprintln(os.Stderr, i18n.T("summary.violations", map[string]any{
				"Count":  total,
				"Checks": checks,
			}))
		}
		if withGuide {
			printGuides(failedGuides(result.Steps))
		}
		return errSilentExit
	}

	return nil
}
