package cmd

import (
	"context"
	"fmt"

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

		cache, err := cmd.Flags().GetBool("cache")
		if err != nil {
			panic(err)
		}

		cloud := aws.New(context.TODO(), profiles, cache)
		fmt.Print(cloud.Update(args[0]), "\n")
	},
}

func init() {
	rootCmd.AddCommand(awsCmd)
}
