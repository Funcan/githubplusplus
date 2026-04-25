package cmd

import (
	"context"
	"text/tabwriter"

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

func init() {
	rootCmd.AddCommand(prsCmd)
}

func runPRs(cmd *cobra.Command, args []string) error {
	return runListing(args, func(ctx context.Context, client *ghclient.Client, w *tabwriter.Writer, owner, repoName string) error {
		prs, err := client.ListOpenPRs(ctx, owner, repoName)
		if err != nil {
			return err
		}
		printItems(w, owner, repoName, prs)
		return nil
	})
}
