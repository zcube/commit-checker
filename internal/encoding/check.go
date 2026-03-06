package encoding

import (
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

// CheckUTF8: chardet을 사용하여 콘텐츠가 유효한 UTF-8인지 검증.
// 유효 여부와 감지된 charset을 반환.
func CheckUTF8(content []byte) Result {
	hasBOM := len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF

	if len(content) == 0 {
		return Result{Valid: true, HasBOM: false, DetectedCharset: "UTF-8", Confidence: 100}
	}

	det := chardet.NewTextDetector()
	best, err := det.DetectBest(content)
	if err != nil {
		// chardet 실패 시 fallback: 모든 바이트가 유효한 UTF-8인지 확인
		return Result{Valid: isValidUTF8(content), HasBOM: hasBOM, DetectedCharset: "unknown"}
	}

	isUTF8 := best.Charset == "UTF-8" || best.Charset == "ISO-8859-1" && isValidUTF8(content)

	return Result{
		Valid:           isUTF8,
		HasBOM:          hasBOM,
		DetectedCharset: best.Charset,
		Confidence:      best.Confidence,
	}
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

// IsBinary: mimetype 패키지를 사용하여 콘텐츠가 바이너리인지 확인.
// magic bytes 기반으로 MIME 타입을 감지하고, text/plain 계열이 아니면 바이너리로 판단.
// JSON, XML, HTML 등 text/plain을 부모로 가지는 타입은 텍스트로 처리.
func IsBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}
	mtype := mimetype.Detect(content)
	for m := mtype; m != nil; m = m.Parent() {
		if m.Is("text/plain") {
			return false
		}
	}
	return true
}
