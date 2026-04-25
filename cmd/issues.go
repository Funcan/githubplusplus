package cmd

import (
	"context"
	"text/tabwriter"

	"github.com/spf13/cobra"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
)

var issuesCmd = &cobra.Command{
	Use:   "issues [repo...]",
	Short: "List open issues for one or more repos",
	Long: `List open issues for one or more repositories.

Each argument may be a local path to a git checkout or a GitHub "owner/repo"
reference. If no arguments are given the repository in the current directory
is used.`,
	RunE: runIssues,
}

func init() {
	rootCmd.AddCommand(issuesCmd)
}

func runIssues(cmd *cobra.Command, args []string) error {
	return runListing(args, func(ctx context.Context, client *ghclient.Client, w *tabwriter.Writer, owner, repoName string) error {
		issues, err := client.ListOpenIssues(ctx, owner, repoName)
		if err != nil {
			return err
		}
		printItems(w, owner, repoName, issues)
		return nil
	})
}
