// Package version 은 빌드 시 주입된 버전 정보를 제공합니다.
// 릴리즈 빌드: go build -ldflags "-X github.com/zcube/commit-checker/internal/version.Version=v1.2.3 ..."
package version

// 빌드 시 -ldflags 로 주입되는 변수들.
// 기본값은 개발 빌드임을 나타냄.
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)
