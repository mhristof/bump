package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mhristof/bump/aws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var awsCmd = &cobra.Command{
	Use:   "aws",
	Short: "Update aws stuff",
	Run: func(cmd *cobra.Command, args []string) {
		profiles := viper.GetStringSlice("profiles")
		profile, ok := viper.GetStringMapString("repository-profiles")[origin]
		if ok {
			log.WithFields(log.Fields{
				"profile": profile,
				"origin":  origin,
			}).Debug("found origin in config")
			profiles = []string{profile}
		} else {
			if profiles[0] == "all" {
				profiles = aws.ProfilesFromConfig()
			}
		}

		log.WithFields(log.Fields{
			"profiles": profiles,
		}).Trace("scanning AWS")

		cache, err := cmd.Flags().GetBool("cache")
		if err != nil {
			panic(err)
		}

		cloud := aws.New(&aws.NewInput{
			Ctx:      context.TODO(),
			Profiles: profiles,
			Cache:    cache,
			AMITags:  viper.GetStringSlice("ami-version-tags"),
		})
		fmt.Print(cloud.Update(args[0]), "\n")
	},
}

func init() {
	rootCmd.AddCommand(awsCmd)
	awsCmd.PersistentFlags().StringSliceP("profiles", "p", []string{os.Getenv("AWS_PROFILE")}, "AWS profiles to scan. Specify 'all' for all available profiles")
	awsCmd.PersistentFlags().StringToStringP("repository-profiles", "R", map[string]string{}, "AWS Profile for specific repository in the form of git remote, example https://github.com/foo=aws-profile-test")
	rootCmd.PersistentFlags().StringSliceP("ami-version-tags", "", []string{}, "AMI tags that hold version information. First tag found will be used.")

	viper.BindPFlag("profiles", awsCmd.PersistentFlags().Lookup("profiles"))
	viper.BindPFlag("repository-profiles", awsCmd.PersistentFlags().Lookup("repository-profiles"))
	viper.BindPFlag("ami-version-tags", awsCmd.PersistentFlags().Lookup("ami-version-tags"))
}
