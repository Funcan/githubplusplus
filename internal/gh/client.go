package gh

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/auth"
	gogithub "github.com/google/go-github/v72/github"
	"golang.org/x/oauth2"
)

// Client wraps the go-github client with helpers.
type Client struct {
	gh *gogithub.Client
}

// New creates a Client using the token from the gh CLI.
func New() (*Client, error) {
	token, _ := auth.TokenForHost("github.com")
	if token == "" {
		return nil, fmt.Errorf("not authenticated: run `gh auth login` first")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)
	return &Client{gh: gogithub.NewClient(httpClient)}, nil
}

// AuthenticatedUser returns the login of the currently authenticated user.
func (c *Client) AuthenticatedUser(ctx context.Context) (string, error) {
	user, _, err := c.gh.Users.Get(ctx, "")
	if err != nil {
		return "", err
	}
	return user.GetLogin(), nil
}

// ListUserRepos returns all repos for the given user login.
func (c *Client) ListUserRepos(ctx context.Context, user string) ([]*gogithub.Repository, error) {
	repos, err := paginate(func(page int) ([]*gogithub.Repository, *gogithub.Response, error) {
		return c.gh.Repositories.ListByUser(ctx, user, &gogithub.RepositoryListByUserOptions{
			ListOptions: gogithub.ListOptions{PerPage: 100, Page: page},
		})
	})
	if err != nil {
		return nil, fmt.Errorf("listing repos for user %q: %w", user, err)
	}
	return repos, nil
}

// GetRepo returns a single repository by owner and name.
func (c *Client) GetRepo(ctx context.Context, owner, repo string) (*gogithub.Repository, error) {
	r, _, err := c.gh.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("getting repo %s/%s: %w", owner, repo, err)
	}
	return r, nil
}

// MergeUpstreamResult holds the outcome of a fork-sync operation.
type MergeUpstreamResult struct {
	// MergeType is "merge", "fast-forward", or "none" (already up to date).
	MergeType string
	Message   string
}

// MergeUpstream syncs the named branch of a fork with its upstream via the
// GitHub API. The call is synchronous: a successful return means the fork has
// been updated.
func (c *Client) MergeUpstream(ctx context.Context, owner, repo, branch string) (*MergeUpstreamResult, error) {
	result, _, err := c.gh.Repositories.MergeUpstream(ctx, owner, repo, &gogithub.RepoMergeUpstreamRequest{
		Branch: &branch,
	})
	if err != nil {
		return nil, fmt.Errorf("merging upstream for %s/%s@%s: %w", owner, repo, branch, err)
	}
	return &MergeUpstreamResult{
		MergeType: result.GetMergeType(),
		Message:   result.GetMessage(),
	}, nil
}

// ForkStatus holds the result of comparing a fork's branch against its upstream.
type ForkStatus struct {
	// Status is one of "identical", "ahead", "behind", or "diverged".
	Status   string
	AheadBy  int
	BehindBy int
}

// CompareWithUpstream compares the fork's default branch against the upstream's
// default branch and returns the relationship between them.
func (c *Client) CompareWithUpstream(ctx context.Context, forkOwner, forkRepo, forkBranch, upstreamOwner, upstreamBranch string) (*ForkStatus, error) {
	// base is the upstream ref; head is the fork branch. The comparison is run
	// in the context of the fork repo so GitHub can resolve both sides.
	base := upstreamOwner + ":" + upstreamBranch
	cmp, _, err := c.gh.Repositories.CompareCommits(ctx, forkOwner, forkRepo, base, forkBranch, nil)
	if err != nil {
		return nil, fmt.Errorf("comparing %s/%s with upstream: %w", forkOwner, forkRepo, err)
	}
	return &ForkStatus{
		Status:   cmp.GetStatus(),
		AheadBy:  cmp.GetAheadBy(),
		BehindBy: cmp.GetBehindBy(),
	}, nil
}

// GetPR returns a single pull request by number. Unlike the list endpoint,
// the individual PR endpoint populates fields such as mergeable_state.
func (c *Client) GetPR(ctx context.Context, owner, repo string, number int) (*gogithub.PullRequest, error) {
	pr, _, err := c.gh.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("getting PR %s/%s#%d: %w", owner, repo, number, err)
	}
	return pr, nil
}

// ListOpenPRs returns all open pull requests for the given repo.
func (c *Client) ListOpenPRs(ctx context.Context, owner, repo string) ([]*gogithub.PullRequest, error) {
	prs, err := paginate(func(page int) ([]*gogithub.PullRequest, *gogithub.Response, error) {
		return c.gh.PullRequests.List(ctx, owner, repo, &gogithub.PullRequestListOptions{
			State:       "open",
			ListOptions: gogithub.ListOptions{PerPage: 100, Page: page},
		})
	})
	if err != nil {
		return nil, fmt.Errorf("listing PRs for %s/%s: %w", owner, repo, err)
	}
	return prs, nil
}

