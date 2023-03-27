package terraform

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Masterminds/semver/v3"
	log "github.com/sirupsen/logrus"
)

func RegistryVersions(module string) ([]*semver.Version, string) {
	url := "https://registry.terraform.io/v1/modules/" + module
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var mod TerraformRegistryModuleResponse
	err = json.Unmarshal(body, &mod)
	if err != nil {
		panic(err)
	}

	log.WithFields(log.Fields{
		"source": mod.Source,
	}).Debug("found module")

	var ret []*semver.Version

	for _, version := range mod.Versions {

		semVersion, err := semver.NewVersion(version)
		if err != nil {
			panic(err)
		}

		log.WithFields(log.Fields{
			"version": version,
			"semver":  semVersion,
		}).Trace("Version")

		ret = append(ret, semVersion)
	}

	log.WithFields(log.Fields{
		"module": module,
		"len":    len(ret),
	}).Debug("Versions")

	return ret, mod.Source
}

type TerraformRegistryModuleResponse struct {
	Description string `json:"description"`
	Downloads   int64  `json:"downloads"`
	Examples    []struct {
		Dependencies []struct {
			Name    string `json:"name"`
			Source  string `json:"source"`
			Version string `json:"version"`
		} `json:"dependencies"`
		Empty  bool `json:"empty"`
		Inputs []struct {
			Default     string `json:"default"`
			Description string `json:"description"`
			Name        string `json:"name"`
			Required    bool   `json:"required"`
			Type        string `json:"type"`
		} `json:"inputs"`
		Name                 string        `json:"name"`
		Outputs              []interface{} `json:"outputs"`
		Path                 string        `json:"path"`
		ProviderDependencies []struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			Source    string `json:"source"`
			Version   string `json:"version"`
		} `json:"provider_dependencies"`
		Readme    string        `json:"readme"`
		Resources []interface{} `json:"resources"`
	} `json:"examples"`
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Namespace       string   `json:"namespace"`
	Owner           string   `json:"owner"`
	Provider        string   `json:"provider"`
	ProviderLogoURL string   `json:"provider_logo_url"`
	Providers       []string `json:"providers"`
	PublishedAt     string   `json:"published_at"`
	Root            struct {
		Dependencies []interface{} `json:"dependencies"`
		Empty        bool          `json:"empty"`
		Inputs       []struct {
			Default     string `json:"default"`
			Description string `json:"description"`
			Name        string `json:"name"`
			Required    bool   `json:"required"`
			Type        string `json:"type"`
		} `json:"inputs"`
		Name    string `json:"name"`
		Outputs []struct {
			Description string `json:"description"`
			Name        string `json:"name"`
		} `json:"outputs"`
		Path                 string `json:"path"`
		ProviderDependencies []struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			Source    string `json:"source"`
			Version   string `json:"version"`
		} `json:"provider_dependencies"`
		Readme    string `json:"readme"`
		Resources []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"resources"`
	} `json:"root"`
	Source     string `json:"source"`
	Submodules []struct {
		Dependencies []interface{} `json:"dependencies"`
		Empty        bool          `json:"empty"`
		Inputs       []struct {
			Default     string `json:"default"`
			Description string `json:"description"`
			Name        string `json:"name"`
			Required    bool   `json:"required"`
			Type        string `json:"type"`
		} `json:"inputs"`
		Name    string `json:"name"`
		Outputs []struct {
			Description string `json:"description"`
			Name        string `json:"name"`
		} `json:"outputs"`
		Path                 string `json:"path"`
		ProviderDependencies []struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			Source    string `json:"source"`
			Version   string `json:"version"`
		} `json:"provider_dependencies"`
		Readme    string `json:"readme"`
		Resources []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"resources"`
	} `json:"submodules"`
	Tag      string   `json:"tag"`
	Verified bool     `json:"verified"`
	Version  string   `json:"version"`
	Versions []string `json:"versions"`
}
