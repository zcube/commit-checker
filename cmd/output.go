package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/zcube/commit-checker/internal/progress"
)

// runStepsAndReport: 검사 step 들을 실행하고 결과를 format 에 맞게 출력.
// json 이면 JSON 출력 후 오류가 있으면 exit(1), 아니면 오류를 stderr 로 출력하고
// 요약 라인과 함께 exit(1). ctx 취소 시 실행을 중단하고 에러를 반환.
func runStepsAndReport(ctx context.Context, steps []progress.Step, format string) error {
	result, err := progress.RunWithProgress(ctx, steps, progress.Options{
		Quiet:   globalQuiet || format == "json",
		NoColor: globalNoColor,
	})
	if err != nil {
		return err
	}

	if format == "json" {
		jsonBytes, jsonErr := progress.FormatJSON(result)
		if jsonErr != nil {
			return jsonErr
		}
		fmt.Println(string(jsonBytes))
		if len(result.AllErrors) > 0 {
			os.Exit(1)
		}
		return nil
	}

	// 텍스트 출력
	for _, e := range result.AllErrors {
		fmt.Fprintln(os.Stderr, e)
	}
	if len(result.AllErrors) > 0 {
		if summary := progress.SummaryLine(result.Steps); summary != "" {
			fmt.Fprintln(os.Stderr, summary)
		}
		os.Exit(1)
	}

	return nil
}
