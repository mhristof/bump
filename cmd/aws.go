package cmd

import (
	"context"

	"github.com/mhristof/bump/aws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var awsCmd = &cobra.Command{
	Use:   "aws",
	Short: "Update aws stuff",
	Run: func(cmd *cobra.Command, args []string) {
		profiles, err := cmd.Flags().GetStringSlice("profiles")
		if err != nil {
			panic(err)
		}

		if profiles[0] == "all" {
			profiles = aws.ProfilesFromConfig()
		}

		log.WithFields(log.Fields{
			"profiles": profiles,
		}).Trace("scanning AWS")

		_ = aws.New(context.TODO(), profiles)
	},
}

func init() {
	rootCmd.AddCommand(awsCmd)
}
