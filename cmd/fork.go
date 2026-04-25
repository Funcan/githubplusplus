package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"

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
