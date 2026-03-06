// Package emoji: 커밋 메시지 및 소스 코드에서 이모지를 감지하는 패키지.
package emoji

import "unicode"

// IsEmoji: 주어진 룬이 이모지 문자인지 확인.
// 이모티콘, 기호, 딩뱃, 교통, 국기, 보충 픽토그래프 등 일반적인 이모지 범위를 포함.
func IsEmoji(r rune) bool {
	switch {
	// Variation Selector-16 (이모지 렌더링 강제)
	case r == 0xFE0F:
		return true
	// Zero Width Joiner (이모지 시퀀스 결합)
	case r == 0x200D:
		return true
	// Combining Enclosing Keycap (키캡 결합 기호)
	case r == 0x20E3:
		return true
	// 기타 기호 (Misc Symbols)
	case r >= 0x2600 && r <= 0x26FF:
		return true
	// 딩뱃 (Dingbats)
	case r >= 0x2700 && r <= 0x27BF:
		return true
	// CJK 기호 및 일부 이모지
	case r == 0x2B50 || r == 0x2B55 || r == 0x2B1B || r == 0x2B1C:
		return true
	// 일반 텍스트 문자로 취급 (이모지 아님)
	case r == 0x00A9 || r == 0x00AE: // © ®
		return false
	case r == 0x203C || r == 0x2049: // ‼ ⁉
		return true
	case r >= 0x2122 && r <= 0x2199: // ™ ↔ 등
		return true
	case r >= 0x21A9 && r <= 0x21AA: // ↩ ↪
		return true
	case r >= 0x231A && r <= 0x231B: // ⌚ ⌛
		return true
	case r == 0x2328: // ⌨
		return true
	case r == 0x23CF: // ⏏
		return true
	case r >= 0x23E9 && r <= 0x23F3: // ⏩-⏳
		return true
	case r >= 0x23F8 && r <= 0x23FA: // ⏸-⏺
		return true
	case r == 0x25AA || r == 0x25AB: // ▪ ▫
		return true
	case r == 0x25B6 || r == 0x25C0: // ▶ ◀
		return true
	case r >= 0x25FB && r <= 0x25FE: // ◻-◾
		return true
	// 이모티콘 (Emoticons)
	case r >= 0x1F600 && r <= 0x1F64F:
		return true
	// 기타 기호 및 픽토그래프
	case r >= 0x1F300 && r <= 0x1F5FF:
		return true
	// 교통 및 지도 기호
	case r >= 0x1F680 && r <= 0x1F6FF:
		return true
	// 지역 표시 기호 (국기)
	case r >= 0x1F1E0 && r <= 0x1F1FF:
		return true
	// 보충 기호 및 픽토그래프
	case r >= 0x1F900 && r <= 0x1F9FF:
		return true
	// 기호 및 픽토그래프 확장-A
	case r >= 0x1FA70 && r <= 0x1FAFF:
		return true
	// 체스 기호
	case r >= 0x1FA00 && r <= 0x1FA6F:
		return true
	// 피부색 수정자
	case r >= 0x1F3FB && r <= 0x1F3FF:
		return true
	// 태그 (국기 시퀀스에 사용)
	case r >= 0xE0020 && r <= 0xE007F:
		return true
	}
	return false
}

// EmojiInfo: 감지된 이모지 문자 정보.
type EmojiInfo struct {
	Line int
	Col  int
	Char string
	Code rune
}

// FindEmojis: 텍스트에서 모든 이모지 발생 위치를 스캔하여 반환.
func FindEmojis(text string) []EmojiInfo {
	var results []EmojiInfo
	line := 1
	col := 0
	for _, r := range text {
		col++
		if r == '\n' {
			line++
			col = 0
			continue
		}
		// 변형 선택자(FE0F) 및 ZWJ는 시퀀스의 일부이므로 단독으로 건너뜀
		if r == 0xFE0F || r == 0x200D {
			continue
		}
		if IsEmoji(r) && !unicode.IsSpace(r) {
			results = append(results, EmojiInfo{
				Line: line,
				Col:  col,
				Char: string(r),
				Code: r,
			})
		}
	}
	return results
}

// ContainsEmoji: 텍스트에 이모지가 포함되어 있는지 확인.
func ContainsEmoji(text string) bool {
	for _, r := range text {
		if r == 0xFE0F || r == 0x200D {
			continue
		}
		if IsEmoji(r) && !unicode.IsSpace(r) {
			return true
		}
	}
	return false
}
