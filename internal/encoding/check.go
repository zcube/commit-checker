package encoding

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"

	"github.com/gabriel-vasile/mimetype"
	"github.com/saintfish/chardet"
)

// Result: 파일 인코딩 검사 결과.
type Result struct {
	Valid           bool
	HasBOM          bool
	DetectedCharset string
	Confidence      int
}

// CheckUTF8: 콘텐츠가 유효한 UTF-8인지 검증.
// 유효성은 바이트 시퀀스로 직접 판단하고, chardet은 에러 메시지용 charset 이름 제공에만 사용.
func CheckUTF8(content []byte) Result {
	hasBOM := len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF

	if len(content) == 0 {
		return Result{Valid: true, HasBOM: false, DetectedCharset: "UTF-8", Confidence: 100}
	}

	valid := isValidUTF8(content)

	// 유효하지 않은 경우에만 chardet으로 감지된 charset을 에러 메시지에 제공
	if !valid {
		det := chardet.NewTextDetector()
		if best, err := det.DetectBest(content); err == nil {
			return Result{Valid: false, HasBOM: hasBOM, DetectedCharset: best.Charset, Confidence: best.Confidence}
		}
		return Result{Valid: false, HasBOM: hasBOM, DetectedCharset: "unknown"}
	}

	return Result{Valid: true, HasBOM: hasBOM, DetectedCharset: "UTF-8", Confidence: 100}
}

// isValidUTF8: 모든 바이트가 유효한 UTF-8 시퀀스를 구성하는지 확인.
func isValidUTF8(content []byte) bool {
	for i := 0; i < len(content); {
		if content[i] < 0x80 {
			i++
			continue
		}
		// 멀티바이트 UTF-8 시퀀스
		size := 0
		switch {
		case content[i]&0xE0 == 0xC0:
			size = 2
		case content[i]&0xF0 == 0xE0:
			size = 3
		case content[i]&0xF8 == 0xF0:
			size = 4
		default:
			return false
		}
		if i+size > len(content) {
			return false
		}
		for j := 1; j < size; j++ {
			if content[i+j]&0xC0 != 0x80 {
				return false
			}
		}
		i += size
	}
	return true
}

// IsBinary: 콘텐츠가 바이너리인지 확인.
// 우선 Go 표준 라이브러리의 포맷 파서로 실행파일 형식을 감지하고,
// 이후 mimetype으로 이미지/아카이브 등 기타 바이너리를 판별.
//   - debug/elf:   Linux/BSD ELF 실행파일
//   - debug/macho: macOS Mach-O 실행파일
//   - debug/pe:    Windows PE 실행파일
//   - mimetype:    그 외 바이너리 (이미지, ZIP, PDF 등)
func IsBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}
	r := bytes.NewReader(content)

	// ELF 실행파일 (Linux/BSD)
	if f, err := elf.NewFile(r); err == nil {
		_ = f.Close()
		return true
	}
	// Mach-O 실행파일 (macOS)
	if f, err := macho.NewFile(r); err == nil {
		_ = f.Close()
		return true
	}
	// PE 실행파일 (Windows)
	if f, err := pe.NewFile(r); err == nil {
		_ = f.Close()
		return true
	}

	// 기타 바이너리 (이미지, 아카이브 등) - mimetype으로 판별
	mtype := mimetype.Detect(content)
	for m := mtype; m != nil; m = m.Parent() {
		if m.Is("text/plain") {
			return false
		}
	}
	return true
}
