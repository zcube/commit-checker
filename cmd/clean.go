package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/pkg/cachedir"
)

var cleanYes bool

var cleanCmd = &cobra.Command{
	Use: "clean",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		repoRoot, err := cachedir.FindRepoRoot(cwd)
		if err != nil {
			fmt.Fprintln(os.Stderr, i18n.T("cmd.clean.not_in_repo", map[string]any{"Path": cwd}))
			os.Exit(1)
		}

		if !globalQuiet {
			fmt.Fprintln(os.Stderr, i18n.T("cmd.clean.scanning", map[string]any{"Path": repoRoot}))
		}

		dirs := cachedir.FindCacheDirsInRepo(repoRoot)
		if len(dirs) == 0 {
			fmt.Println(i18n.T("cmd.clean.no_cache_dirs", nil))
			return nil
		}

		type entry struct {
			abs     string
			rel     string
			size    int64
			tracked int
		}
		entries := make([]entry, 0, len(dirs))
		var total int64
		for _, d := range dirs {
			rel, _ := filepath.Rel(repoRoot, d)
			size := cachedir.GetDirSize(d)
			tracked, _ := cachedir.ListTrackedEntries(repoRoot, d)
			entries = append(entries, entry{abs: d, rel: rel, size: size, tracked: len(tracked)})
			total += size
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].size > entries[j].size })

		fmt.Println(i18n.T("cmd.clean.found_header", map[string]any{
			"Count": len(entries),
			"Size":  cachedir.FormatBytes(total),
		}))
		for _, e := range entries {
			tracked := ""
			if e.tracked > 0 {
				tracked = i18n.T("cmd.clean.entry_tracked", map[string]any{"Count": e.tracked})
			}
			fmt.Println(i18n.T("cmd.clean.entry", map[string]any{
				"Path":    e.rel,
				"Size":    cachedir.FormatBytes(e.size),
				"Tracked": tracked,
			}))
		}

		if !cleanYes {
			fmt.Println(i18n.T("cmd.clean.dry_run_hint", nil))
			return nil
		}

		var freed int64
		var cleaned int
		for _, e := range entries {
			untracked, err := cachedir.ListUntrackedEntries(repoRoot, e.abs)
			if err != nil {
				continue
			}
			for _, p := range untracked {
				size := cachedir.GetDirSize(p)
				if err := os.RemoveAll(p); err == nil {
					freed += size
				}
			}
			cleaned++
		}
		fmt.Println(i18n.T("cmd.clean.cleaned_summary", map[string]any{
			"Size":  cachedir.FormatBytes(freed),
			"Count": cleaned,
		}))
		return nil
	},
}

func init() {
	cleanCmd.Short = i18n.T("cmd.clean.short", nil)
	cleanCmd.Long = i18n.T("cmd.clean.long", nil)
	cleanCmd.Flags().BoolVar(&cleanYes, "yes", false, "delete untracked files (without this flag, runs in dry-run)")
	rootCmd.AddCommand(cleanCmd)
}
