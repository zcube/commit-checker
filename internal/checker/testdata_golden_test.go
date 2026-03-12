// commit-checker:file-lang=any
package checker_test

// testdata에 있는 모든 소스 파일에 대해 파서 추출 결과와 언어 체크 결과를 골든 파일과 비교합니다.
// 파서 동작이나 언어 감지 로직이 변경되면 이 테스트가 실패하여 동작 변경을 알립니다.
//
// 골든 파일 업데이트:
//
//	go test ./internal/checker/... -run TestCommentCheckerGolden -update

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/zcube/commit-checker/internal/comment"
	"github.com/zcube/commit-checker/internal/directive"
	"github.com/zcube/commit-checker/internal/langdetect"
)

var updateGolden = flag.Bool("update", false, "골든 파일 업데이트")

func TestCommentCheckerGolden(t *testing.T) {
	testdataDir := filepath.Join("..", "comment", "testdata")
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("testdata 읽기 실패: %v", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var sb strings.Builder
	for _, name := range names {
		path := filepath.Join(testdataDir, name)
		parser := comment.GetParser(path)
		if parser == nil {
			fmt.Fprintf(&sb, "%s: parser=none\n", name)
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("파일 읽기 실패 %s: %v", name, err)
		}

		comments, err := parser.ParseFile(string(content))
		if err != nil {
			fmt.Fprintf(&sb, "%s: parse_error=%v\n", name, err)
			continue
		}

		var numComments, numStrings, numImports int
		for _, c := range comments {
			switch c.Kind {
			case comment.KindComment:
				numComments++
			case comment.KindString:
				numStrings++
			case comment.KindImport:
				numImports++
			}
		}

		states := directive.Analyze(comments, "korean")
		var langErrs []string
		for i, c := range comments {
			state := states[i]
			if state.Skip || c.Kind == comment.KindImport || c.Kind == comment.KindString {
				continue
			}
			text := strings.TrimSpace(c.Text)
			ok, hasContent := langdetect.IsRequiredLanguage(text, state.Language, 5, nil)
			if !hasContent {
				continue
			}
			if !ok {
				detected := langdetect.Dominant(text)
				preview := text
			if len([]rune(preview)) > 40 {
				preview = string([]rune(preview)[:40]) + "..."
			}
			langErrs = append(langErrs, fmt.Sprintf("  line %d: detected=%s text=%q", c.Line, detected, preview))
			}
		}

		langResult := "PASS"
		if len(langErrs) > 0 {
			langResult = "FAIL"
		}
		fmt.Fprintf(&sb, "%s: comments=%d strings=%d imports=%d lang=%s\n",
			name, numComments, numStrings, numImports, langResult)
		for _, e := range langErrs {
			fmt.Fprintln(&sb, e)
		}
	}

	goldenPath := filepath.Join("testdata", "comment_checker.golden")
	got := sb.String()

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("골든 파일 업데이트 완료: %s", goldenPath)
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("골든 파일 없음 (생성하려면: go test ./internal/checker/... -run TestCommentCheckerGolden -update): %v", err)
	}

	if got != string(wantBytes) {
		t.Errorf("골든 파일과 불일치 — 동작이 변경되었습니다.\n"+
			"업데이트하려면: go test ./internal/checker/... -run TestCommentCheckerGolden -update\n\n"+
			"=== 기대값 ===\n%s\n=== 실제값 ===\n%s", wantBytes, got)
	}
}
