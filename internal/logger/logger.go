package logger

import (
	"io"
	"os"
	"sync"

	charmlog "github.com/charmbracelet/log"
)

// charmlog.NewWithOptions 가 stderr 가 TTY 일 때 색 감지용 OSC 쿼리를 전송하며,
// PTY 환경(예: lefthook, CI)에서 응답이 없으면 ~15초 블록합니다.
// 따라서 패키지 init 에서 즉시 생성하지 않고, 실제 로깅이 필요할 때 lazy 로 초기화합니다.

var (
	mu       sync.RWMutex
	instance *charmlog.Logger
	output   io.Writer      = os.Stderr // 현재 출력 대상 추적
	level    charmlog.Level = charmlog.WarnLevel
	initOnce sync.Once
)

// ensure 는 charmlog 인스턴스를 한 번만 생성합니다 (lazy init).
func ensure() *charmlog.Logger {
	initOnce.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		instance = charmlog.NewWithOptions(output, charmlog.Options{
			Level:           level,
			ReportTimestamp: false,
			ReportCaller:    false,
		})
	})
	mu.RLock()
	defer mu.RUnlock()
	return instance
}

// SetQuiet: quiet=true이면 모든 로그 출력을 억제합니다.
// charmlog 인스턴스가 아직 생성되지 않았다면 출력 변수만 업데이트하고
// 이후 ensure() 가 적절히 처리합니다.
func SetQuiet(quiet bool) {
	mu.Lock()
	if quiet {
		output = io.Discard
	} else {
		output = os.Stderr
	}
	inst := instance
	mu.Unlock()
	if inst != nil {
		inst.SetOutput(output)
	}
}

// SetNoColor: noColor=true이면 ANSI 색상 출력을 비활성화합니다.
// charmlog 인스턴스가 아직 없으면 기록만 해두고 ensure() 가 처리합니다.
func SetNoColor(noColor bool) {
	mu.Lock()
	defer mu.Unlock()
	if !noColor || instance == nil {
		return
	}
	instance = charmlog.NewWithOptions(output, charmlog.Options{
		Level:           level,
		ReportTimestamp: false,
		Formatter:       charmlog.TextFormatter,
	})
}

// SetLevel: 최소 로그 레벨을 설정합니다.
func SetLevel(l charmlog.Level) {
	mu.Lock()
	level = l
	inst := instance
	mu.Unlock()
	if inst != nil {
		inst.SetLevel(l)
	}
}

// Warn: 경고 메시지를 로깅합니다.
func Warn(msg string, keyvals ...any) {
	ensure().Warn(msg, keyvals...)
}

// Debug: 디버그 메시지를 로깅합니다.
func Debug(msg string, keyvals ...any) {
	ensure().Debug(msg, keyvals...)
}

// Info: 정보 메시지를 로깅합니다.
func Info(msg string, keyvals ...any) {
	ensure().Info(msg, keyvals...)
}

// Error: 오류 메시지를 로깅합니다.
func Error(msg string, keyvals ...any) {
	ensure().Error(msg, keyvals...)
}
