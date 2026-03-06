package emoji

import "testing"

func TestIsEmoji(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'A', false},
		{'가', false},
		{'あ', false},
		{' ', false},
		{'©', false}, // common text char, not treated as emoji
		{'😀', true},
		{'🎉', true},
		{'🚀', true},
		{'❌', true},
		{'✅', true},
		{'⭐', true},
		{'🇰', true}, // regional indicator
		{'🤔', true},
		{'👍', true},
		{'🔥', true},
		{'💡', true},
	}
	for _, tt := range tests {
		got := IsEmoji(tt.r)
		if got != tt.want {
			t.Errorf("IsEmoji(%q U+%04X) = %v, want %v", string(tt.r), tt.r, got, tt.want)
		}
	}
}

func TestContainsEmoji(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"hello world", false},
		{"변수 설정", false},
		{"fix: bug 수정", false},
		{"fix: 🐛 bug fix", true},
		{"feat: ✨ new feature", true},
		{"🚀 deploy", true},
		{"no emoji here! @#$%", false},
	}
	for _, tt := range tests {
		got := ContainsEmoji(tt.text)
		if got != tt.want {
			t.Errorf("ContainsEmoji(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestFindEmojis(t *testing.T) {
	text := "line1 😀 text\nline2 🎉 more"
	emojis := FindEmojis(text)
	if len(emojis) != 2 {
		t.Fatalf("expected 2 emojis, got %d", len(emojis))
	}
	if emojis[0].Line != 1 || emojis[0].Char != "😀" {
		t.Errorf("first emoji: line=%d char=%s", emojis[0].Line, emojis[0].Char)
	}
	if emojis[1].Line != 2 || emojis[1].Char != "🎉" {
		t.Errorf("second emoji: line=%d char=%s", emojis[1].Line, emojis[1].Char)
	}
}
