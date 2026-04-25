package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
)

// listItem is satisfied by GitHub issue and pull-request objects, which share
// the same tab-separated output format.
type listItem interface {
	GetNumber() int
	GetTitle() string
	GetHTMLURL() string
}

// runListing is the shared scaffold for the issues and prs commands.
// It sets up the tabwriter, iterates over repos, and flushes at the end.
func runListing(args []string, fetch func(context.Context, *ghclient.Client, *tabwriter.Writer, string, string) error) error {
	if len(args) == 0 {
		args = []string{"."}
	}
	ctx := context.Background()
	client, err := ghclient.New()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	err = forEachExpandedRepo(ctx, client, args, "one or more repos could not be listed", func(owner, repoName string) error {
		return fetch(ctx, client, w, owner, repoName)
	})
	if flushErr := w.Flush(); flushErr != nil && err == nil {
		return flushErr
	}
	return err
}

// printItems writes each item to w in tab-separated owner/repo#N title url format.
func printItems[T listItem](w *tabwriter.Writer, owner, repoName string, items []T) {
	for _, item := range items {
		fmt.Fprintf(w, "%s/%s#%d\t%s\t%s\n",
			owner,
			repoName,
			item.GetNumber(),
			item.GetTitle(),
			item.GetHTMLURL(),
		)
	}
}
