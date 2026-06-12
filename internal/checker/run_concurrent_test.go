package checker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

// TestForEachFileConcurrent_CtxCanceled: ctx가 이미 취소된 경우 파일을 처리하지 않고
// ctx.Err()를 반환하며 조기 중단하는지 검증.
func TestForEachFileConcurrent_CtxCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // 호출 전에 미리 취소

	var calls atomic.Int64
	files := []string{"a.txt", "b.txt", "c.txt"}

	errs, err := forEachFileConcurrent(ctx, files, func(string) ([]string, error) {
		calls.Add(1)
		return []string{"violation"}, nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if errs != nil {
		t.Fatalf("expected nil errs on cancel, got %v", errs)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("expected no fn calls after cancel, got %d", got)
	}
}

// TestForEachFileConcurrent_Normal: 정상 ctx에서는 모든 파일을 처리하고 메시지를 수집.
func TestForEachFileConcurrent_Normal(t *testing.T) {
	var calls atomic.Int64
	files := []string{"a.txt", "b.txt", "c.txt"}

	errs, err := forEachFileConcurrent(t.Context(), files, func(string) ([]string, error) {
		calls.Add(1)
		return []string{"msg"}, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != len(files) {
		t.Fatalf("expected %d messages, got %d", len(files), len(errs))
	}
	if got := calls.Load(); got != int64(len(files)) {
		t.Fatalf("expected %d fn calls, got %d", len(files), got)
	}
}
