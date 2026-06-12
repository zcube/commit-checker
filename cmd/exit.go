package cmd

import (
	"errors"
	"io"

	"github.com/charmbracelet/fang"
)

// errSilentExit: 위반/실패 내용을 이미 stderr 에 출력한 뒤 종료 코드 1 만 필요할 때
// RunE 가 os.Exit 대신 반환하는 sentinel 에러. Execute() 가 이 에러를 받으면
// 추가 출력 없이 종료 코드 1 로 끝낸다 (fang 의 에러 렌더링은 silentErrorHandler 가 억제).
var errSilentExit = errors.New("checks failed")

// silentErrorHandler: fang 의 기본 에러 렌더링을 감싸는 핸들러.
// errSilentExit 는 RunE 가 이미 stderr 에 보고를 마친 상태이므로 중복 출력하지 않고,
// 그 외 에러는 fang 기본 렌더링을 그대로 사용한다.
func silentErrorHandler(w io.Writer, styles fang.Styles, err error) {
	if errors.Is(err, errSilentExit) {
		return
	}
	fang.DefaultErrorHandler(w, styles, err)
}
