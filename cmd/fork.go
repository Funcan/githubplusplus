package cmd

import "github.com/spf13/cobra"

var forkCmd = &cobra.Command{
	Use:   "fork",
	Short: "Commands for working with forked repositories",
}

func init() {
	rootCmd.AddCommand(forkCmd)
}
