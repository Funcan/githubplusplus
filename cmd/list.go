package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	gogithub "github.com/google/go-github/v72/github"
	"github.com/spf13/cobra"

	ghclient "github.com/Funcan/githubplusplus/internal/gh"
)

var listFlags filterFlags
var listFormat string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List repos for a user or org",
	RunE:  runList,
}

func init() {
	addFilterFlags(listCmd, &listFlags)
	listCmd.Flags().StringVar(&listFormat, "format", "table", "Output format: table or json")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	client, err := ghclient.New()
	if err != nil {
		return err
	}

	repos, err := collectRepos(ctx, client)
	if err != nil {
		return err
	}
	repos = applyFilters(repos, &listFlags)

	if listFormat == "json" {
		return printJSON(repos)
	}
	return printTable(repos)
}

func printTable(repos []*gogithub.Repository) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tLANGUAGE\tFORK\tARCHIVED\tDESCRIPTION")
	for _, r := range repos {
		fmt.Fprintf(w, "%s\t%s\t%v\t%v\t%s\n",
			r.GetFullName(),
			r.GetLanguage(),
			r.GetFork(),
			r.GetArchived(),
			r.GetDescription(),
		)
	}
	return w.Flush()
}

func printJSON(repos []*gogithub.Repository) error {
	// Use a simple loop to emit one JSON object per line (JSON Lines).
	for _, r := range repos {
		fmt.Printf(`{"name":%q,"language":%q,"fork":%v,"archived":%v,"description":%q,"clone_url":%q}`,
			r.GetFullName(),
			r.GetLanguage(),
			r.GetFork(),
			r.GetArchived(),
			r.GetDescription(),
			r.GetCloneURL(),
		)
		fmt.Println()
	}
	return nil
}
