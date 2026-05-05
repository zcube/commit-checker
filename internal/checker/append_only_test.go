package checker_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func appendOnlyConfig(paths ...string) *config.Config {
	cfg := &config.Config{}
	cfg.AppendOnly.Enabled = true
	cfg.AppendOnly.Paths = paths
	return cfg
}

// TestCheckAppendOnly_Disabled: enabled=false 이면 검사 건너뜀.
func TestCheckAppendOnly_Disabled(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/001.sql", "CREATE TABLE a (id INT);\n")
	stageFile(t, dir, "migrations/001.sql", "DROP TABLE a;\n")

	cfg := &config.Config{}
	cfg.AppendOnly.Enabled = false
	cfg.AppendOnly.Paths = []string{"migrations/**"}

	errs, err := checker.CheckAppendOnly(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("disabled check should return no errors, got: %v", errs)
	}
}

// TestCheckAppendOnly_NewFile: append-only 경로에 새 파일 추가는 허용.
func TestCheckAppendOnly_NewFile(t *testing.T) {
	dir := newGitRepo(t)
	// 빈 초기 커밋
	seedCommit(t, dir, "README.md", "init\n")
	// 새 마이그레이션 파일 추가
	stageFile(t, dir, "migrations/001.sql", "CREATE TABLE users (id SERIAL PRIMARY KEY);\n")

	errs, err := checker.CheckAppendOnly(appendOnlyConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("new file in append-only path should be allowed, got: %v", errs)
	}
}

// TestCheckAppendOnly_AppendAtEnd: 파일 끝에 내용 추가는 허용.
func TestCheckAppendOnly_AppendAtEnd(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/001.sql", "CREATE TABLE users (id SERIAL PRIMARY KEY);\n")
	stageFile(t, dir, "migrations/001.sql",
		"CREATE TABLE users (id SERIAL PRIMARY KEY);\n"+
			"ALTER TABLE users ADD COLUMN email TEXT;\n")

	errs, err := checker.CheckAppendOnly(appendOnlyConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("append at end should be allowed, got: %v", errs)
	}
}

// TestCheckAppendOnly_DeleteFile: append-only 경로에서 파일 삭제는 차단.
func TestCheckAppendOnly_DeleteFile(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/001.sql", "CREATE TABLE users (id SERIAL PRIMARY KEY);\n")
	gitMust(t, dir, "git", "rm", "migrations/001.sql")

	errs, err := checker.CheckAppendOnly(appendOnlyConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("file deletion in append-only path should be blocked")
	}
}

// TestCheckAppendOnly_ModifyExisting: 기존 줄 수정은 차단.
func TestCheckAppendOnly_ModifyExisting(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/001.sql", "CREATE TABLE users (id INT);\n")
	stageFile(t, dir, "migrations/001.sql", "CREATE TABLE users (id SERIAL);\n")

	errs, err := checker.CheckAppendOnly(appendOnlyConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("modification in append-only path should be blocked")
	}
}

// TestCheckAppendOnly_DeleteLines: 기존 줄 삭제는 차단.
func TestCheckAppendOnly_DeleteLines(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/001.sql",
		"CREATE TABLE users (id SERIAL PRIMARY KEY);\n"+
			"CREATE INDEX idx_users ON users(id);\n")
	// 두 번째 줄 제거
	stageFile(t, dir, "migrations/001.sql", "CREATE TABLE users (id SERIAL PRIMARY KEY);\n")

	errs, err := checker.CheckAppendOnly(appendOnlyConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("line deletion in append-only path should be blocked")
	}
}

// TestCheckAppendOnly_InsertInMiddle: 파일 중간에 줄 삽입은 차단.
func TestCheckAppendOnly_InsertInMiddle(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/001.sql",
		"CREATE TABLE users (id SERIAL PRIMARY KEY);\n"+
			"CREATE TABLE posts (id SERIAL PRIMARY KEY);\n")
	// 두 줄 사이에 삽입
	stageFile(t, dir, "migrations/001.sql",
		"CREATE TABLE users (id SERIAL PRIMARY KEY);\n"+
			"CREATE TABLE comments (id SERIAL PRIMARY KEY);\n"+
			"CREATE TABLE posts (id SERIAL PRIMARY KEY);\n")

	errs, err := checker.CheckAppendOnly(appendOnlyConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("middle insertion in append-only path should be blocked")
	}
}

// TestCheckAppendOnly_PathNotMatched: 패턴 외 경로는 검사 제외.
func TestCheckAppendOnly_PathNotMatched(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "src/main.go", "package main\n")
	stageFile(t, dir, "src/main.go", "package main\n\nfunc main() {}\n")

	errs, err := checker.CheckAppendOnly(appendOnlyConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("file outside append-only path should not be checked, got: %v", errs)
	}
}