// ListOpenPRsToUpstream returns all open pull requests whose head is the named
// fork against the upstream repository.
func (c *Client) ListOpenPRsToUpstream(ctx context.Context, upstreamOwner, upstreamRepo, forkOwner, forkRepo string) ([]*gogithub.PullRequest, error) {
	forkFullName := forkOwner + "/" + forkRepo
	prs, err := paginate(func(page int) ([]*gogithub.PullRequest, *gogithub.Response, error) {
		all, resp, err := c.gh.PullRequests.List(ctx, upstreamOwner, upstreamRepo, &gogithub.PullRequestListOptions{
			State:       "open",
			ListOptions: gogithub.ListOptions{PerPage: 100, Page: page},
		})
		if err != nil {
			return nil, resp, err
		}
		var filtered []*gogithub.PullRequest
		for _, pr := range all {
			if pr.GetHead().GetRepo().GetFullName() == forkFullName {
				filtered = append(filtered, pr)
			}
		}
		return filtered, resp, nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing PRs for %s/%s: %w", upstreamOwner, upstreamRepo, err)
	}
	return prs, nil
}

// CountOpenPRsToUpstream returns the number of open pull requests from forkOwner
// against the upstream repository.
func (c *Client) CountOpenPRsToUpstream(ctx context.Context, upstreamOwner, upstreamRepo, forkOwner, forkRepo string) (int, error) {
	prs, err := c.ListOpenPRsToUpstream(ctx, upstreamOwner, upstreamRepo, forkOwner, forkRepo)
	if err != nil {
		return 0, err
	}
	return len(prs), nil
}

// TransferRepo moves owner/repo to newOwner (an org or user login).
// If GitHub queues the transfer asynchronously (202 Accepted), it polls until
// the repository appears under newOwner or the context deadline is exceeded.
func (c *Client) TransferRepo(ctx context.Context, owner, repo, newOwner string) error {
	_, _, err := c.gh.Repositories.Transfer(ctx, owner, repo, gogithub.TransferRequest{
		NewOwner: newOwner,
	})
	if err != nil {
		var acceptedErr *gogithub.AcceptedError
		if !errors.As(err, &acceptedErr) {
			return fmt.Errorf("transferring %s/%s to %s: %w", owner, repo, newOwner, err)
		}
		// 202 Accepted: transfer is queued — poll until it lands.
		if pollErr := c.pollTransfer(ctx, newOwner, repo); pollErr != nil {
			return fmt.Errorf("waiting for transfer of %s/%s to %s: %w", owner, repo, newOwner, pollErr)
		}
	}
	return nil
}

// pollTransfer waits until newOwner/repo is accessible, indicating the
// transfer completed. It retries every 3 s up to the context deadline.
func (c *Client) pollTransfer(ctx context.Context, newOwner, repo string) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_, _, err := c.gh.Repositories.Get(ctx, newOwner, repo)
			if err == nil {
				return nil
			}
		}
	}
}

// ListOpenIssues returns all open issues (excluding pull requests) for the given repo.
func (c *Client) ListOpenIssues(ctx context.Context, owner, repo string) ([]*gogithub.Issue, error) {
	issues, err := paginate(func(page int) ([]*gogithub.Issue, *gogithub.Response, error) {
		all, resp, err := c.gh.Issues.ListByRepo(ctx, owner, repo, &gogithub.IssueListByRepoOptions{
			State:       "open",
			ListOptions: gogithub.ListOptions{PerPage: 100, Page: page},
		})
		if err != nil {
			return nil, resp, err
		}
		var filtered []*gogithub.Issue
		for _, issue := range all {
			if issue.PullRequestLinks == nil {
				filtered = append(filtered, issue)
			}
		}
		return filtered, resp, nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing issues for %s/%s: %w", owner, repo, err)
	}
	return issues, nil
}

// ListDir returns the contents of a directory in the repository's default branch.
// Returns nil (not an error) if the path does not exist.
func (c *Client) ListDir(ctx context.Context, owner, repo, path string) ([]*gogithub.RepositoryContent, error) {
	_, dir, resp, err := c.gh.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("listing %s in %s/%s: %w", path, owner, repo, err)
	}
	return dir, nil
}

// GetFileContent returns the decoded text content of a file in the repository's default branch.
func (c *Client) GetFileContent(ctx context.Context, owner, repo, path string) (string, error) {
	file, _, _, err := c.gh.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		return "", fmt.Errorf("getting %s in %s/%s: %w", path, owner, repo, err)
	}
	content, err := file.GetContent()
	if err != nil {
		return "", fmt.Errorf("decoding %s in %s/%s: %w", path, owner, repo, err)
	}
	return content, nil
}

// FileExists reports whether the given path exists in the repository's default branch.
func (c *Client) FileExists(ctx context.Context, owner, repo, path string) (bool, error) {
	_, _, resp, err := c.gh.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false, nil
		}
		return false, fmt.Errorf("checking %s in %s/%s: %w", path, owner, repo, err)
	}
	return true, nil
}

// ListOrgRepos returns all repos for the given org.
func (c *Client) ListOrgRepos(ctx context.Context, org string) ([]*gogithub.Repository, error) {
	repos, err := paginate(func(page int) ([]*gogithub.Repository, *gogithub.Response, error) {
		return c.gh.Repositories.ListByOrg(ctx, org, &gogithub.RepositoryListByOrgOptions{
			ListOptions: gogithub.ListOptions{PerPage: 100, Page: page},
		})
	})
	if err != nil {
		return nil, fmt.Errorf("listing repos for org %q: %w", org, err)
	}
	return repos, nil
}
