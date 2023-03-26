package awsdata

import (
	"context"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

func awsProfiles() []string {
	configPath, err := homedir.Expand("~/.aws/config")
	if err != nil {
		return []string{}
	}

	config, err := ini.Load(configPath)

	profiles := []string{}

	for _, section := range config.Sections() {
		if !strings.HasPrefix(section.Name(), "profile ") {
			continue
		}

		profiles = append(profiles, strings.ReplaceAll(section.Name(), "profile ", ""))
	}

	return profiles
}

func New() *AWS {
	ret := AWS{
		repos:    map[string][]*semver.Version{},
		services: map[string]*ecr.Client{},
	}

	for _, profile := range awsProfiles() {
		cfg, err := config.LoadDefaultConfig(
			context.Background(),
			config.WithSharedConfigProfile(profile),
		)
		if err != nil {
			log.WithFields(log.Fields{
				"profile": profile,
				"error":   err,
			}).Error("Failed to load AWS config")
		}

		ret.services[profile] = ecr.NewFromConfig(cfg)
	}

	return &ret
}

type AWS struct {
	services map[string]*ecr.Client
	repos    map[string][]*semver.Version
}

func (a *AWS) Tags(repositoryName string) []*semver.Version {
	if _, ok := a.repos[repositoryName]; ok {
		return a.repos[repositoryName]
	}

	for _, client := range a.services {
		a.repos[repositoryName] = append(a.repos[repositoryName], ecrRepo(client, repositoryName)...)
	}

	uniqueVersions := map[string]*semver.Version{}
	for _, version := range a.repos[repositoryName] {
		uniqueVersions[version.String()] = version
	}

	var uniqueVersionsSlice []*semver.Version
	for version, data := range uniqueVersions {
		log.WithFields(log.Fields{
			"repository": repositoryName,
			"version":    version,
		}).Trace("unique version")

		uniqueVersionsSlice = append(uniqueVersionsSlice, data)
	}

	a.repos[repositoryName] = uniqueVersionsSlice
	sort.Sort(semver.Collection(a.repos[repositoryName]))

	log.WithFields(log.Fields{
		"repository": repositoryName,
		"versions":   a.repos[repositoryName],
	}).Debug("retrieved tags")

	return a.repos[repositoryName]
}

func ecrRepo(client *ecr.Client, repositoryName string) []*semver.Version {
	// describe repositories
	repos, err := client.DescribeRepositories(context.Background(), &ecr.DescribeRepositoriesInput{})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to describe repositories")
	}

	for _, repo := range repos.Repositories {
		log.WithFields(log.Fields{
			"repo": *repo.RepositoryUri,
		}).Trace("Repository")

		if *repo.RepositoryUri != repositoryName {
			continue
		}

		log.WithFields(log.Fields{
			"repo": *repo.RepositoryUri,
		}).Debug("Repository")

		// describe images
		images, err := client.DescribeImages(context.Background(), &ecr.DescribeImagesInput{
			RepositoryName: repo.RepositoryName,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Failed to describe images")
		}

		var semverImages []*semver.Version

		for _, image := range images.ImageDetails {
			for _, tag := range image.ImageTags {
				log.WithFields(log.Fields{
					"image": tag,
				}).Trace("Image")
				version, err := semver.StrictNewVersion(tag)
				if err != nil {
					continue
				}

				semverImages = append(semverImages, version)
				log.WithFields(log.Fields{
					"image":  tag,
					"semver": version,
				}).Debug("found image tag")
			}
		}

		return semverImages
	}

	return []*semver.Version{}
}
