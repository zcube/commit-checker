package config

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AllowedWordsCacheConfig: allowed_words_url 캐싱 설정.
type AllowedWordsCacheConfig struct {
	// Enabled: 캐싱 활성화 여부 (기본값: false).
	Enabled *bool `yaml:"enabled"`

	// TTL: 캐시 유효 기간 (기본값: "24h").
	// time.ParseDuration 형식: "1h", "30m", "24h" 등.
	TTL string `yaml:"ttl"`

	// Dir: 캐시 디렉터리 경로 (기본값: ~/.cache/commit-checker).
	Dir string `yaml:"dir"`
}

// IsEnabled: 캐싱 활성화 여부 반환 (기본값: false).
func (c *AllowedWordsCacheConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// GetTTL: 캐시 유효 기간 반환 (기본값: 24h).
func (c *AllowedWordsCacheConfig) GetTTL() time.Duration {
	if c.TTL == "" {
		return 24 * time.Hour
	}
	d, err := time.ParseDuration(c.TTL)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

// GetDir: 캐시 디렉터리 경로 반환.
func (c *AllowedWordsCacheConfig) GetDir() string {
	if c.Dir != "" {
		return c.Dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "commit-checker-cache")
	}
	return filepath.Join(home, ".cache", "commit-checker")
}

// cacheKey: URL에서 캐시 파일명 생성.
func cacheKey(rawURL string) string {
	h := sha256.Sum256([]byte(rawURL))
	return fmt.Sprintf("words_%x.txt", h[:8])
}

// loadCachedWords: 캐시된 허용 단어를 로드.
// 캐시가 없거나 만료되면 nil, false 반환.
func loadCachedWords(cache *AllowedWordsCacheConfig, rawURL string) ([]string, bool) {
	if !cache.IsEnabled() {
		return nil, false
	}
	path := filepath.Join(cache.GetDir(), cacheKey(rawURL))
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if time.Since(info.ModTime()) > cache.GetTTL() {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return parseWordLines(string(data)), true
}

// saveCachedWords: 허용 단어를 캐시 파일에 저장.
func saveCachedWords(cache *AllowedWordsCacheConfig, rawURL string, body []byte) {
	if !cache.IsEnabled() {
		return
	}
	dir := cache.GetDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	path := filepath.Join(dir, cacheKey(rawURL))
	_ = os.WriteFile(path, body, 0644)
}
