package terraform

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Masterminds/semver/v3"
	log "github.com/sirupsen/logrus"
)

func RegistryVersions(module string) []*semver.Version {
	url := "https://registry.terraform.io/v1/modules/" + module + "/versions"
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var versions VersionsResponse
	err = json.Unmarshal(body, &versions)
	if err != nil {
		panic(err)
	}

	var ret []*semver.Version

	for _, v := range versions.Modules[0].Versions {

		version, err := semver.NewVersion(v.Version)
		if err != nil {
			panic(err)
		}

		log.WithFields(log.Fields{
			"module":  module,
			"version": v.Version,
			"semver":  version,
		}).Trace("Version")

		ret = append(ret, version)
	}

	log.WithFields(log.Fields{
		"module": module,
		"len":    len(ret),
	}).Debug("Versions")

	return ret
}

type VersionsResponse struct {
	Modules []struct {
		Source   string `json:"source"`
		Versions []struct {
			Root struct {
				Dependencies []interface{} `json:"dependencies"`
				Providers    []struct {
					Name      string `json:"name"`
					Namespace string `json:"namespace"`
					Source    string `json:"source"`
					Version   string `json:"version"`
				} `json:"providers"`
			} `json:"root"`
			Submodules []struct {
				Dependencies []interface{} `json:"dependencies"`
				Path         string        `json:"path"`
				Providers    []struct {
					Name      string `json:"name"`
					Namespace string `json:"namespace"`
					Source    string `json:"source"`
					Version   string `json:"version"`
				} `json:"providers"`
			} `json:"submodules"`
			Version string `json:"version"`
		} `json:"versions"`
	} `json:"modules"`
}
