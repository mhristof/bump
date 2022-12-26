package aws

import (
	"encoding/json"
	"regexp"
	"sort"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

func (c *Account) amis() error {
	paginator := ec2.NewDescribeImagesPaginator(c.ec2, &ec2.DescribeImagesInput{
		Owners: []string{"self"},
	})

	for page := 0; paginator.HasMorePages(); page++ {
		log.WithFields(log.Fields{
			"page": page,
		}).Trace("retrieving AMI page")

		results, err := paginator.NextPage(c.ctx)
		if err != nil {
			return errors.Wrapf(err, "cannot retrieve page: %d", page)
		}

		c.Images = append(c.Images, results.Images...)
	}

	log.WithFields(log.Fields{
		"len":     len(c.Images),
		"profile": c.profile,
	}).Info("found amis")

	return nil
}

func trimImageName(s string) string {
	trims := []*regexp.Regexp{
		regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}[A-Z]*`),
		regexp.MustCompile(`v\d*\.\d*.\d*`),
		regexp.MustCompile(`--$`),
	}

	for _, trim := range trims {
		s = trim.ReplaceAllString(s, "")
	}

	return s
}

func (c *Client) updateAMI(name string) string {
	var images []string

	var thisImage *types.Image

	for account, resources := range *c {
		for _, image := range resources.Images {
			if *image.Name == name {
				thisImage = &image

				thisImageJSON, err := json.MarshalIndent(thisImage, "", "")
				if err != nil {
					panic(err)
				}

				log.WithFields(log.Fields{
					"account": account,
					"image":   string(thisImageJSON),
					"name":    name,
				}).Debug("found image with name")
				break
			}
		}
	}

	cleanName := trimImageName(name)

	for account, resources := range *c {
		for _, image := range resources.Images {
			if amiCompare(thisImage, &image, cleanName) {
				log.WithFields(log.Fields{
					"image.Name": *image.Name,
					"cleanName":  cleanName,
					"name":       name,
					"account":    account,
				}).Debug("found matching candidate")

				images = append(images, *image.Name)
			}
		}
	}
	return nextAMIVersion(images, cleanName)
}

func amiCompare(this *types.Image, that *types.Image, trimmedName string) bool {
	if this != nil && len(this.Tags) > 0 {
		matched := 0

		for _, tag := range this.Tags {
			for _, thatTag := range that.Tags {
				if *tag.Key == *thatTag.Key {
					matched++
				}
			}
		}

		matchedPercent := (100 * matched) / len(this.Tags)

		log.WithFields(log.Fields{
			"this.Name": *this.Name,
			"that.Name": *that.Name,
			"matched":   matched,
			"matched-%": matchedPercent,
		}).Trace("comparing tags")

		if matchedPercent < 80 {
			return false
		}

	}

	if trimImageName(*that.Name) == trimmedName {
		return true
	}

	return false
}

func nextAMIVersion(images []string, current string) string {
	if len(images) == 0 {
		return current
	}

	sort.Slice(images, func(i, j int) bool {
		return semver.Compare(images[i], images[j]) > 0
	})

	log.WithFields(log.Fields{
		"images": images,
	}).Trace("sorted images")

	return images[0]
}
