package config_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/config"
)

func ptrBool(b bool) *bool { return &b }

// SubjectLimitConfig

func TestSubjectLimitConfig_IsEnabled_Nil(t *testing.T) {
	c := &config.SubjectLimitConfig{}
	if c.IsEnabled() {
		t.Error("nil Enabled should default to false")
	}
}

func TestSubjectLimitConfig_IsEnabled_True(t *testing.T) {
	c := &config.SubjectLimitConfig{Enabled: ptrBool(true)}
	if !c.IsEnabled() {
		t.Error("expected true")
	}
}

func TestSubjectLimitConfig_GetMaxLength_Default(t *testing.T) {
	c := &config.SubjectLimitConfig{}
	if c.GetMaxLength() != 72 {
		t.Errorf("expected 72, got %d", c.GetMaxLength())
	}
}

func TestSubjectLimitConfig_GetMaxLength_Custom(t *testing.T) {
	c := &config.SubjectLimitConfig{MaxLength: 50}
	if c.GetMaxLength() != 50 {
		t.Errorf("expected 50, got %d", c.GetMaxLength())
	}
}

// BodyLineLimitConfig

func TestBodyLineLimitConfig_IsEnabled_Nil(t *testing.T) {
	c := &config.BodyLineLimitConfig{}
	if c.IsEnabled() {
		t.Error("nil Enabled should default to false")
	}
}

func TestBodyLineLimitConfig_IsEnabled_True(t *testing.T) {
	c := &config.BodyLineLimitConfig{Enabled: ptrBool(true)}
	if !c.IsEnabled() {
		t.Error("expected true")
	}
}

func TestBodyLineLimitConfig_GetMaxLength_Default(t *testing.T) {
	c := &config.BodyLineLimitConfig{}
	if c.GetMaxLength() != 100 {
		t.Errorf("expected 100, got %d", c.GetMaxLength())
	}
}

func TestBodyLineLimitConfig_GetMaxLength_Custom(t *testing.T) {
	c := &config.BodyLineLimitConfig{MaxLength: 80}
	if c.GetMaxLength() != 80 {
		t.Errorf("expected 80, got %d", c.GetMaxLength())
	}
}

// LintConfig

func TestLintConfig_IsEnabled_Nil(t *testing.T) {
	c := &config.LintConfig{}
	if !c.IsEnabled() {
		t.Error("nil Enabled should default to true")
	}
}

func TestLintConfig_IsEnabled_False(t *testing.T) {
	c := &config.LintConfig{Enabled: ptrBool(false)}
	if c.IsEnabled() {
		t.Error("expected false")
	}
}

// LintRuleConfig

func TestLintRuleConfig_IsEnabled_Nil(t *testing.T) {
	c := &config.LintRuleConfig{}
	if !c.IsEnabled() {
		t.Error("nil Enabled should default to true")
	}
}

func TestLintRuleConfig_IsEnabled_False(t *testing.T) {
	c := &config.LintRuleConfig{Enabled: ptrBool(false)}
	if c.IsEnabled() {
		t.Error("expected false")
	}
}

// JSONLintConfig

func TestJSONLintConfig_IsEnabled_Nil(t *testing.T) {
	c := &config.JSONLintConfig{}
	if !c.IsEnabled() {
		t.Error("nil Enabled should default to true")
	}
}

func TestJSONLintConfig_IsEnabled_False(t *testing.T) {
	c := &config.JSONLintConfig{Enabled: ptrBool(false)}
	if c.IsEnabled() {
		t.Error("expected false")
	}
}

func TestJSONLintConfig_IsAllowJSON5_Nil(t *testing.T) {
	c := &config.JSONLintConfig{}
	if c.IsAllowJSON5() {
		t.Error("nil AllowJSON5 should default to false")
	}
}

func TestJSONLintConfig_IsAllowJSON5_True(t *testing.T) {
	c := &config.JSONLintConfig{AllowJSON5: ptrBool(true)}
	if !c.IsAllowJSON5() {
		t.Error("expected true")
	}
}

func TestJSONLintConfig_IsCommentFilter_Nil(t *testing.T) {
	c := &config.JSONLintConfig{}
	if c.IsCommentFilter() {
		t.Error("nil CommentFilter should default to false")
	}
}

func TestJSONLintConfig_IsCommentFilter_True(t *testing.T) {
	c := &config.JSONLintConfig{CommentFilter: ptrBool(true)}
	if !c.IsCommentFilter() {
		t.Error("expected true")
	}
}

// YAMLLintConfig

func TestYAMLLintConfig_IsEnabled_Nil(t *testing.T) {
	c := &config.YAMLLintConfig{}
	if !c.IsEnabled() {
		t.Error("nil Enabled should default to true")
	}
}

