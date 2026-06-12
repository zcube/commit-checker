package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	charmlog "github.com/charmbracelet/log"
)

// setBufferOutput 은 로거 출력을 버퍼로 교체하고 테스트 종료 시 원복합니다.
// 패키지 전역 상태(output, instance)를 건드리므로 병렬 실행하지 않습니다.
func setBufferOutput(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}

	mu.Lock()
	prevOutput := output
	output = buf
	mu.Unlock()

	// 인스턴스가 아직 없으면 여기서 buf 를 대상으로 생성되고,
	// 이미 있으면 SetOutput 으로 교체합니다.
	ensure().SetOutput(buf)

	t.Cleanup(func() {
		mu.Lock()
		output = prevOutput
		mu.Unlock()
		ensure().SetOutput(prevOutput)
	})
	return buf
}

// setLevelWithCleanup 은 로그 레벨을 설정하고 테스트 종료 시 원복합니다.
func setLevelWithCleanup(t *testing.T, l charmlog.Level) {
	t.Helper()

	mu.RLock()
	prevLevel := level
	mu.RUnlock()

	SetLevel(l)
	t.Cleanup(func() { SetLevel(prevLevel) })
}

// ensure 는 항상 동일한 인스턴스를 반환해야 합니다 (lazy init + 재사용).
func TestEnsure_ReturnsSameInstance(t *testing.T) {
	first := ensure()
	if first == nil {
		t.Fatal("ensure() 가 nil 을 반환했습니다")
	}
	if second := ensure(); first != second {
		t.Error("ensure() 가 호출마다 다른 인스턴스를 반환합니다")
	}
}

func TestSetQuiet(t *testing.T) {
	buf := setBufferOutput(t)
	setLevelWithCleanup(t, charmlog.WarnLevel)

	// quiet=true: 출력 대상이 io.Discard 로 바뀌고 로그가 억제되어야 합니다.
	SetQuiet(true)
	mu.RLock()
	got := output
	mu.RUnlock()
	if got != io.Discard {
		t.Error("SetQuiet(true) 후 output 이 io.Discard 가 아닙니다")
	}

	Warn("quiet 상태 메시지")
	if buf.Len() != 0 {
		t.Errorf("quiet 상태에서 버퍼에 출력이 발생했습니다:\n%s", buf.String())
	}

	// quiet=false: 출력 대상이 stderr 로 복귀해야 합니다.
	SetQuiet(false)
	mu.RLock()
	got = output
	mu.RUnlock()
	if got != os.Stderr {
		t.Error("SetQuiet(false) 후 output 이 os.Stderr 가 아닙니다")
	}

	// 이후 테스트가 stderr 로 출력하지 않도록 버퍼로 되돌립니다.
	ensure().SetOutput(buf)
	mu.Lock()
	output = buf
	mu.Unlock()
}

func TestSetLevel_FiltersBelowLevel(t *testing.T) {
	buf := setBufferOutput(t)
	setLevelWithCleanup(t, charmlog.ErrorLevel)

	Warn("억제되어야 하는 경고")
	if buf.Len() != 0 {
		t.Errorf("ErrorLevel 에서 Warn 이 출력되었습니다:\n%s", buf.String())
	}

	Error("출력되어야 하는 오류")
	if !strings.Contains(buf.String(), "출력되어야 하는 오류") {
		t.Errorf("ErrorLevel 에서 Error 가 출력되지 않았습니다:\n%s", buf.String())
	}
}

// 각 레벨 함수가 메시지와 key-value 쌍을 출력하는지 확인합니다.
func TestLogFunctions_WriteMessageAndKeyvals(t *testing.T) {
	cases := []struct {
		name  string
		logFn func(msg string, keyvals ...any)
		msg   string
	}{
		{"Debug", Debug, "디버그 메시지"},
		{"Info", Info, "정보 메시지"},
		{"Warn", Warn, "경고 메시지"},
		{"Error", Error, "오류 메시지"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := setBufferOutput(t)
			setLevelWithCleanup(t, charmlog.DebugLevel)

			tc.logFn(tc.msg, "key", "value")

			out := buf.String()
			if !strings.Contains(out, tc.msg) {
				t.Errorf("출력에 메시지가 없습니다 (%q):\n%s", tc.msg, out)
			}
			if !strings.Contains(out, "key") || !strings.Contains(out, "value") {
				t.Errorf("출력에 key-value 쌍이 없습니다:\n%s", out)
			}
		})
	}
}

func TestSetNoColor_False_KeepsInstance(t *testing.T) {
	setBufferOutput(t)

	mu.RLock()
	before := instance
	mu.RUnlock()

	// noColor=false 는 아무것도 하지 않아야 합니다.
	SetNoColor(false)

	mu.RLock()
	after := instance
	mu.RUnlock()
	if before != after {
		t.Error("SetNoColor(false) 가 인스턴스를 교체했습니다")
	}
}

func TestSetNoColor_True_RecreatesInstance(t *testing.T) {
	buf := setBufferOutput(t)
	setLevelWithCleanup(t, charmlog.WarnLevel)

	mu.RLock()
	before := instance
	mu.RUnlock()

	SetNoColor(true)

	mu.RLock()
	after := instance
	mu.RUnlock()
	if before == after {
		t.Error("SetNoColor(true) 가 인스턴스를 재생성하지 않았습니다")
	}

	// 재생성된 인스턴스가 현재 output(버퍼)으로, ANSI 색상 없이 출력해야 합니다.
	Warn("색상 없는 경고")
	out := buf.String()
	if !strings.Contains(out, "색상 없는 경고") {
		t.Errorf("재생성된 인스턴스가 버퍼로 출력하지 않았습니다:\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("noColor 설정 후에도 ANSI 이스케이프가 출력되었습니다:\n%q", out)
	}
}

// instance 가 아직 생성되지 않은 상태에서는 SetNoColor 가 기록 없이 반환해야 합니다.
func TestSetNoColor_NilInstance_NoOp(t *testing.T) {
	setBufferOutput(t) // instance 생성 보장 및 출력 원복 예약

	mu.Lock()
	prev := instance
	instance = nil
	mu.Unlock()

	// 다른 테스트가 nil instance 상태를 보지 않도록 반드시 원복합니다.
	t.Cleanup(func() {
		mu.Lock()
		instance = prev
		mu.Unlock()
	})

	SetNoColor(true)

	mu.RLock()
	got := instance
	mu.RUnlock()
	if got != nil {
		t.Error("instance 가 nil 일 때 SetNoColor 가 인스턴스를 생성했습니다")
	}
}
