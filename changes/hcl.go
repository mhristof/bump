package changes

import (
	"os"
	"sort"

	"github.com/Masterminds/semver/v3"
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
		for i := 0; i < len(versions); i++ {
			if versions[i].GreaterThan(moduleVersion) {
				log.WithFields(log.Fields{
					"module":  module.Name,
					"version": versions[i],
				}).Debug("found latest change")

				ret = append(ret, &Change{
					line:       string(data),
					Module:     module.Name,
					file:       path,
					format:     Terraform,
					version:    moduleVersion,
					newVersion: versions[i],
				})

				break
			}
		}
	}

	return ret
}
