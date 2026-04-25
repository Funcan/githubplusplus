package cmd

import (
	"context"
	"errors"
	"net/http"

	gogithub "github.com/google/go-github/v72/github"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
)

// collectRepos fetches repos from all configured targets (user + orgs).
// If neither --user nor --org is specified, it defaults to the authenticated user.
func collectRepos(ctx context.Context, client *ghclient.Client) ([]*gogithub.Repository, error) {
	users := []string{}
	if flagUser != "" {
		users = append(users, flagUser)
	}

	// If neither flag is set, default to the authenticated user.
	if flagUser == "" && len(flagOrgs) == 0 {
		login, err := client.AuthenticatedUser(ctx)
		if err != nil {
			return nil, err
		}
		users = append(users, login)
	}

	var all []*gogithub.Repository

	for _, u := range users {
		repos, err := client.ListUserRepos(ctx, u)
		if err != nil {
			return nil, err
		}
		all = append(all, repos...)
	}

	for _, org := range flagOrgs {
		repos, err := client.ListOrgRepos(ctx, org)
		if err != nil {
			if isNotFound(err) {
				// Not an org — try as a user.
				repos, err = client.ListUserRepos(ctx, org)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
		all = append(all, repos...)
	}

	return all, nil
}

func isNotFound(err error) bool {
	var errResp *gogithub.ErrorResponse
	if errors.As(err, &errResp) {
		return errResp.Response.StatusCode == http.StatusNotFound
	}
	return false
}