// TestCheckAppendOnly_GlobalIgnore: global_ignore 패턴은 검사 제외.
func TestCheckAppendOnly_GlobalIgnore(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/001.sql", "CREATE TABLE users (id INT);\n")
	stageFile(t, dir, "migrations/001.sql", "MODIFIED CONTENT;\n")

	cfg := appendOnlyConfig("migrations/**")
	cfg.Exceptions.GlobalIgnore = []string{"migrations/001.sql"}

	errs, err := checker.CheckAppendOnly(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("globally ignored file should be skipped, got: %v", errs)
	}
}

// TestCheckAppendOnly_MultipleViolations: 여러 파일 위반 시 모두 보고.
func TestCheckAppendOnly_MultipleViolations(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/001.sql", "CREATE TABLE a (id INT);\n")
	seedCommit(t, dir, "migrations/002.sql", "CREATE TABLE b (id INT);\n")
	stageFile(t, dir, "migrations/001.sql", "MODIFIED A;\n")
	stageFile(t, dir, "migrations/002.sql", "MODIFIED B;\n")

	errs, err := checker.CheckAppendOnly(appendOnlyConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) < 2 {
		t.Errorf("expected at least 2 errors for 2 violations, got %d: %v", len(errs), errs)
	}
}

// ---- FilenameOrder=numeric 테스트 ------------------------------------------

func appendOnlyNumericConfig(paths ...string) *config.Config {
	cfg := appendOnlyConfig(paths...)
	cfg.AppendOnly.FilenameOrder = "numeric"
	return cfg
}

// TestCheckAppendOnly_FilenameOrder_NewFileAfter: 기존 파일보다 뒤에 오는 이름은 허용.
func TestCheckAppendOnly_FilenameOrder_NewFileAfter(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/001.sql", "CREATE TABLE a (id INT);\n")
	seedCommit(t, dir, "migrations/002.sql", "CREATE TABLE b (id INT);\n")
	stageFile(t, dir, "migrations/003.sql", "CREATE TABLE c (id INT);\n")

	errs, err := checker.CheckAppendOnly(appendOnlyNumericConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("file with name after existing files should be allowed, got: %v", errs)
	}
}

// TestCheckAppendOnly_FilenameOrder_NewFileBefore: 기존 파일보다 앞에 오는 이름은 차단.
func TestCheckAppendOnly_FilenameOrder_NewFileBefore(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/002.sql", "CREATE TABLE b (id INT);\n")
	stageFile(t, dir, "migrations/001.sql", "CREATE TABLE a (id INT);\n")

	errs, err := checker.CheckAppendOnly(appendOnlyNumericConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("file with name before existing max should be blocked")
	}
}

// TestCheckAppendOnly_FilenameOrder_SameName: 기존 파일과 같은 이름은 차단 (IsNew=false라 내용 검사로 처리).
func TestCheckAppendOnly_FilenameOrder_SameName_Modify(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/002.sql", "CREATE TABLE b (id INT);\n")
	// 같은 이름 파일 수정 시도 → 내용 검사에서 차단
	stageFile(t, dir, "migrations/002.sql", "DROP TABLE b;\n")

	errs, err := checker.CheckAppendOnly(appendOnlyNumericConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("modification of existing file should be blocked")
	}
}

// TestCheckAppendOnly_FilenameOrder_NumericSort: 9 < 10 (lexicographic 이 아닌 numeric 정렬).
func TestCheckAppendOnly_FilenameOrder_NumericSort(t *testing.T) {
	dir := newGitRepo(t)
	// 9번 파일이 최대. lexicographic 으로는 "9" > "10" 이지만 numeric 으로는 9 < 10
	for _, f := range []string{"001", "002", "003", "004", "005", "006", "007", "008", "009"} {
		seedCommit(t, dir, "migrations/"+f+".sql", "-- "+f+"\n")
	}
	stageFile(t, dir, "migrations/010.sql", "-- 010\n")

	errs, err := checker.CheckAppendOnly(appendOnlyNumericConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("010 should come after 009 in numeric sort, got errors: %v", errs)
	}
}

// TestCheckAppendOnly_FilenameOrder_FirstFile: 기존 파일이 없으면 어떤 이름이든 허용.
func TestCheckAppendOnly_FilenameOrder_FirstFile(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "README.md", "init\n")
	stageFile(t, dir, "migrations/001.sql", "CREATE TABLE a (id INT);\n")

	errs, err := checker.CheckAppendOnly(appendOnlyNumericConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("first file in directory should always be allowed, got: %v", errs)
	}
}

// TestCheckAppendOnly_FilenameOrder_Disabled: filename_order 없으면 순서 검사 안 함.
func TestCheckAppendOnly_FilenameOrder_Disabled(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "migrations/002.sql", "CREATE TABLE b (id INT);\n")
	stageFile(t, dir, "migrations/001.sql", "CREATE TABLE a (id INT);\n")

	// filename_order 설정 없음
	errs, err := checker.CheckAppendOnly(appendOnlyConfig("migrations/**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("without filename_order option, order should not be checked, got: %v", errs)
	}
}
