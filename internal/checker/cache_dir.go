package checker

import (
	"context"
	"path/filepath"

	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/gitdiff"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/pathutil"
	"github.com/zcube/commit-checker/pkg/cachedir"
)

// CheckCacheDirStaged 는 staged diff 에서 캐시/빌드 디렉터리(node_modules, dist 등) 안의 파일이
// 커밋되려는 경우를 차단합니다. 검증기는 부모 디렉터리 인디케이터를 확인하여 false positive 를 줄입니다.
func CheckCacheDirStaged(ctx context.Context, cfg *config.Config) ([]string, error) {
	if !cfg.CacheDir.IsEnabled() {
		return nil, nil
	}

	repoRoot, err := cachedir.FindRepoRoot(".")
	if err != nil {
		return nil, nil
	}

	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		return nil, err
	}

	ignoreDirs := toSet(cfg.CacheDir.IgnoreDirs)

	var errs []string
	seen := make(map[string]bool)
	for _, d := range diffs {
		// 취소 시 남은 파일 검사를 중단
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if d.IsDeleted {
			continue
		}
		if pathutil.MatchesAny(d.Path, cfg.Exceptions.GlobalIgnore) {
			continue
		}
		absPath := filepath.Join(repoRoot, d.Path)
		cacheDir, ok := cachedir.FindCacheDirAncestor(repoRoot, absPath)
		if !ok {
			continue
		}
		if ignoreDirs[filepath.Base(cacheDir)] {
			continue
		}
		// 같은 캐시 디렉터리 안의 여러 파일이면 한 번만 보고
		rel, _ := filepath.Rel(repoRoot, cacheDir)
		if seen[rel] {
			continue
		}
		seen[rel] = true
		errs = append(errs, i18n.T("diff.cache_dir_staged", map[string]any{
			"Path":     d.Path,
			"CacheDir": rel,
		}))
	}
	return errs, nil
}

// CheckCacheDirCommitted 는 추적 중인 파일들 중 캐시/빌드 디렉터리에 들어 있는 파일을 보고합니다.
// run 커맨드용으로, 이미 커밋된 캐시 산출물을 감사합니다.
func CheckCacheDirCommitted(ctx context.Context, cfg *config.Config) ([]string, error) {
	if !cfg.CacheDir.IsEnabled() {
		return nil, nil
	}

	repoRoot, err := cachedir.FindRepoRoot(".")
	if err != nil {
		return nil, nil
	}

	cacheDirs := cachedir.FindCacheDirsInRepo(repoRoot)
	if len(cacheDirs) == 0 {
		return nil, nil
	}

	ignoreDirs := toSet(cfg.CacheDir.IgnoreDirs)

	var errs []string
	for _, dir := range cacheDirs {
		// 취소 시 남은 디렉터리 검사를 중단
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if ignoreDirs[filepath.Base(dir)] {
			continue
		}
		tracked, err := cachedir.ListTrackedEntries(repoRoot, dir)
		if err != nil || len(tracked) == 0 {
			continue
		}
		rel, _ := filepath.Rel(repoRoot, dir)
		if pathutil.MatchesAny(rel, cfg.Exceptions.GlobalIgnore) {
			continue
		}
		errs = append(errs, i18n.T("diff.cache_dir_committed", map[string]any{
			"CacheDir": rel,
			"Count":    len(tracked),
		}))
	}
	return errs, nil
}

func toSet(names []string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}
