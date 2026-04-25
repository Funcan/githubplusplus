package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
)

var forkStatusIgnore []string

var forkStatusCmd = &cobra.Command{
	Use:   "status [repo...]",
	Short: "Show whether one or more forks are up to date with their upstream",
	Long: `Compare each fork's default branch against its upstream and print the status.

Each argument may be a local path to a git checkout or a GitHub "owner/repo"
reference. If no arguments are given the repository in the current directory
is used.

Possible statuses:
  identical  The fork is fully up to date with upstream.
  behind     The fork is missing commits from upstream (needs updating).
  ahead      The fork has commits not present in upstream.
  diverged   The fork and upstream have each moved forward independently.`,
	RunE: runForkStatus,
}

func init() {
	forkCmd.AddCommand(forkStatusCmd)
	forkStatusCmd.Flags().StringSliceVar(&forkStatusIgnore, "ignore-status", nil, "Comma-separated list of statuses to suppress (identical,behind,ahead,diverged,PRsOpen)")
}

func runForkStatus(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"."}
	}

	ctx := context.Background()
	client, err := ghclient.New()
	if err != nil {
		return err
	}

	var anyErr bool
	ignored := make(map[string]bool, len(forkStatusIgnore))
	for _, s := range forkStatusIgnore {
		ignored[s] = true
	}
	for _, arg := range args {
		if err := printForkStatus(ctx, client, arg, ignored); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", arg, err)
			anyErr = true
		}
	}
	if anyErr {
		return fmt.Errorf("one or more repos could not be checked")
	}
	return nil
}

func printForkStatus(ctx context.Context, client *ghclient.Client, arg string, ignored map[string]bool) error {
	_, owner, repoName, err := resolveRepoArg(arg)
	if err != nil {
		return err
	}

	repo, err := client.GetRepo(ctx, owner, repoName)
	if err != nil {
		return err
	}

	if !repo.GetFork() {
		return fmt.Errorf("%s/%s is not a fork", owner, repoName)
	}

	parent := repo.GetParent()
	if parent == nil {
		return fmt.Errorf("%s/%s: upstream parent metadata unavailable", owner, repoName)
	}

	forkBranch := repo.GetDefaultBranch()
	upstreamOwner := parent.GetOwner().GetLogin()
	upstreamRepo := parent.GetName()
	upstreamBranch := parent.GetDefaultBranch()

	cmp, err := client.CompareWithUpstream(ctx, owner, repoName, forkBranch, upstreamOwner, upstreamBranch)
	if err != nil {
		return err
	}

	openPRs, err := client.CountOpenPRsToUpstream(ctx, upstreamOwner, upstreamRepo, owner, repoName)
	if err != nil {
		return err
	}

	var labels []string

	if !ignored[cmp.Status] {
		switch cmp.Status {
		case "identical":
			labels = append(labels, "identical")
		case "behind":
			labels = append(labels, fmt.Sprintf("behind (%d commit(s) behind upstream)", cmp.BehindBy))
		case "ahead":
			labels = append(labels, fmt.Sprintf("ahead (%d commit(s) ahead of upstream)", cmp.AheadBy))
		case "diverged":
			labels = append(labels, fmt.Sprintf("diverged (%d ahead, %d behind)", cmp.AheadBy, cmp.BehindBy))
		default:
			labels = append(labels, cmp.Status)
		}
	}

	if openPRs > 0 && !ignored["PRsOpen"] {
		labels = append(labels, fmt.Sprintf("PRsOpen (%d)", openPRs))
	}

	if len(labels) > 0 {
		fmt.Printf("%s/%s: %s\n", owner, repoName, strings.Join(labels, ", "))
	}

	return nil
}
