package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
)

var forkPRsCmd = &cobra.Command{
	Use:   "prs [repo...]",
	Short: "List open pull requests from one or more forks to their upstream",
	Long: `List the URLs of all open pull requests from each fork to its upstream.

Each argument may be a local path to a git checkout or a GitHub "owner/repo"
reference. If no arguments are given the repository in the current directory
is used.`,
	RunE: runForkPRs,
}

func init() {
	forkCmd.AddCommand(forkPRsCmd)
}

func runForkPRs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"."}
	}

	ctx := context.Background()
	client, err := ghclient.New()
	if err != nil {
		return err
	}

	var anyErr bool
	for _, arg := range args {
		if err := printForkPRs(ctx, client, arg); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", arg, err)
			anyErr = true
		}
	}
	if anyErr {
		return fmt.Errorf("one or more repos could not be checked")
	}
	return nil
}

func printForkPRs(ctx context.Context, client *ghclient.Client, arg string) error {
	_, owner, repoName, err := resolveRepoArg(arg)
	if err != nil {
		return err
	}

	repo, err := client.GetRepo(ctx, owner, repoName)
	if err != nil {
		return err
	}

	parent, err := validateFork(repo, owner, repoName)
	if err != nil {
		return err
	}

	upstreamOwner := parent.GetOwner().GetLogin()
	upstreamRepo := parent.GetName()

	prs, err := client.ListOpenPRsToUpstream(ctx, upstreamOwner, upstreamRepo, owner, repoName)
	if err != nil {
		return err
	}

	for _, pr := range prs {
		fmt.Println(pr.GetHTMLURL())
	}

	return nil
}
