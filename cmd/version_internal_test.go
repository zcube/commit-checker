package cmd

import (
	"strings"
	"testing"

	"github.com/zcube/commit-checker/internal/logger"
)

func TestVersionCmd_PrintsVersion(t *testing.T) {
	out := captureStdout(t, func() { versionCmd.Run(versionCmd, nil) })
	if !strings.Contains(out, "commit-checker") {
		t.Errorf("버전 출력에 commit-checker 가 없습니다:\n%s", out)
	}
}

// PersistentPreRunE 가 전역 플래그를 로거에 반영하는지 확인.
func TestRootPersistentPreRunE_Quiet(t *testing.T) {
	origQuiet := globalQuiet
	t.Cleanup(func() {
		globalQuiet = origQuiet
		logger.SetQuiet(origQuiet)
	})

	globalQuiet = true
	if err := rootCmd.PersistentPreRunE(rootCmd, nil); err != nil {
		t.Fatalf("PersistentPreRunE: %v", err)
	}
}
