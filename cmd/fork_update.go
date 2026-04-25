package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

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

	var anyErr bool
	for _, arg := range args {
		if err := updateFork(ctx, client, arg); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", arg, err)
			anyErr = true
		}
	}
	if anyErr {
		return fmt.Errorf("one or more repos could not be updated")
	}
	return nil
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

	if !repo.GetFork() {
		return fmt.Errorf("%s/%s is not a fork", owner, repoName)
	}

	if repo.GetParent() == nil {
		return fmt.Errorf("%s/%s: upstream parent metadata unavailable", owner, repoName)
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

// resolveRepoArg converts a CLI argument into a (localPath, owner, repo) triple.
// The argument may be a local directory path or an "owner/repo" GitHub reference.
func resolveRepoArg(arg string) (localPath, owner, repoName string, err error) {
	info, statErr := os.Stat(arg)
	if statErr == nil && info.IsDir() {
		remoteURL, err := git.GetRemoteURL(arg, "origin")
		if err != nil {
			return "", "", "", fmt.Errorf("not a git repo or missing origin remote: %w", err)
		}
		owner, repoName, err = parseGitHubURL(remoteURL)
		if err != nil {
			return "", "", "", fmt.Errorf("origin %q is not a github.com remote: %w", remoteURL, err)
		}
		return arg, owner, repoName, nil
	}

	parts := strings.SplitN(arg, "/", 2)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" && !strings.Contains(parts[1], "/") {
		return "", parts[0], parts[1], nil
	}

	return "", "", "", fmt.Errorf("%q is not an existing directory or an owner/repo reference", arg)
}

// parseGitHubURL extracts the owner and repo name from an HTTPS or SSH GitHub remote URL.
func parseGitHubURL(rawURL string) (owner, repo string, err error) {
	if strings.HasPrefix(rawURL, "git@github.com:") {
		path := strings.TrimPrefix(rawURL, "git@github.com:")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], nil
		}
		return "", "", fmt.Errorf("malformed SSH URL %q", rawURL)
	}

	u, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return "", "", fmt.Errorf("cannot parse URL: %w", parseErr)
	}
	if !strings.EqualFold(u.Host, "github.com") {
		return "", "", fmt.Errorf("host is %q, not github.com", u.Host)
	}
	path := strings.Trim(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("cannot extract owner/repo from path %q", u.Path)
}
