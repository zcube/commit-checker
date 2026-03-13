package logger

import (
	"io"
	"os"
	"sync"

	charmlog "github.com/charmbracelet/log"
)

var (
	mu       sync.RWMutex
	instance *charmlog.Logger
	output   io.Writer = os.Stderr // 현재 출력 대상 추적
	level    charmlog.Level = charmlog.WarnLevel
)

func init() {
	instance = charmlog.NewWithOptions(os.Stderr, charmlog.Options{
		Level:           charmlog.WarnLevel,
		ReportTimestamp: false,
		ReportCaller:    false,
	})
}

// SetQuiet: quiet=true이면 모든 로그 출력을 억제합니다.
func SetQuiet(quiet bool) {
	mu.Lock()
	defer mu.Unlock()
	if quiet {
		output = io.Discard
		instance.SetOutput(io.Discard)
	} else {
		output = os.Stderr
		instance.SetOutput(os.Stderr)
	}
}

// SetNoColor: noColor=true이면 ANSI 색상 출력을 비활성화합니다.
func SetNoColor(noColor bool) {
	mu.Lock()
	defer mu.Unlock()
	if noColor {
		noColorInstance := charmlog.NewWithOptions(output, charmlog.Options{
			Level:           level,
			ReportTimestamp: false,
			Formatter:       charmlog.TextFormatter,
		})
		instance = noColorInstance
	}
}

// SetLevel: 최소 로그 레벨을 설정합니다.
func SetLevel(l charmlog.Level) {
	mu.Lock()
	defer mu.Unlock()
	level = l
	instance.SetLevel(l)
}

// Warn: 경고 메시지를 로깅합니다.
func Warn(msg string, keyvals ...any) {
	mu.RLock()
	defer mu.RUnlock()
	instance.Warn(msg, keyvals...)
}

// Debug: 디버그 메시지를 로깅합니다.
func Debug(msg string, keyvals ...any) {
	mu.RLock()
	defer mu.RUnlock()
	instance.Debug(msg, keyvals...)
}

// Info: 정보 메시지를 로깅합니다.
func Info(msg string, keyvals ...any) {
	mu.RLock()
	defer mu.RUnlock()
	instance.Info(msg, keyvals...)
}

// Error: 오류 메시지를 로깅합니다.
func Error(msg string, keyvals ...any) {
	mu.RLock()
	defer mu.RUnlock()
	instance.Error(msg, keyvals...)
}
