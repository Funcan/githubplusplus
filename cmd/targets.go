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

	// Track which usernames have already been fetched to avoid duplicate API
	// calls when the same handle appears in both --user and --org.
	fetched := make(map[string]bool)
	var all []*gogithub.Repository

	for _, u := range users {
		if fetched[u] {
			continue
		}
		fetched[u] = true
		repos, err := client.ListUserRepos(ctx, u)
		if err != nil {
			return nil, err
		}
		all = append(all, repos...)
	}

	for _, org := range flagOrgs {
		if fetched[org] {
			continue
		}
		fetched[org] = true
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

	return deduplicateRepos(all), nil
}

// deduplicateRepos returns repos with duplicates removed, preserving the first
// occurrence of each repository identified by its full name (owner/repo).
func deduplicateRepos(repos []*gogithub.Repository) []*gogithub.Repository {
	seen := make(map[string]struct{}, len(repos))
	out := make([]*gogithub.Repository, 0, len(repos))
	for _, r := range repos {
		key := r.GetFullName()
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, r)
		}
	}
	return out
}

func isNotFound(err error) bool {
	var errResp *gogithub.ErrorResponse
	if errors.As(err, &errResp) {
		return errResp.Response.StatusCode == http.StatusNotFound
	}
	return false
}
