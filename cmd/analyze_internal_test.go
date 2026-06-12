package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

// ---- runAnalyze ---------------------------------------------------------------

func TestRunAnalyze_InRepo(t *testing.T) {
	dir := newTestGitRepo(t)

	// 다양한 언어 파일을 커밋해 언어 감지 대상으로 만든다
	writeTestFile(t, filepath.Join(dir, "main.go"), "package main\n")
	writeTestFile(t, filepath.Join(dir, "app.ts"), "const x = 1;\n")
	writeTestFile(t, filepath.Join(dir, "data.json"), "{}\n")
	writeTestFile(t, filepath.Join(dir, ".golangci.yml"), "version: \"2\"\n")
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "chore: 분석 대상 파일 추가")

	var err error
	out := captureStdout(t, func() { err = runAnalyze() })
	if err != nil {
		t.Fatalf("runAnalyze: %v", err)
	}
	if !strings.Contains(out, "Go") {
		t.Errorf("출력에 Go 언어가 없습니다:\n%s", out)
	}
	if !strings.Contains(out, ".golangci.yml") {
		t.Errorf("출력에 lint 설정 파일명이 없습니다:\n%s", out)
	}
}

func TestRunAnalyze_NotARepo(t *testing.T) {
	chdirTemp(t)
	if err := runAnalyze(); err == nil {
		t.Error("git 저장소가 아니면 에러가 나야 합니다")
	}
}

// ---- getTrackedFiles ----------------------------------------------------------

func TestGetTrackedFiles(t *testing.T) {
	newTestGitRepo(t)
	files, err := getTrackedFiles()
	if err != nil {
		t.Fatalf("getTrackedFiles: %v", err)
	}
	// newTestGitRepo 가 README.md 를 커밋함
	found := false
	for _, f := range files {
		if f == "README.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("README.md 가 추적 목록에 없습니다: %v", files)
	}
}

// ---- 언어 필터 ------------------------------------------------------------------

func TestFilterProgrammingLangs(t *testing.T) {
	langs := []langInfo{
		{Name: "Go", Count: 3},
		{Name: "YAML", Count: 2},
		{Name: "JSON", Count: 1},
		{Name: "Python", Count: 1},
	}
	got := filterProgrammingLangs(langs)
	if len(got) != 2 {
		t.Fatalf("프로그래밍 언어 2개를 기대했지만 %d개: %v", len(got), got)
	}
	for _, l := range got {
		if l.Name == "YAML" || l.Name == "JSON" {
			t.Errorf("데이터 형식 %s 이 프로그래밍 언어에 포함되었습니다", l.Name)
		}
	}
}

func TestFilterDataLangs(t *testing.T) {
	langs := []langInfo{
		{Name: "Go", Count: 3},
		{Name: "YAML", Count: 2},
		{Name: "Markdown", Count: 1},
	}
	got := filterDataLangs(langs)
	if len(got) != 1 || got[0].Name != "YAML" {
		t.Errorf("YAML 만 데이터 형식이어야 합니다: %v", got)
	}
}

// ---- checkLintConfig ------------------------------------------------------------

func TestCheckLintConfig(t *testing.T) {
	dir := chdirTemp(t)

	// 설정 파일이 없으면 미발견
	if found, _ := checkLintConfig("Go"); found {
		t.Error("설정 파일이 없는데 발견으로 보고되었습니다")
	}

	// .golangci.yml 생성 후 발견
	writeTestFile(t, filepath.Join(dir, ".golangci.yml"), "version: \"2\"\n")
	found, name := checkLintConfig("Go")
	if !found || name != ".golangci.yml" {
		t.Errorf("checkLintConfig(Go) = (%v, %q), want (true, .golangci.yml)", found, name)
	}

	// 알 수 없는 언어는 항상 미발견
	if found, _ := checkLintConfig("COBOL"); found {
		t.Error("알 수 없는 언어는 미발견이어야 합니다")
	}
}

// ---- checkAndReport -------------------------------------------------------------

func TestCheckAndReport(t *testing.T) {
	dir := chdirTemp(t)
	writeTestFile(t, filepath.Join(dir, ".gitignore"), "dist/\n")

	out := captureStdout(t, func() {
		checkAndReport(".gitignore")    // 존재하는 파일
		checkAndReport(".editorconfig") // 존재하지 않는 파일
	})
	if !strings.Contains(out, ".gitignore") {
		t.Errorf("존재하는 파일 보고가 없습니다:\n%s", out)
	}
	if !strings.Contains(out, ".editorconfig") {
		t.Errorf("미존재 파일 보고가 없습니다:\n%s", out)
	}
}
