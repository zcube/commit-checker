// Package version 은 빌드 시 주입된 버전 정보를 제공합니다.
// 릴리즈 빌드: go build -ldflags "-X github.com/zcube/commit-checker/internal/version.Version=v1.2.3 ..."
// ldflags 가 없는 빌드(go install 등)는 모듈 빌드 정보에서 버전을 유도합니다.
package version

import "runtime/debug"

// 빌드 시 -ldflags 로 주입되는 변수들.
// 기본값은 개발 빌드임을 나타냄.
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

// init 은 ldflags 미주입 시 go install 이 기록한 모듈 버전과
// VCS 정보로 기본값을 보완합니다.
func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	// go install module@v1.2.3 으로 설치하면 Main.Version 에 태그가 기록됨.
	// 소스 트리에서 go build/run 하면 "(devel)" 이므로 보완하지 않음.
	if Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
	if Commit == "none" {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				Commit = s.Value
			}
			if s.Key == "vcs.time" && s.Value != "" && BuildTime == "unknown" {
				BuildTime = s.Value
			}
		}
	}
}
