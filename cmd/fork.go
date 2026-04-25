package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/v72/github"
	"github.com/spf13/cobra"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
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

type resolvedRepo struct {
	owner string
	repo  string
}

// expandRepoArg resolves an argument to one or more (owner, repo) pairs.
// It handles:
//   - "." or a local path    → single repo resolved from the git remote
//   - "owner/repo"           → single repo
//   - "github.com/owner/repo" → single repo
//   - "owner"               → all repos for that user or org
//   - "github.com/owner"    → all repos for that user or org
func expandRepoArg(ctx context.Context, client *ghclient.Client, arg string) ([]resolvedRepo, error) {
	if info, statErr := os.Stat(arg); statErr == nil && info.IsDir() {
		_, owner, repoName, err := resolveRepoArg(arg)
		if err != nil {
			return nil, err
		}
		return []resolvedRepo{{owner, repoName}}, nil
	}

	ref := strings.TrimPrefix(arg, "github.com/")
	parts := strings.SplitN(ref, "/", 3)

	switch len(parts) {
	case 1:
		if parts[0] != "" {
			return expandHandle(ctx, client, parts[0])
		}
	case 2:
		if parts[0] != "" && parts[1] != "" {
			return []resolvedRepo{{parts[0], parts[1]}}, nil
		}
	}

	return nil, fmt.Errorf("%q is not an existing directory, an owner/repo reference, or an owner reference", arg)
}

// expandHandle lists all repos for the given user or org handle.
// It tries the handle as an org first; if not found, falls back to a user.
func expandHandle(ctx context.Context, client *ghclient.Client, handle string) ([]resolvedRepo, error) {
	repos, err := client.ListOrgRepos(ctx, handle)
	if err != nil {
		if !isNotFound(err) {
			return nil, fmt.Errorf("listing repos for %q: %w", handle, err)
		}
		repos, err = client.ListUserRepos(ctx, handle)
		if err != nil {
			return nil, fmt.Errorf("listing repos for %q: %w", handle, err)
		}
	}
	result := make([]resolvedRepo, 0, len(repos))
	for _, r := range repos {
		result = append(result, resolvedRepo{r.GetOwner().GetLogin(), r.GetName()})
	}
	return result, nil
}

// forEachExpandedRepo resolves each arg (expanding user/org handles to all their
// repos) and calls fn(owner, repo) for every resolved repository. Errors are
// printed to stderr; if any occur the summary failMsg is returned.
func forEachExpandedRepo(ctx context.Context, client *ghclient.Client, args []string, failMsg string, fn func(owner, repo string) error) error {
	var anyErr bool
	for _, arg := range args {
		repos, err := expandRepoArg(ctx, client, arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", arg, err)
			anyErr = true
			continue
		}
		for _, r := range repos {
			if err := fn(r.owner, r.repo); err != nil {
				fmt.Fprintf(os.Stderr, "error: %s/%s: %v\n", r.owner, r.repo, err)
				anyErr = true
			}
		}
	}
	if anyErr {
		return fmt.Errorf("%s", failMsg)
	}
	return nil
}
