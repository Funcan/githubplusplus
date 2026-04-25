package gh

import (
	"context"
	"fmt"

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
	var all []*gogithub.Repository
	opts := &gogithub.RepositoryListByUserOptions{
		ListOptions: gogithub.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := c.gh.Repositories.ListByUser(ctx, user, opts)
		if err != nil {
			return nil, fmt.Errorf("listing repos for user %q: %w", user, err)
		}
		all = append(all, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// ListOrgRepos returns all repos for the given org.
func (c *Client) ListOrgRepos(ctx context.Context, org string) ([]*gogithub.Repository, error) {
	var all []*gogithub.Repository
	opts := &gogithub.RepositoryListByOrgOptions{
		ListOptions: gogithub.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := c.gh.Repositories.ListByOrg(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("listing repos for org %q: %w", org, err)
		}
		all = append(all, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}
