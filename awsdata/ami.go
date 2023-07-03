package awsdata

import (
	"context"
	"regexp"
	"sort"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	log "github.com/sirupsen/logrus"
)

type imageInput struct {
	name  string
	owner string
	arch  string
}

func (a *AWS) image(imageInput *imageInput) []types.Image {
	if a.amis[imageInput.name] != nil {
		return a.amis[imageInput.name]
	}

	filters := []types.Filter{
		{
			Name:   aws.String("name"),
			Values: []string{imageInput.name},
		},
	}

	if imageInput.owner != "" {
		filters = append(filters, types.Filter{
			Name:   aws.String("owner-id"),
			Values: []string{imageInput.owner},
		})
	}

	if imageInput.arch != "" {
		filters = append(filters, types.Filter{
			Name:   aws.String("architecture"),
			Values: []string{imageInput.arch},
		})
	}

	wg := sync.WaitGroup{}
	guard := make(chan struct{}, a.threads)
	wg.Add(len(a.ec2))

	for profile, client := range a.ec2 {
		guard <- struct{}{}

		go func(profile string, client *ec2.Client) {
			defer wg.Done()
			defer func() { <-guard }()

			image, err := client.DescribeImages(context.Background(), &ec2.DescribeImagesInput{
				Filters: filters,
			})
			if err != nil {
				log.WithFields(log.Fields{
					"profile": profile,
					"error":   err,
				}).Trace("error describing images")

				return
			}

			if len(image.Images) == 0 {
				log.WithFields(log.Fields{
					"profile": profile,
					"error":   err,
					"name":    imageInput.name,
				}).Trace("image not found")

				return
			}

			a.amis[imageInput.name] = append(a.amis[imageInput.name], image.Images...)
		}(profile, client)
	}

	return a.amis[imageInput.name]
}

func (a *AWS) ValidAMI(name string) string {
	image := a.image(&imageInput{name: name})

	if len(image) == 0 {
		log.WithFields(log.Fields{
			"name": name,
		}).Trace("image not found")

		return ""
	}

	imageName := *image[0].Name
	owner := *image[0].OwnerId
	architecture := string(image[0].Architecture)

	re := regexp.MustCompile(`\d+\.\d+\.\d+`)
	version := re.FindString(imageName)

	log.WithFields(log.Fields{
		"image":        name,
		"owner":        owner,
		"architecture": architecture,
		"version":      version,
	}).Debug("found image")

	re = regexp.MustCompile(version + `.*`)
	newImageName := re.ReplaceAllString(imageName, "*")
	newImages := a.image(&imageInput{
		name:  newImageName,
		owner: owner,
		arch:  architecture,
	})

	if len(newImages) == 0 {
		log.Trace("no new images found")

		return ""
	}

	sort.Slice(newImages, func(i, j int) bool {
		return *newImages[i].CreationDate > *newImages[j].CreationDate
	})

	newImage := *newImages[0].Name

	log.WithFields(log.Fields{
		"name":         name,
		"newImageName": newImageName,
		"newVersion":   newImage,
		"len":          len(newImages),
	}).Debug("found image")

	return newImage
}
