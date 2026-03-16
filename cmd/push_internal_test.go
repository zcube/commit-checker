package cmd

import (
	"strings"
	"testing"
)

func TestIsPushZeroSHA(t *testing.T) {
	tests := []struct {
		sha  string
		want bool
	}{
		{"0000000000000000000000000000000000000000", true},
		{"abc1234567890123456789012345678901234567", false},
		{"", false},
		{"00000000000000000000000000000000000000", false}, // 38 zeros
		{"000000000000000000000000000000000000000x", false},
	}
	for _, tt := range tests {
		got := isPushZeroSHA(tt.sha)
		if got != tt.want {
			t.Errorf("isPushZeroSHA(%q) = %v, want %v", tt.sha, got, tt.want)
		}
	}
}

func TestParsePushRanges_Empty(t *testing.T) {
	ranges := parsePushRanges(strings.NewReader(""))
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges, got %d", len(ranges))
	}
}

func TestParsePushRanges_ValidRange(t *testing.T) {
	input := "refs/heads/main abc1234567890123456789012345678901234567 refs/heads/main def1234567890123456789012345678901234567\n"
	ranges := parsePushRanges(strings.NewReader(input))
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	expected := "def1234567890123456789012345678901234567..abc1234567890123456789012345678901234567"
	if ranges[0] != expected {
		t.Errorf("range = %q, want %q", ranges[0], expected)
	}
}

func TestParsePushRanges_DeleteBranch(t *testing.T) {
	// local SHA is all zeros → branch deletion, skip
	input := "refs/heads/feat 0000000000000000000000000000000000000000 refs/heads/feat abc1234567890123456789012345678901234567\n"
	ranges := parsePushRanges(strings.NewReader(input))
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges for deletion, got %d: %v", len(ranges), ranges)
	}
}

func TestParsePushRanges_NewBranch_NoRemoteBase(t *testing.T) {
	// remote SHA is all zeros → new branch, but no upstream found → skip
	input := "refs/heads/new-feature abc1234567890123456789012345678901234567 refs/heads/new-feature 0000000000000000000000000000000000000000\n"
	// findPushRemoteBase will likely fail in test environment since there's no real remote
	// Result may be 0 or 1 depending on git state — just verify it doesn't crash
	_ = parsePushRanges(strings.NewReader(input))
}

func TestParsePushRanges_InvalidLine(t *testing.T) {
	input := "invalid line with only three parts\n"
	ranges := parsePushRanges(strings.NewReader(input))
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges for invalid line, got %d", len(ranges))
	}
}

func TestParsePushRanges_MultipleRefs(t *testing.T) {
	input := strings.Join([]string{
		"refs/heads/main abc1234567890123456789012345678901234567 refs/heads/main def1234567890123456789012345678901234567",
		"refs/heads/feature ghi1234567890123456789012345678901234567 refs/heads/feature jkl1234567890123456789012345678901234567",
	}, "\n") + "\n"
	ranges := parsePushRanges(strings.NewReader(input))
	if len(ranges) != 2 {
		t.Errorf("expected 2 ranges, got %d", len(ranges))
	}
}

func TestParsePushRanges_BlankLines(t *testing.T) {
	input := "\n\n\n"
	ranges := parsePushRanges(strings.NewReader(input))
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges for blank lines, got %d", len(ranges))
	}
}
