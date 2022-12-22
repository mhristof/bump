package aws

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func ProfilesFromConfig() []string {
	config, err := homedir.Expand("~/.aws/config")
	if err != nil {
		panic(err)
	}

	viper.SetConfigFile(config)
	viper.SetConfigType("ini")

	err = viper.ReadInConfig() // Find and read the config file
	if err != nil {            // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	profiles := map[string]string{}

	// viper returns the config as lowercase and we need the actual name
	caseInsensitiveProfiles := actualProfiles(config)

	for profile := range viper.AllSettings() {
		if !strings.HasPrefix(profile, "profile ") {
			continue
		}

		name := strings.Fields(profile)[1]
		for _, prof := range caseInsensitiveProfiles {
			if strings.ToLower(prof) == name {
				name = prof
			}
		}

		config := viper.GetStringMapString(profile)
		account := config["sso_account_id"]
		role := config["sso_role_name"]

		log.WithFields(log.Fields{
			"account": account,
			"role":    role,
			"name":    name,
		}).Trace("found role for account in config")

		_, ok := profiles[account]
		if !ok {
			log.WithFields(log.Fields{
				"name":    name,
				"account": account,
			}).Trace("first hit for aws account")

			profiles[account] = name
		}

		if strings.Contains(strings.ToLower(config["sso_role_name"]), "readonly") {
			log.WithFields(log.Fields{
				"name":    name,
				"account": account,
			}).Trace("overwritting profile due to read-onlyness")

			profiles[account] = name
		}
	}

	ret := make([]string, len(profiles))
	i := 0

	log.WithFields(log.Fields{
		"profiles": profiles,
	}).Debug("found config profiles with roles")

	for _, profile := range profiles {
		ret[i] = profile
		i++
	}

	return ret
}

func actualProfiles(config string) []string {
	data, err := ioutil.ReadFile(config)
	if err != nil {
		panic(err)
	}

	var ret []string

	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "[profile ") {
			continue
		}

		ret = append(ret, strings.Trim(strings.Fields(line)[1], "]"))
	}

	return ret
}