func TestYAMLLintConfig_IsEnabled_False(t *testing.T) {
	c := &config.YAMLLintConfig{Enabled: ptrBool(false)}
	if c.IsEnabled() {
		t.Error("expected false")
	}
}

func TestYAMLLintConfig_IsCommentFilter_Nil(t *testing.T) {
	c := &config.YAMLLintConfig{}
	if c.IsCommentFilter() {
		t.Error("nil CommentFilter should default to false")
	}
}

func TestYAMLLintConfig_IsCommentFilter_True(t *testing.T) {
	c := &config.YAMLLintConfig{CommentFilter: ptrBool(true)}
	if !c.IsCommentFilter() {
		t.Error("expected true")
	}
}

// EncodingConfig

func TestEncodingConfig_IsEnabled_Nil(t *testing.T) {
	c := &config.EncodingConfig{}
	if !c.IsEnabled() {
		t.Error("nil Enabled should default to true")
	}
}

func TestEncodingConfig_IsEnabled_False(t *testing.T) {
	c := &config.EncodingConfig{Enabled: ptrBool(false)}
	if c.IsEnabled() {
		t.Error("expected false")
	}
}

func TestEncodingConfig_IsRequireUTF8_Nil(t *testing.T) {
	c := &config.EncodingConfig{}
	if !c.IsRequireUTF8() {
		t.Error("nil RequireUTF8 should default to true")
	}
}

func TestEncodingConfig_IsRequireUTF8_False(t *testing.T) {
	c := &config.EncodingConfig{RequireUTF8: ptrBool(false)}
	if c.IsRequireUTF8() {
		t.Error("expected false")
	}
}

func TestEncodingConfig_IsNoInvisibleChars_Nil(t *testing.T) {
	c := &config.EncodingConfig{}
	if c.IsNoInvisibleChars() {
		t.Error("nil NoInvisibleChars should default to false")
	}
}

func TestEncodingConfig_IsNoInvisibleChars_True(t *testing.T) {
	c := &config.EncodingConfig{NoInvisibleChars: ptrBool(true)}
	if !c.IsNoInvisibleChars() {
		t.Error("expected true")
	}
}

func TestEncodingConfig_IsNoAmbiguousChars_Nil(t *testing.T) {
	c := &config.EncodingConfig{}
	if c.IsNoAmbiguousChars() {
		t.Error("nil NoAmbiguousChars should default to false")
	}
}

func TestEncodingConfig_IsNoAmbiguousChars_True(t *testing.T) {
	c := &config.EncodingConfig{NoAmbiguousChars: ptrBool(true)}
	if !c.IsNoAmbiguousChars() {
		t.Error("expected true")
	}
}

// BinaryFileConfig

func TestBinaryFileConfig_IsEnabled_Nil(t *testing.T) {
	c := &config.BinaryFileConfig{}
	if !c.IsEnabled() {
		t.Error("nil Enabled should default to true")
	}
}

func TestBinaryFileConfig_IsEnabled_False(t *testing.T) {
	c := &config.BinaryFileConfig{Enabled: ptrBool(false)}
	if c.IsEnabled() {
		t.Error("expected false")
	}
}

// EditorConfigConfig

func TestEditorConfigConfig_IsEnabled_Nil(t *testing.T) {
	c := &config.EditorConfigConfig{}
	if !c.IsEnabled() {
		t.Error("nil Enabled should default to true")
	}
}

func TestEditorConfigConfig_IsEnabled_False(t *testing.T) {
	c := &config.EditorConfigConfig{Enabled: ptrBool(false)}
	if c.IsEnabled() {
		t.Error("expected false")
	}
}

// CommitMessageConfig

func TestCommitMessageConfig_IsEnabled_Nil(t *testing.T) {
	c := &config.CommitMessageConfig{}
	if !c.IsEnabled() {
		t.Error("nil Enabled should default to true")
	}
}

func TestCommitMessageConfig_IsEnabled_False(t *testing.T) {
	c := &config.CommitMessageConfig{Enabled: ptrBool(false)}
	if c.IsEnabled() {
		t.Error("expected false")
	}
}

func TestCommitMessageConfig_IsNoEmoji_Nil(t *testing.T) {
	c := &config.CommitMessageConfig{}
	if c.IsNoEmoji() {
		t.Error("nil NoEmoji should default to false")
	}
}

func TestCommitMessageConfig_IsNoEmoji_True(t *testing.T) {
	c := &config.CommitMessageConfig{NoEmoji: ptrBool(true)}
	if !c.IsNoEmoji() {
		t.Error("expected true")
	}
}
