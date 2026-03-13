package config

import (
	"strings"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)
	warns := Validate(cfg, "test.yml")
	if len(warns) != 0 {
		t.Errorf("expected no warnings, got: %v", warns)
	}
}

func TestValidate_InvalidRequiredLanguage(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)
	cfg.CommentLanguage.RequiredLanguage = "spanish"
	warns := Validate(cfg, "test.yml")
	if len(warns) == 0 {
		t.Error("expected warning for invalid required_language")
	}
	if !strings.Contains(warns[0], "spanish") {
		t.Errorf("warning should mention 'spanish', got: %s", warns[0])
	}
}

func TestValidate_InvalidGlobPattern(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)
	cfg.BinaryFile.IgnoreFiles = []string{"[invalid-glob"}
	warns := Validate(cfg, "test.yml")
	if len(warns) == 0 {
		t.Error("expected warning for invalid glob pattern")
	}
}

func TestValidate_AllowedWordsFileMissing(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)
	cfg.CommentLanguage.AllowedWordsFile = "/nonexistent/path/words.txt"
	warns := Validate(cfg, "test.yml")
	if len(warns) == 0 {
		t.Error("expected warning for missing allowed_words_file")
	}
}

func TestValidate_InvalidFileLanguage(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)
	cfg.CommentLanguage.FileLanguages = []FileLanguageRule{
		{Pattern: "**/*.go", Language: "klingon"},
	}
	warns := Validate(cfg, "test.yml")
	if len(warns) == 0 {
		t.Error("expected warning for invalid file_languages language")
	}
}
