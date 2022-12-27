package aws

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrTypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

func (c *Account) findECRImages() error {
	c.ecrReposMutex.Lock()
	if len(c.ECRRepos) == 0 {
		c.ecrRepos()
	}
	c.ecrReposMutex.Unlock()

	var wg sync.WaitGroup
	wg.Add(len(c.ECRRepos))

	for _, repo := range c.ECRRepos {
		go func(name *string) {
			defer wg.Done()
			paginator := ecr.NewDescribeImagesPaginator(c.ecr, &ecr.DescribeImagesInput{
				RepositoryName: name,
			})

			for page := 0; paginator.HasMorePages(); page++ {

				log.WithFields(log.Fields{
					"page": page,
					"name": *name,
				}).Trace("retrieving ecr.DescribeImages page")

				results, err := paginator.NextPage(c.ctx)
				if err != nil {
					log.WithFields(log.Fields{
						"err":  err,
						"page": page,
						"name": *name,
					}).Error("cannot get next page")

					return
				}

				log.WithFields(log.Fields{
					"len":     len(results.ImageDetails),
					"repo":    *name,
					"profile": c.profile,
				}).Trace("found ECR images from repo")

				c.ecrImagesMutex.Lock()
				c.ECRImages = append(c.ECRImages, results.ImageDetails...)
				c.ecrImagesMutex.Unlock()
			}
		}(repo.RepositoryName)
	}

	wg.Wait()

	log.WithFields(log.Fields{
		"len":     len(c.ECRImages),
		"profile": c.profile,
	}).Info("found ECR images for profile")

	return nil
}

func (c *Account) ecrRepos() error {
	paginator := ecr.NewDescribeRepositoriesPaginator(c.ecr, &ecr.DescribeRepositoriesInput{})

	for page := 0; paginator.HasMorePages(); page++ {
		log.WithFields(log.Fields{
			"page": page,
		}).Trace("retrieving ecr repos page")

		results, err := paginator.NextPage(c.ctx)
		if err != nil {
			return errors.Wrapf(err, "cannot retrieve page: %d", page)
		}

		c.ECRRepos = append(c.ECRRepos, results.Repositories...)
	}

	log.WithFields(log.Fields{
		"len": len(c.ECRRepos),
	}).Trace("found ECR repos")

	return nil
}

func (c *Client) updateECR(name string) string {
	fields := strings.Split(name, ":")
	repo := fields[0]

	var version string
	if len(fields) > 1 {
		version = fields[1]
	}

	var matches []ecrTypes.ImageDetail

	for account, data := range c.accounts {
		for _, ecrImage := range data.ECRImages {
			if *ecrImage.RepositoryName == repo {
				log.WithFields(log.Fields{
					"ecrImage": ecrImage,
					"account":  account,
					"version":  version,
					"repo":     repo,
				}).Trace("found ecr image to update")

				matches = append(matches, ecrImage)
			}
		}
	}
	newVersion := nextECRVersion(matches, version)

	if newVersion != "" {
		repo = fmt.Sprintf("%s:%s", repo, newVersion)
	}

	return repo
}

func nextECRVersion(images []ecrTypes.ImageDetail, current string) string {
	var tags []string
	for _, image := range images {
		tags = append(tags, image.ImageTags...)
	}

	if len(tags) == 0 {
		return current
	}

	sort.Slice(tags, func(i, j int) bool {
		return semver.Compare(tags[i], tags[j]) < 0
	})

	log.WithFields(log.Fields{
		"tags": tags,
	}).Trace("sorted tags from images")

	return tags[len(tags)-1]
}
