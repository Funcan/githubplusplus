package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

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
		name: "ci-on-pr",
		run: func(ctx context.Context, client *ghclient.Client, owner, repo string) error {
			files, err := client.ListDir(ctx, owner, repo, ".github/workflows")
			if err != nil {
				return err
			}
			for _, f := range files {
				name := f.GetName()
				if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
					continue
				}
				content, err := client.GetFileContent(ctx, owner, repo, f.GetPath())
				if err != nil {
					return err
				}
				if workflowTriggersOnPR(content) {
					return nil
				}
			}
			return fmt.Errorf("no workflow triggers on pull_request")
		},
	},
	{
		name: "protected-default-branch",
		run: func(ctx context.Context, client *ghclient.Client, owner, repo string) error {
			var checked bool
			for _, branch := range []string{"main", "master"} {
				exists, err := client.BranchExists(ctx, owner, repo, branch)
				if err != nil {
					return err
				}
				if !exists {
					continue
				}
				checked = true
				protection, err := client.GetBranchProtection(ctx, owner, repo, branch)
				if err != nil {
					return err
				}
				if protection == nil || protection.RequiredPullRequestReviews == nil {
					return fmt.Errorf("branch %s does not require pull request reviews", branch)
				}
			}
			if !checked {
				return fmt.Errorf("neither main nor master branch found")
			}
			return nil
		},
	},
	{
		name: "ci-required-to-merge",
		run: func(ctx context.Context, client *ghclient.Client, owner, repo string) error {
			var checked bool
			for _, branch := range []string{"main", "master"} {
				exists, err := client.BranchExists(ctx, owner, repo, branch)
				if err != nil {
					return err
				}
				if !exists {
					continue
				}
				checked = true
				protection, err := client.GetBranchProtection(ctx, owner, repo, branch)
				if err != nil {
					return err
				}
				rsc := protection.GetRequiredStatusChecks()
				if rsc == nil || (len(rsc.GetContexts()) == 0 && len(rsc.GetChecks()) == 0) {
					return fmt.Errorf("branch %s has no required status checks", branch)
				}
			}
			if !checked {
				return fmt.Errorf("neither main nor master branch found")
			}
			return nil
		},
	},
	{
		name: "dependabot-config",
		run: func(ctx context.Context, client *ghclient.Client, owner, repo string) error {
			for _, path := range []string{".github/dependabot.yml", ".github/dependabot.yaml"} {
				ok, err := client.FileExists(ctx, owner, repo, path)
				if err != nil {
					return err
				}
				if ok {
					return nil
				}
			}
			return fmt.Errorf("no .github/dependabot.yml found")
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

// workflowTriggersOnPR reports whether a GitHub Actions workflow file triggers
// on the pull_request event. The on: key may be a string, a sequence, or a mapping.
func workflowTriggersOnPR(content string) bool {
	var wf struct {
		On interface{} `yaml:"on"`
	}
	if err := yaml.Unmarshal([]byte(content), &wf); err != nil {
		return false
	}
	switch v := wf.On.(type) {
	case string:
		return v == "pull_request"
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == "pull_request" {
				return true
			}
		}
	case map[string]interface{}:
		_, ok := v["pull_request"]
		return ok
	}
	return false
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
