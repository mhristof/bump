package aws

import (
	"regexp"
	"sort"
	"strings"

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
		regexp.MustCompile(`-\d{10}$`),              // epoch time
		regexp.MustCompile(`-\d{4}-\d{2}-\d{2}$`),   // 2022-01-01
		regexp.MustCompile(`-\d{4}\.\d{2}\.\d{2}$`), // 2022.01.01
		regexp.MustCompile(`-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}[A-Z]*`),
		regexp.MustCompile(`-v\d*\.\d*.\d*`), // semver
		regexp.MustCompile(`--$`),
	}

	for _, trim := range trims {
		s = trim.ReplaceAllString(s, "")
	}

	return s
}

func (c *Client) findAMI(name string) (types.Image, []types.Image) {
	var exact types.Image
	var partial []types.Image

	for _, resources := range c.accounts {
		for _, image := range resources.Images {
			if *image.Name == name {
				exact = image

				continue
			}

			if strings.HasPrefix(trimImageName(name), trimImageName(*image.Name)) {
				partial = append(partial, image)
			}
		}
	}

	return exact, partial
}

func pointerString(in *string) string {
	if in == nil {
		return "nil"
	}
	return *in
}

func (c *Client) updateAMI(name string) string {
	images := map[string]types.Image{}
	partialMatchedVersions := map[string]types.Image{}

	thisImage, partialImages := c.findAMI(name)
	cleanName := trimImageName(name)

	log.WithFields(log.Fields{
		"name":         pointerString(thisImage.Name),
		"len(partial)": len(partialImages),
		"cleanName":    cleanName,
		"thisImage":    thisImage,
	}).Debug("found Image")

	for account, resources := range c.accounts {
		for _, image := range resources.Images {
			if amiCompare(thisImage, &image, cleanName) {
				log.WithFields(log.Fields{
					"image.Name":     *image.Name,
					"cleanName":      cleanName,
					"name":           name,
					"account":        account,
					"thisImage.Name": pointerString(thisImage.Name),
				}).Debug("found matching candidate")

				images[*image.Name] = image
			}

			for _, pImage := range partialImages {
				partialMatchedVersions[*pImage.Name] = pImage
			}
		}
	}

	log.WithFields(log.Fields{
		"images":                      len(images),
		"len(partialMatchedVersions)": len(partialMatchedVersions),
	}).Debug("found image candidates")

	nextVersion := ""

	if thisImage.Name != nil {
		nextVersion = nextAMIVersion(c, images, thisImage)
		log.WithFields(log.Fields{
			"nextVersion": nextVersion,
		}).Debug("from exact match")
	}

	if nextVersion == "" && len(partialMatchedVersions) > 0 {
		nextVersion = nextAMIVersion(c, partialMatchedVersions, partialMatchedVersions[mapKeys(partialMatchedVersions)[0]])
		log.WithFields(log.Fields{
			"nextVersion": nextVersion,
		}).Debug("from partial match")
	}

	return nextVersion
}

func (c *Client) amiVersion(image types.Image, key string) (string, string) {
	for _, tag := range image.Tags {
		if key != "" && *tag.Key == key {
			return key, *tag.Value
		}

		for _, vTag := range c.AMITags {
			if vTag == *tag.Key {
				return *tag.Key, *tag.Value
			}
		}
	}

	return "CreationDate", *image.CreationDate
}

func mapKeys[C any](in map[string]C) []string {
	ret := make([]string, len(in))

	i := 0
	for k := range in {
		ret[i] = k
		i++
	}

	return ret
}

func amiCompare(this types.Image, that *types.Image, trimmedName string) bool {
	if this.Name != nil && len(this.Tags) > 0 {
		matched := 0

		for _, tag := range this.Tags {
			for _, thatTag := range that.Tags {
				if *tag.Key == *thatTag.Key {
					matched++
				}
			}
		}

		matchedPercent := (100 * matched) / len(this.Tags)

		if matchedPercent < 80 {
			return false
		}

	}

	if trimImageName(*that.Name) == trimmedName {
		return true
	}

	return false
}

func nextAMIVersion(c *Client, images map[string]types.Image, current types.Image) string {
	if len(images) == 0 {
		return *current.Name
	}

	versionKey, versionValue := c.amiVersion(current, "")
	isSemver := semver.IsValid(versionValue)

	log.WithFields(log.Fields{
		"name":     *current.Name,
		"key":      versionKey,
		"value":    versionValue,
		"isSemver": isSemver,
	}).Debug("found version from this ami")

	versions := make(map[string]string, len(images))
	for k, v := range images {
		_, version := c.amiVersion(v, versionKey)
		if isSemver && !semver.IsValid(version) {
			continue
		}
		versions[k] = version
	}

	versionKeys := mapKeys(versions)
	sort.SliceStable(versionKeys, func(i, j int) bool {
		if isSemver {
			return semver.Compare(versions[versionKeys[i]], versions[versionKeys[j]]) > 0
		}

		return versions[versionKeys[i]] > versions[versionKeys[j]]
	})

	return versionKeys[0]
}
