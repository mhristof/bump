package changes

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Masterminds/semver/v3"
	log "github.com/sirupsen/logrus"
)

func dockerHub(image string) string {
	fields := strings.Split(image, ":")

	name := fields[0]
	tag := fields[1]

	tagVersion, err := semver.NewVersion(tag)
	if err != nil {
		log.WithFields(log.Fields{
			"image": image,
			"name":  name,
			"tag":   tag,
			"error": err,
		}).Error("cannot parse tag")

		return ""
	}

	log.WithFields(log.Fields{
		"image": image,
		"name":  name,
		"tag":   tag,
	}).Debug("dockerHub")

	resp, err := http.Get("https://hub.docker.com/v2/repositories/" + name + "/tags/" + tag)
	if err != nil {
		log.WithFields(log.Fields{
			"name":  name,
			"tag":   tag,
			"error": err,
		}).Error("cannot get specific tag")

		return ""
	}

	resp.Body.Close()

	resp, err = http.Get("https://hub.docker.com/v2/repositories/" + name + "/tags")
	if err != nil {
		log.WithFields(log.Fields{
			"name":  name,
			"error": err,
		}).Error("cannot get tags")

		return ""
	}

	var tags DockerHubTagsResponse
	err = json.NewDecoder(resp.Body).Decode(&tags)
	if err != nil {
		log.WithFields(log.Fields{
			"name":  name,
			"error": err,
		}).Error("cannot decode tags")

		return ""
	}

	for _, result := range tags.Results {
		version, err := semver.NewVersion(result.Name)
		if err != nil {
			log.WithFields(log.Fields{
				"name":  name,
				"error": err,
			}).Debug("cannot parse version")

			continue
		}

		if version.Compare(tagVersion) > 0 {
			return name + ":" + result.Name
		}
	}

	return image
}

type DockerHubTagsResponse struct {
	Count    int64       `json:"count"`
	Next     string      `json:"next"`
	Previous interface{} `json:"previous"`
	Results  []struct {
		ContentType string `json:"content_type"`
		Creator     int64  `json:"creator"`
		Digest      string `json:"digest"`
		FullSize    int64  `json:"full_size"`
		ID          int64  `json:"id"`
		Images      []struct {
			Architecture string      `json:"architecture"`
			Digest       string      `json:"digest"`
			Features     string      `json:"features"`
			LastPulled   string      `json:"last_pulled"`
			LastPushed   string      `json:"last_pushed"`
			Os           string      `json:"os"`
			OsFeatures   string      `json:"os_features"`
			OsVersion    interface{} `json:"os_version"`
			Size         int64       `json:"size"`
			Status       string      `json:"status"`
			Variant      interface{} `json:"variant"`
		} `json:"images"`
		LastUpdated         string `json:"last_updated"`
		LastUpdater         int64  `json:"last_updater"`
		LastUpdaterUsername string `json:"last_updater_username"`
		MediaType           string `json:"media_type"`
		Name                string `json:"name"`
		Repository          int64  `json:"repository"`
		TagLastPulled       string `json:"tag_last_pulled"`
		TagLastPushed       string `json:"tag_last_pushed"`
		TagStatus           string `json:"tag_status"`
		V2                  bool   `json:"v2"`
	} `json:"results"`
}
