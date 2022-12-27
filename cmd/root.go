package cmd

import (
	"fmt"
	"os"

	"github.com/mhristof/go-git"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version = "devel"
	pwd     string
	dryrun  bool
	origin  string
)

var rootCmd = &cobra.Command{
	Use:     "bump",
	Short:   "Bump versions left and right",
	Long:    `TODO: changeme`,
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		Verbose(cmd)
		cwd, err := cmd.Flags().GetString("cwd")
		if err != nil {
			panic(err)
		}
		pwd = cwd

		dryrun, err = cmd.Flags().GetBool("dryrun")
		if err != nil {
			panic(err)
		}

		origin = git.Origin(pwd)
	},
}

// Verbose Increase verbosity.
func Verbose(cmd *cobra.Command) {
	verbose, err := cmd.Flags().GetCount("verbose")
	if err != nil {
		panic(err)
	}

	level := log.DebugLevel

	switch verbose {
	case 0:
		level = log.InfoLevel
	case 1:
		level = log.DebugLevel
	case 2:
		level = log.TraceLevel
	}

	log.SetLevel(level)

	// log.SetFormatter(&log.JSONFormatter{})
}

func init() {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().CountP("verbose", "v", "Increase verbosity")
	rootCmd.PersistentFlags().BoolP("dryrun", "n", false, "Dry run")
	rootCmd.PersistentFlags().StringP("cwd", "C", pwd, "Run from that directory")
	rootCmd.PersistentFlags().BoolP("cache", "c", true, "Enable cache")

	viper.SetConfigName("bump") // name of config file (without extension)
	viper.SetConfigType("yaml") // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("$XDG_CONFIG_HOME")
	err = viper.ReadInConfig() // Find and read the config file
	if err != nil {            // Handle errors reading the config file
		log.Info("generated config from current settings")
		viper.SafeWriteConfig()
	}
}

// Execute The main function for the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
