package checker_test

// core.quotePath 회귀 테스트.
// git 기본 설정(quotepath=true)에서는 비ASCII·특수문자 경로가 C-스타일로
// 인용되어 출력되므로, -z(NUL 구분) 미사용 시 한글 파일명이 깨진다.
// 개발 머신의 전역 설정과 무관하게 검증하기 위해 테스트 리포에
// quotepath=true 를 명시한다.

import (
	"slices"
	"strings"
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
)

// newQuotePathRepo: core.quotepath=true 가 명시된 git 리포 생성.
func newQuotePathRepo(t *testing.T) string {
	t.Helper()
	dir := newGitRepo(t)
	gitMust(t, dir, "git", "config", "core.quotepath", "true")
	return dir
}

// TestGetTrackedFiles_한글파일명: ls-files 출력의 한글 경로가 원형으로 반환되어야 함.
func TestGetTrackedFiles_한글파일명(t *testing.T) {
	dir := newQuotePathRepo(t)
	writeFile(t, dir, "한글파일.go", "package main\n")
	writeFile(t, dir, "with\"quote.txt", "x\n")
	gitMust(t, dir, "git", "add", ".")
	gitMust(t, dir, "git", "commit", "-m", "init")

	files, err := checker.GetTrackedFiles()
	if err != nil {
		t.Fatalf("GetTrackedFiles error: %v", err)
	}
	for _, want := range []string{"한글파일.go", "with\"quote.txt"} {
		if !slices.Contains(files, want) {
			t.Errorf("GetTrackedFiles()에 %q 가 원형으로 없음: %v", want, files)
		}
	}
	for _, f := range files {
		if strings.HasPrefix(f, "\"") {
			t.Errorf("C-스타일 인용된 경로가 반환됨: %q", f)
		}
	}
}

// TestGetStagedDiff_한글파일명: diff 헤더의 한글 경로가 원형으로 파싱되어야 함.
func TestGetStagedDiff_한글파일명(t *testing.T) {
	dir := newQuotePathRepo(t)
	writeFile(t, dir, "한글문서.md", "# 제목\n")
	gitMust(t, dir, "git", "add", ".")

	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff error: %v", err)
	}
	found := false
	for _, d := range diffs {
		if d.Path == "한글문서.md" {
			found = true
		}
		if strings.HasPrefix(d.Path, "\"") {
			t.Errorf("C-스타일 인용된 경로가 파싱됨: %q", d.Path)
		}
	}
	if !found {
		t.Errorf("diff 에서 한글 경로를 원형으로 찾지 못함: %+v", diffs)
	}
}

// TestGetStagedDiff_따옴표파일명: git 은 `"` 포함 경로를 core.quotepath 설정과
// 무관하게 항상 C-스타일로 인용하므로, GetStagedDiff 의 quotepath=false 주입만으로는
// 부족하고 ParseDiff 의 unquote 처리가 필요하다.
func TestGetStagedDiff_따옴표파일명(t *testing.T) {
	dir := newQuotePathRepo(t)
	writeFile(t, dir, "with\"quote.txt", "x\n")
	gitMust(t, dir, "git", "add", ".")

	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff error: %v", err)
	}
	found := false
	for _, d := range diffs {
		if d.Path == "with\"quote.txt" {
			found = true
		}
		if strings.HasPrefix(d.Path, "\"") {
			t.Errorf("C-스타일 인용된 경로가 파싱됨: %q", d.Path)
		}
	}
	if !found {
		t.Errorf("diff 에서 따옴표 포함 경로를 원형으로 찾지 못함: %+v", diffs)
	}
}

// TestCheckAppendOnly_한글파일명_FilenameOrder: ls-tree 기반 파일명 순서 검사가
// 한글 파일명에서도 기존 파일을 인식해야 함.
func TestCheckAppendOnly_한글파일명_FilenameOrder(t *testing.T) {
	dir := newQuotePathRepo(t)
	writeFile(t, dir, "db/migrations/002_한글마이그레이션.sql", "create table b;\n")
	gitMust(t, dir, "git", "add", ".")
	gitMust(t, dir, "git", "commit", "-m", "init")

	// 기존 002 보다 앞서는 001 추가 → 순서 위반이 감지되어야 함
	writeFile(t, dir, "db/migrations/001_새파일.sql", "create table a;\n")
	gitMust(t, dir, "git", "add", ".")

	cfg := &config.Config{}
	cfg.AppendOnly.Enabled = true
	cfg.AppendOnly.Paths = []string{"db/migrations/**"}

	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff error: %v", err)
	}
	errs, err := checker.CheckAppendOnly(t.Context(), cfg, diffs)
	if err != nil {
		t.Fatalf("CheckAppendOnly error: %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("한글 기존 파일과의 순서 위반이 감지되어야 함, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0], "001_새파일.sql") {
		t.Errorf("위반 메시지에 새 파일 경로가 없음: %q", errs[0])
	}
}

// TestRunLint_한글파일명_StagedFiles: getStagedFiles 경유 lint 검사가
// 한글 파일명의 구문 오류를 찾아야 함 (경로가 깨지면 파일을 못 읽어 누락됨).
func TestRunLint_한글파일명_StagedFiles(t *testing.T) {
	dir := newQuotePathRepo(t)
	writeFile(t, dir, "한글설정.json", "{invalid json,,}\n")
	gitMust(t, dir, "git", "add", ".")

	cfg := &config.Config{}
	cfg.Lint.Enabled = truePtr()

	errs, err := checker.CheckLint(t.Context(), cfg)
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "한글설정.json") {
			found = true
		}
	}
	if !found {
		t.Errorf("한글 파일명의 lint 위반이 감지되어야 함, got: %v", errs)
	}
}
