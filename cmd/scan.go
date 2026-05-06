package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
)

// check is a named function that inspects a single repository.
// It returns nil if the check passes, or an error describing the failure.
type check struct {
	name string
	run  func(ctx context.Context, client *ghclient.Client, owner, repo string) error
}

var checks = []check{
	{
		name: "dummy",
		run: func(ctx context.Context, client *ghclient.Client, owner, repo string) error {
			return nil
		},
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan [repo...]",
	Short: "Run checks against one or more repositories",
	Long: `Run checks against one or more repositories.

Each argument may be a local path to a git checkout or a GitHub "owner/repo"
reference. If no arguments are given the repository in the current directory
is used.`,
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"."}
	}
	ctx := context.Background()
	client, err := ghclient.New()
	if err != nil {
		return err
	}

	return forEachExpandedRepo(ctx, client, args, "one or more repos could not be scanned", func(owner, repo string) error {
		for _, c := range checks {
			err := c.run(ctx, client, owner, repo)
			if err != nil {
				fmt.Printf("%s/%s: %s FAIL: %v\n", owner, repo, c.name, err)
			} else {
				fmt.Printf("%s/%s: %s PASS\n", owner, repo, c.name)
			}
		}
		return nil
	})
}
