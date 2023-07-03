package changes

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	prettyConverted, _ := json.MarshalIndent(converted, "", "  ")
	fmt.Println(string(prettyConverted))

	var indented bytes.Buffer
	if err := json.Indent(&indented, converted, "", "    "); err != nil {
		log.Fatalf("Failed to indent: %s", err)
	}

	fmt.Println(string(indented.Bytes()))

	//_ = hclsimple.Decode("foo.hcl", data, nil, &versions)
	//prettyVersions, _ := json.MarshalIndent(versions, "", "  ")
	//fmt.Println(string(prettyVersions))

	return ret
}

// file, diags := hclwrite.ParseConfig(data, "versions.tf", hcl.Pos{Line: 1, Column: 1})
// if diags.HasErrors() {
// 	fmt.Printf("errors: %s", diags)
// }

// file.WriteTo(os.Stdout)

// prettyFile, _ := json.MarshalIndent(file, "", "  ")
// fmt.Println("file:", string(prettyFile))

// prettyBody, _ := json.MarshalIndent(file.Body(), "", "  ")
// fmt.Println("body:", string(prettyBody))

// prettyBodyBlocks, _ := json.MarshalIndent(file.Body().Blocks(), "", "  ")
// fmt.Println("body blocks:", string(prettyBodyBlocks))

// for _, block := range file.Body().Blocks() {
// 	prettyBlock, _ := json.MarshalIndent(block, "", "  ")
// 	fmt.Println("block:", string(prettyBlock))
// 	fmt.Print("block type:", block.Type())
// 	fmt.Print("block labels:", block.Labels())
// }

// prettyAttrs, _ := json.MarshalIndent(file.Body().Attributes(), "", "  ")
// fmt.Println("attrs:", string(prettyAttrs))

// for k, attr := range file.Body().Attributes() {
// 	fmt.Println("foo", k)
// 	prettyAttr, _ := json.MarshalIndent(attr, "", "  ")
// 	fmt.Println("attr:", string(prettyAttr))
// }

// return ret
// }
