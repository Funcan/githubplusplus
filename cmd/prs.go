package cmd

import (
	"context"
	"text/tabwriter"

	gogithub "github.com/google/go-github/v72/github"
	"github.com/spf13/cobra"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
)

var prsCmd = &cobra.Command{
	Use:   "prs [repo...]",
	Short: "List open pull requests for one or more repos",
	Long: `List open pull requests for one or more repositories.

Each argument may be a local path to a git checkout or a GitHub "owner/repo"
reference. If no arguments are given the repository in the current directory
is used.`,
	RunE: runPRs,
}

var prsReadyOnly bool

func init() {
	rootCmd.AddCommand(prsCmd)
	prsCmd.Flags().BoolVar(&prsReadyOnly, "ready", false, "Only list PRs that are ready to merge (checks passed, no conflicts, reviews approved)")
}

func runPRs(cmd *cobra.Command, args []string) error {
	return runListing(args, func(ctx context.Context, client *ghclient.Client, w *tabwriter.Writer, owner, repoName string) error {
		prs, err := client.ListOpenPRs(ctx, owner, repoName)
		if err != nil {
			return err
		}
		if prsReadyOnly {
			prs, err = fetchReadyPRs(ctx, client, owner, repoName, prs)
			if err != nil {
				return err
			}
		}
		printItems(w, owner, repoName, prs)
		return nil
	})
}

// fetchReadyPRs re-fetches each PR individually (the list endpoint does not
// populate mergeable_state) and returns those that are ready to merge.
func fetchReadyPRs(ctx context.Context, client *ghclient.Client, owner, repo string, prs []*gogithub.PullRequest) ([]*gogithub.PullRequest, error) {
	full := make([]*gogithub.PullRequest, 0, len(prs))
	for _, pr := range prs {
		fetched, err := client.GetPR(ctx, owner, repo, pr.GetNumber())
		if err != nil {
			return nil, err
		}
		full = append(full, fetched)
	}
	return filterReadyPRs(full), nil
}

// filterReadyPRs returns only PRs whose mergeable_state is "clean", meaning
// all required checks have passed, there are no merge conflicts, and any
// required reviews have been approved.
func filterReadyPRs(prs []*gogithub.PullRequest) []*gogithub.PullRequest {
	var out []*gogithub.PullRequest
	for _, pr := range prs {
		if pr.GetMergeableState() == "clean" {
			out = append(out, pr)
		}
	}
	return out
}
