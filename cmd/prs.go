package cmd

import (
	"context"
	"fmt"
	"os"
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
	if len(args) == 0 {
		args = []string{"."}
	}

	ctx := context.Background()
	client, err := ghclient.New()
	if err != nil {
		return err
	}

	return forEachRepoArg(args, "one or more repos could not be listed", func(arg string) error {
		return printPRs(ctx, client, arg)
	})
}

func printPRs(ctx context.Context, client *ghclient.Client, arg string) error {
	_, owner, repoName, err := resolveRepoArg(arg)
	if err != nil {
		return err
	}

	prs, err := client.ListOpenPRs(ctx, owner, repoName)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, pr := range prs {
		fmt.Fprintf(w, "%s/%s#%d\t%s\t%s\n",
			owner,
			repoName,
			pr.GetNumber(),
			pr.GetTitle(),
			pr.GetHTMLURL(),
		)
	}
	return w.Flush()
}
