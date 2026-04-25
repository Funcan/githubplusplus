package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
	"github.com/Funcan/githubplusplus/internal/git"
)

var cloneFlags filterFlags
var cloneDest string
var cloneSkipExisting bool
var cloneConcurrency int

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone matching repos into a local directory",
	RunE:  runClone,
}

func init() {
	addFilterFlags(cloneCmd, &cloneFlags)
	cloneCmd.Flags().StringVar(&cloneDest, "dest", ".", "Directory to clone repos into")
	cloneCmd.Flags().BoolVar(&cloneSkipExisting, "skip-existing", false, "Skip repos whose directory already exists")
	cloneCmd.Flags().IntVar(&cloneConcurrency, "concurrency", 8, "Number of parallel clone workers")
	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	client, err := ghclient.New()
	if err != nil {
		return err
	}

	repos, err := collectRepos(ctx, client)
	if err != nil {
		return err
	}
	repos = applyFilters(repos, &cloneFlags)

	type job struct {
		name string
		url  string
		dest string
	}

	var jobs []job
	for _, r := range repos {
		dest := filepath.Join(cloneDest, r.GetName())
		if cloneSkipExisting {
			if _, err := os.Stat(dest); err == nil {
				fmt.Printf("skipping %s (already exists)\n", r.GetFullName())
				continue
			}
		}
		jobs = append(jobs, job{name: r.GetFullName(), url: r.GetCloneURL(), dest: dest})
	}

	errs := runConcurrent(jobs, cloneConcurrency, func(j job) error {
		fmt.Printf("cloning %s ...\n", j.name)
		if err := git.Clone(j.url, j.dest); err != nil {
			return fmt.Errorf("%s: %w", j.name, err)
		}
		return nil
	})

	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "error: %v\n", e)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d clone(s) failed", len(errs))
	}
	return nil
}
