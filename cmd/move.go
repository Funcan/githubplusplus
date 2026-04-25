package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
)

var moveToOrg string
var moveToUser string
var moveConcurrency int

var moveCmd = &cobra.Command{
	Use:   "move <owner/repo> [<owner/repo>...]",
	Short: "Move repos to a different org or user",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runMove,
}

// sanitizeOwner strips URL scheme and github.com host from a value that should
// be a bare org or user login (e.g. "https://github.com/acme" → "acme").
func sanitizeOwner(s string) string {
	// If it looks like a URL, parse out just the first path segment.
	if strings.Contains(s, "/") || strings.Contains(s, ":") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			// e.g. https://github.com/acme or github.com/acme parsed with scheme
			parts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
			if parts[0] != "" {
				fmt.Fprintf(os.Stderr, "warning: treating %q as owner login %q\n", s, parts[0])
				return parts[0]
			}
		}
		// Plain "github.com/acme" without scheme — url.Parse gives no Host.
		trimmed := strings.TrimPrefix(s, "github.com/")
		if trimmed != s {
			parts := strings.SplitN(trimmed, "/", 2)
			fmt.Fprintf(os.Stderr, "warning: treating %q as owner login %q\n", s, parts[0])
			return parts[0]
		}
	}
	return s
}

func init() {
	moveCmd.Flags().StringVar(&moveToOrg, "to-org", "", "Destination org")
	moveCmd.Flags().StringVar(&moveToUser, "to-user", "", "Destination user")
	moveCmd.Flags().IntVar(&moveConcurrency, "concurrency", 8, "Number of parallel transfer workers")
	rootCmd.AddCommand(moveCmd)
}

func runMove(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if moveToOrg != "" && moveToUser != "" {
		return fmt.Errorf("only one of --to-org or --to-user may be specified")
	}
	dest := moveToOrg
	if dest == "" {
		dest = moveToUser
	}
	if dest == "" {
		return fmt.Errorf("one of --to-org or --to-user must be specified")
	}
	dest = sanitizeOwner(dest)

	type job struct {
		owner, repo string
	}

	seen := make(map[string]bool)
	var jobs []job
	for _, arg := range args {
		if seen[arg] {
			continue
		}
		seen[arg] = true
		parts := strings.SplitN(arg, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid repo %q: must be in owner/repo format", arg)
		}
		jobs = append(jobs, job{owner: parts[0], repo: parts[1]})
	}

	client, err := ghclient.New()
	if err != nil {
		return err
	}

	ch := make(chan job, len(jobs))
	for _, j := range jobs {
		ch <- j
	}
	close(ch)

	workers := moveConcurrency
	if workers > len(jobs) {
		workers = len(jobs)
	}
	if workers < 1 {
		workers = 1
	}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range ch {
				fmt.Printf("moving %s/%s to %s ...\n", j.owner, j.repo, dest)
				if err := client.TransferRepo(ctx, j.owner, j.repo, dest); err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("%s/%s: %w", j.owner, j.repo, err))
					mu.Unlock()
				} else {
					fmt.Printf("moved %s/%s -> %s/%s\n", j.owner, j.repo, dest, j.repo)
				}
			}
		}()
	}
	wg.Wait()

	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "error: %v\n", e)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d move(s) failed", len(errs))
	}
	return nil
}
