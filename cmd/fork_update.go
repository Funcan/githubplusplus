package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
	"github.com/Funcan/githubplusplus/internal/git"
)

var forkUpdateCmd = &cobra.Command{
	Use:   "update [repo...]",
	Short: "Pull upstream changes into one or more forks",
	Long: `Pull upstream changes into one or more forked repositories.

Each argument may be a local path to a git checkout or a GitHub "owner/repo"
reference. If no arguments are given the repository in the current directory
is used.

For each target the fork is synced with its upstream via the GitHub API.  If
the target was given as a local path, the local clone is also updated with
"git fetch origin" followed by a fast-forward merge of the fork's default
branch (only when that branch is currently checked out).`,
	RunE: runForkUpdate,
}

func init() {
	forkCmd.AddCommand(forkUpdateCmd)
}

func runForkUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"."}
	}

	ctx := context.Background()
	client, err := ghclient.New()
	if err != nil {
		return err
	}

	return forEachRepoArg(args, "one or more repos could not be updated", func(arg string) error {
		return updateFork(ctx, client, arg)
	})
}

func updateFork(ctx context.Context, client *ghclient.Client, arg string) error {
	localPath, owner, repoName, err := resolveRepoArg(arg)
	if err != nil {
		return err
	}

	repo, err := client.GetRepo(ctx, owner, repoName)
	if err != nil {
		return err
	}

	if _, err := validateFork(repo, owner, repoName); err != nil {
		return err
	}

	defaultBranch := repo.GetDefaultBranch()

	fmt.Printf("syncing %s/%s (branch %q) with upstream...\n", owner, repoName, defaultBranch)
	result, err := client.MergeUpstream(ctx, owner, repoName, defaultBranch)
	if err != nil {
		return err
	}
	switch result.MergeType {
	case "none":
		fmt.Printf("%s/%s is already up to date\n", owner, repoName)
	default:
		fmt.Printf("%s/%s: %s (%s)\n", owner, repoName, result.MergeType, result.Message)
	}

	if localPath == "" {
		return nil
	}

	fmt.Printf("fetching origin in %s...\n", localPath)
	if err := git.Fetch(localPath, "origin"); err != nil {
		return err
	}

	currentBranch, err := git.GetCurrentBranch(localPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s: could not determine current branch, skipping merge: %v\n", localPath, err)
		return nil
	}

	if currentBranch != defaultBranch {
		fmt.Printf("note: %s is on branch %q, not %q; skipping fast-forward\n", localPath, currentBranch, defaultBranch)
		return nil
	}

	ref := "origin/" + defaultBranch
	fmt.Printf("fast-forwarding %s to %s...\n", localPath, ref)
	if err := git.MergeFFOnly(localPath, ref); err != nil {
		return err
	}

	return nil
}
