package changes

import (
	"bytes"
	"encoding/json"
	"os"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/mhristof/bump/terraform"
	log "github.com/sirupsen/logrus"
	"github.com/tmccombs/hcl2json/convert"
)

type Config struct {
	Modules []Module `hcl:"module,block"`
}

type Module struct {
	Name    string `hcl:"name,label"`
	Source  string `hcl:"source,optional"`
	Version string `hcl:"version,optional"`
}

type Foo struct {
	Terraform []struct {
		RequiredProviders []struct {
			Name    string `hcl:"name,label"`
			Version string `hcl:"version,optional"`
		} `hcl:"required_providers,block"`

		RequiredVersion string `hcl:"required_version"`
	} `hcl:"terraform,block"`
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

		versions, source := terraform.RegistryVersions(module.Source)

		sort.Sort(sort.Reverse(semver.Collection(versions)))

		moduleVersion, err := semver.NewVersion(module.Version)
		if err != nil {
			//log.WithFields(log.Fields{
			//"module": module.Name,
			//}).Warning("failed to parse version")
			continue
		}

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
					Source:     source,
				})

				break
			}
		}
	}

	// var versions Foo

	var options convert.Options
	converted, err := convert.Bytes(data, "foo", options)
	if err != nil {
		log.Fatalf("Failed to convert: %s", err)
	}

	// prettyConverted, _ := json.MarshalIndent(converted, "", "  ")
	// fmt.Println(string(prettyConverted))

	var indented bytes.Buffer
	if err := json.Indent(&indented, converted, "", "    "); err != nil {
		log.Fatalf("Failed to indent: %s", err)
	}

	return ret
}
