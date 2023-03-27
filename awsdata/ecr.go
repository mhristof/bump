package awsdata

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrTypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
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
	if err != nil {
		return []string{}
	}

	profiles := []string{}

	accountRoles := map[string]string{}

	for _, section := range config.Sections() {
		if !strings.HasPrefix(section.Name(), "profile ") {
			continue
		}

		accountID := section.Key("sso_account_id").String()
		region := section.Key("region").String()
		role := section.Key("sso_role_name").String()

		key := accountID + "/" + region

		existingRole, ok := accountRoles[key]
		if ok && strings.Contains(existingRole, "ReadOnly") {
			log.WithFields(log.Fields{
				"key":       key,
				"accountID": accountID,
				"role":      role,
			}).Debug("skipping role, already have a readonly one")

			continue
		}

		log.WithFields(log.Fields{
			"accountID":    accountID,
			"region":       region,
			"replaced":     ok,
			"existingRole": existingRole,
			"role":         role,
		}).Debug("updating/adding role")

		accountRoles[key] = role

		profiles = append(profiles, strings.ReplaceAll(section.Name(), "profile ", ""))
	}

	return profiles
}

func New(threads int) *AWS {
	ret := AWS{
		repos:    map[string][]*semver.Version{},
		services: map[string]*ecr.Client{},
		threads:  threads,
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
	reposMux sync.Mutex
	threads  int
}

func (a *AWS) Tags(repositoryName string) []*semver.Version {
	if _, ok := a.repos[repositoryName]; ok {
		return a.repos[repositoryName]
	}

	wg := sync.WaitGroup{}
	wg.Add(len(a.services))
	guard := make(chan struct{}, a.threads)

	for _, client := range a.services {
		guard <- struct{}{}

		go func(client *ecr.Client) {
			defer wg.Done()
			regionRepos := ecrRepo(client, repositoryName)

			a.reposMux.Lock()
			defer a.reposMux.Unlock()

			a.repos[repositoryName] = append(a.repos[repositoryName], regionRepos...)
			<-guard
		}(client)
	}

	wg.Wait()

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
	sort.Sort(sort.Reverse(semver.Collection(a.repos[repositoryName])))

	log.WithFields(log.Fields{
		"repository": repositoryName,
		"versions":   a.repos[repositoryName],
	}).Debug("retrieved tags")

	return a.repos[repositoryName]
}

func ecrRepo(client *ecr.Client, repositoryName string) []*semver.Version {
	paginator := ecr.NewDescribeRepositoriesPaginator(client, &ecr.DescribeRepositoriesInput{})

	repos := []ecrTypes.Repository{}
	for page := 0; paginator.HasMorePages(); page++ {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"page":  page,
			}).Error("Failed to describe repositories")
		}

		log.WithFields(log.Fields{
			"page": page,
		}).Trace("retrieved repos page")

		repos = append(repos, page.Repositories...)
	}

	for _, repo := range repos {
		log.WithFields(log.Fields{
			"repo": *repo.RepositoryUri,
		}).Trace("Repository")

		if *repo.RepositoryUri != repositoryName {
			continue
		}

		log.WithFields(log.Fields{
			"repo": *repo.RepositoryUri,
		}).Debug("Repository")

		// describe image tags with pages
		paginator := ecr.NewDescribeImagesPaginator(client, &ecr.DescribeImagesInput{
			RepositoryName: repo.RepositoryName,
		})

		images := []ecrTypes.ImageDetail{}
		for page := 0; paginator.HasMorePages(); page++ {
			page, err := paginator.NextPage(context.Background())
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"page":  page,
				}).Error("Failed to describe images")
			}

			images = append(images, page.ImageDetails...)
		}

		var semverImages []*semver.Version

		for _, image := range images {
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
