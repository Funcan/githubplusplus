package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	flagUser string
	flagOrgs []string
)

var rootCmd = &cobra.Command{
	Use:   "gh++",
	Short: "A tool for managing GitHub repos at scale",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagUser, "user", "", "Target a personal account (defaults to authenticated user if no --org specified)")
	rootCmd.PersistentFlags().StringArrayVar(&flagOrgs, "org", nil, "Target an org (can be specified multiple times)")
}
