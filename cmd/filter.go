package cmd

import (
	"strings"

	gogithub "github.com/google/go-github/v72/github"
	"github.com/spf13/cobra"
)

// filterFlags holds the filter options shared between list and clone.
type filterFlags struct {
	forks      bool
	noForks    bool
	language   string
	archived   bool
	noArchived bool
}

// addFilterFlags attaches shared filter flags to a command.
func addFilterFlags(cmd *cobra.Command, f *filterFlags) {
	cmd.Flags().BoolVar(&f.forks, "forks", false, "Include only forks")
	cmd.Flags().BoolVar(&f.noForks, "no-forks", false, "Exclude forks")
	cmd.Flags().StringVar(&f.language, "language", "", "Filter by primary language")
	cmd.Flags().BoolVar(&f.archived, "archived", false, "Include only archived repos")
	cmd.Flags().BoolVar(&f.noArchived, "no-archived", false, "Exclude archived repos")
}

// applyFilters returns only the repos that match all active filters.
func applyFilters(repos []*gogithub.Repository, f *filterFlags) []*gogithub.Repository {
	var out []*gogithub.Repository
	for _, r := range repos {
		if f.forks && !r.GetFork() {
			continue
		}
		if f.noForks && r.GetFork() {
			continue
		}
		if f.language != "" && !strings.EqualFold(r.GetLanguage(), f.language) {
			continue
		}
		if f.archived && !r.GetArchived() {
			continue
		}
		if f.noArchived && r.GetArchived() {
			continue
		}
		out = append(out, r)
	}
	return out
}
