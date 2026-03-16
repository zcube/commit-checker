package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func boolPtr(b bool) *bool { return &b }

func TestAllowedWordsCacheConfig_IsEnabled(t *testing.T) {
	var c AllowedWordsCacheConfig
	if c.IsEnabled() {
		t.Error("nil Enabled should return false")
	}
	c.Enabled = boolPtr(true)
	if !c.IsEnabled() {
		t.Error("true should return true")
	}
	c.Enabled = boolPtr(false)
	if c.IsEnabled() {
		t.Error("false should return false")
	}
}

func TestAllowedWordsCacheConfig_GetTTL(t *testing.T) {
	var c AllowedWordsCacheConfig
	if c.GetTTL() != 24*time.Hour {
		t.Errorf("empty TTL should default to 24h, got %v", c.GetTTL())
	}
	c.TTL = "1h"
	if c.GetTTL() != time.Hour {
		t.Errorf("expected 1h, got %v", c.GetTTL())
	}
	c.TTL = "invalid"
	if c.GetTTL() != 24*time.Hour {
		t.Errorf("invalid TTL should default to 24h, got %v", c.GetTTL())
	}
}

func TestAllowedWordsCacheConfig_GetDir(t *testing.T) {
	var c AllowedWordsCacheConfig
	dir := c.GetDir()
	if dir == "" {
		t.Error("default dir should not be empty")
	}
	c.Dir = "/custom/cache"
	if c.GetDir() != "/custom/cache" {
		t.Errorf("expected /custom/cache, got %q", c.GetDir())
	}
}

func TestSaveCachedWords_And_LoadCachedWords(t *testing.T) {
	tmpDir := t.TempDir()
	enabled := true
	c := &AllowedWordsCacheConfig{
		Enabled: &enabled,
		Dir:     tmpDir,
	}

	rawURL := "https://example.com/words.txt"
	saveCachedWords(c, rawURL, []byte("word1\nword2\n"))

	words, ok := loadCachedWords(c, rawURL)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(words) != 2 {
		t.Errorf("expected 2 words, got %d: %v", len(words), words)
	}
}

func TestLoadCachedWords_Miss_WhenDisabled(t *testing.T) {
	var c AllowedWordsCacheConfig
	words, ok := loadCachedWords(&c, "https://example.com/words.txt")
	if ok || words != nil {
		t.Error("disabled cache should return nil, false")
	}
}

func TestLoadCachedWords_Miss_WhenExpired(t *testing.T) {
	tmpDir := t.TempDir()
	enabled := true
	c := &AllowedWordsCacheConfig{
		Enabled: &enabled,
		Dir:     tmpDir,
		TTL:     "1ms",
	}

	rawURL := "https://example.com/words.txt"
	saveCachedWords(c, rawURL, []byte("word1\n"))

	path := filepath.Join(tmpDir, cacheKey(rawURL))
	past := time.Now().Add(-time.Second)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	_, ok := loadCachedWords(c, rawURL)
	if ok {
		t.Error("expired cache should return false")
	}
}

func TestLoadCachedWords_Miss_WhenNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	enabled := true
	c := &AllowedWordsCacheConfig{
		Enabled: &enabled,
		Dir:     tmpDir,
	}
	_, ok := loadCachedWords(c, "https://example.com/missing.txt")
	if ok {
		t.Error("missing file should return false")
	}
}
