// allowed_words.go: allowed_words_file/allowed_words_url 에서 허용 단어 목록을 읽어 병합하는 로직.
package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// resolveAllowedWords 는 allowed_words_file, allowed_words_url 에서 단어를 읽어
// allowed_words 목록에 병합합니다.
func resolveAllowedWords(cfg *Config) error {
	if cfg.CommentLanguage.AllowedWordsFile != "" {
		filePath := cfg.CommentLanguage.AllowedWordsFile
		// ~ 경로 확장
		if strings.HasPrefix(filePath, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				filePath = filepath.Join(home, filePath[2:])
				cfg.CommentLanguage.AllowedWordsFile = filePath
			}
		}
		words, err := loadWordsFromFile(filePath)
		if err != nil {
			return fmt.Errorf("allowed_words_file 읽기 실패: %w", err)
		}
		cfg.CommentLanguage.AllowedWords = append(cfg.CommentLanguage.AllowedWords, words...)
	}
	if cfg.CommentLanguage.AllowedWordsURL != "" {
		cache := &cfg.CommentLanguage.AllowedWordsCache
		if words, ok := loadCachedWords(cache, cfg.CommentLanguage.AllowedWordsURL); ok {
			cfg.CommentLanguage.AllowedWords = append(cfg.CommentLanguage.AllowedWords, words...)
		} else {
			words, body, err := loadWordsFromURLWithBody(cfg.CommentLanguage.AllowedWordsURL)
			if err != nil {
				return fmt.Errorf("allowed_words_url 가져오기 실패: %w", err)
			}
			cfg.CommentLanguage.AllowedWords = append(cfg.CommentLanguage.AllowedWords, words...)
			saveCachedWords(cache, cfg.CommentLanguage.AllowedWordsURL, body)
		}
	}
	return nil
}

// parseWordLines 는 텍스트를 줄 단위로 분리하여 단어 목록을 반환합니다.
// '#' 로 시작하는 줄과 빈 줄은 무시합니다.
func parseWordLines(text string) []string {
	var words []string
	for _, line := range strings.Split(text, "\n") {
		word := strings.TrimSpace(line)
		if word == "" || strings.HasPrefix(word, "#") {
			continue
		}
		words = append(words, word)
	}
	return words
}

func loadWordsFromFile(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return parseWordLines(string(data)), nil
}

// maxAllowedWordsSize: URL에서 가져오는 허용 단어 파일의 최대 크기 (10MB).
const maxAllowedWordsSize = 10 * 1024 * 1024

func loadWordsFromURLWithBody(rawURL string) ([]string, []byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(rawURL) //nolint:gosec
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, maxAllowedWordsSize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, nil, err
	}
	if int64(len(body)) > maxAllowedWordsSize {
		return nil, nil, fmt.Errorf("allowed_words_url response exceeds 10MB limit")
	}
	return parseWordLines(string(body)), body, nil
}
