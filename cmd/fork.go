package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/v72/github"
	"github.com/spf13/cobra"

	"github.com/Funcan/githubplusplus/internal/git"
)

var forkCmd = &cobra.Command{
	Use:   "fork",
	Short: "Commands for working with forked repositories",
}

func init() {
	rootCmd.AddCommand(forkCmd)
}

// resolveRepoArg converts a CLI argument into a (localPath, owner, repo) triple.
// The argument may be a local directory path, an "owner/repo" GitHub reference,
// or a "github.com/owner/repo" reference.
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

	ref := strings.TrimPrefix(arg, "github.com/")
	parts := strings.SplitN(ref, "/", 2)
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

// validateFork checks that repo is a fork with available parent metadata and
// returns the parent repository. Returns an error if either check fails.
func validateFork(repo *github.Repository, owner, repoName string) (*github.Repository, error) {
	if !repo.GetFork() {
		return nil, fmt.Errorf("%s/%s is not a fork", owner, repoName)
	}
	parent := repo.GetParent()
	if parent == nil {
		return nil, fmt.Errorf("%s/%s: upstream parent metadata unavailable", owner, repoName)
	}
	return parent, nil
}

// forEachRepoArg calls fn for each arg, printing errors to stderr as they
// occur. If any call fails it returns an error using failMsg as the summary.
func forEachRepoArg(args []string, failMsg string, fn func(string) error) error {
	var anyErr bool
	for _, arg := range args {
		if err := fn(arg); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", arg, err)
			anyErr = true
		}
	}
	if anyErr {
		return fmt.Errorf("%s", failMsg)
	}
	return nil
}
