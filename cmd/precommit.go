package cmd

import (
	"github.com/mhristof/bump/precommit"
	"github.com/spf13/cobra"
)

var precommitCmd = &cobra.Command{
	Use:   "precommit",
	Short: "Update precommit",
	Run: func(cmd *cobra.Command, args []string) {
		precommit.Update(pwd, dryrun)
	},
}

func init() {
	rootCmd.AddCommand(precommitCmd)
}
