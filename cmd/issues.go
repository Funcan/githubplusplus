package cmd

import (
	"context"
	"fmt"
	"os"
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
		return printIssues(ctx, client, w, owner, repoName)
	})
	if flushErr := w.Flush(); flushErr != nil && err == nil {
		return flushErr
	}
	return err
}

func printIssues(ctx context.Context, client *ghclient.Client, w *tabwriter.Writer, owner, repoName string) error {
	issues, err := client.ListOpenIssues(ctx, owner, repoName)
	if err != nil {
		return err
	}

	for _, issue := range issues {
		fmt.Fprintf(w, "%s/%s#%d\t%s\t%s\n",
			owner,
			repoName,
			issue.GetNumber(),
			issue.GetTitle(),
			issue.GetHTMLURL(),
		)
	}
	return nil
}
