package changes

import (
	"os"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/mhristof/bump/terraform"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Modules []Module `hcl:"module,block"`
}

type Module struct {
	Name    string `hcl:"name,label"`
	Source  string `hcl:"source,optional"`
	Version string `hcl:"version,optional"`
}

func parseHCL(path string) Changes {
	log.WithField("file", path).Debug("Parsing HCL")

	var config Config

	// load file
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read configuration: %s", err)
	}

	_ = hclsimple.Decode("foo.hcl", data, nil, &config)

	var ret Changes

	for _, module := range config.Modules {
		log.WithField("module", module).Debug("Module")

		versions := terraform.RegistryVersions(module.Source)

		sort.Sort(sort.Reverse(semver.Collection(versions)))

		moduleVersion := semver.MustParse(module.Version)
		for i := len(versions) - 1; i >= 0; i-- {
			if versions[i].GreaterThan(moduleVersion) {
				log.WithFields(log.Fields{
					"module":  module.Name,
					"version": versions[i],
				}).Debug("adding change")

				ret = append(ret, &Change{
					line:       string(data),
					NewLine:    strings.ReplaceAll(string(data), module.Version, versions[i].String()),
					module:     module.Name,
					file:       path,
					format:     Terraform,
					version:    moduleVersion,
					newVersion: versions[i],
				})
			}
		}
	}

	return ret
}
