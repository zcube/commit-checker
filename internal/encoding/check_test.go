package encoding

import (
	"testing"
)

func TestCheckUTF8_ValidASCII(t *testing.T) {
	result := CheckUTF8([]byte("hello world"))
	if !result.Valid {
		t.Error("expected valid UTF-8 for ASCII")
	}
	if result.HasBOM {
		t.Error("expected no BOM")
	}
}

func TestCheckUTF8_ValidUTF8(t *testing.T) {
	result := CheckUTF8([]byte("한국어 텍스트"))
	if !result.Valid {
		t.Error("expected valid UTF-8 for Korean text")
	}
}

func TestCheckUTF8_WithBOM(t *testing.T) {
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte("hello")...)
	result := CheckUTF8(content)
	if !result.Valid {
		t.Error("expected valid UTF-8 with BOM")
	}
	if !result.HasBOM {
		t.Error("expected BOM to be detected")
	}
}

func TestCheckUTF8_InvalidBytes(t *testing.T) {
	// Invalid UTF-8 byte sequence
	content := []byte{0xFF, 0xFE, 0x68, 0x65, 0x6C, 0x6C, 0x6F}
	result := CheckUTF8(content)
	if result.Valid {
		t.Error("expected invalid UTF-8 for bad bytes")
	}
}

func TestCheckUTF8_Latin1(t *testing.T) {
	// Latin-1 encoded text (not valid UTF-8)
	content := []byte{0xC4, 0xD6, 0xDC} // ÄÖÜ in Latin-1
	result := CheckUTF8(content)
	if result.Valid {
		t.Error("expected invalid UTF-8 for Latin-1 text")
	}
}

func TestCheckUTF8_Empty(t *testing.T) {
	result := CheckUTF8([]byte{})
	if !result.Valid {
		t.Error("expected valid for empty input")
	}
}

func TestIsBinary_TextFile(t *testing.T) {
	if IsBinary([]byte("hello world\nfoo bar\n")) {
		t.Error("expected text to not be binary")
	}
}

func TestIsBinary_BinaryFile(t *testing.T) {
	content := []byte{0x7f, 'E', 'L', 'F', 0, 0, 0, 0}
	if !IsBinary(content) {
		t.Error("expected ELF content to be binary")
	}
}

func TestIsBinary_Empty(t *testing.T) {
	if IsBinary([]byte{}) {
		t.Error("expected empty to not be binary")
	}
}
