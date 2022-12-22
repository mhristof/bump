package aws

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrTypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/mhristof/bump/cache"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Account struct {
	images         []types.Image
	ECRRepos       []ecrTypes.Repository
	ecrReposMutex  sync.Mutex
	ecrImages      []ecrTypes.ImageDetail
	ecrImagesMutex sync.Mutex
	ec2            *ec2.Client
	ecr            *ecr.Client
	ctx            context.Context
	profile        string
}

type Client map[string]*Account

// Ec2 Create an ec2 aws client.
func ec2Client(ctx context.Context, profile string) *ec2.Client {
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("cannot create aws config")
	}

	return ec2.NewFromConfig(cfg)
}

func ecrClient(ctx context.Context, profile string) *ecr.Client {
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("cannot create aws config")
	}

	return ecr.NewFromConfig(cfg)
}

func (c *Account) ecrRepos() error {
	paginator := ecr.NewDescribeRepositoriesPaginator(c.ecr, &ecr.DescribeRepositoriesInput{})

	for page := 0; paginator.HasMorePages(); page++ {
		log.WithFields(log.Fields{
			"page": page,
		}).Trace("retrieving ecr repos page")

		results, err := paginator.NextPage(c.ctx)
		if err != nil {
			return errors.Wrapf(err, "cannot retrieve page: %f", page)
		}

		c.ECRRepos = append(c.ECRRepos, results.Repositories...)
	}

	log.WithFields(log.Fields{
		"len": len(c.ECRRepos),
	}).Trace("found ECR repos")

	return nil
}

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
				c.ecrImages = append(c.ecrImages, results.ImageDetails...)
				c.ecrImagesMutex.Unlock()
			}
		}(repo.RepositoryName)
	}

	wg.Wait()

	log.WithFields(log.Fields{
		"len":     len(c.ecrImages),
		"profile": c.profile,
	}).Info("found ECR images for profile")

	return nil
}

func (c *Account) amis() error {
	paginator := ec2.NewDescribeImagesPaginator(c.ec2, &ec2.DescribeImagesInput{
		ExecutableUsers: []string{"self"},
	})

	for page := 0; paginator.HasMorePages(); page++ {
		log.WithFields(log.Fields{
			"page": page,
		}).Trace("retrieving AMI page")

		results, err := paginator.NextPage(c.ctx)
		if err != nil {
			return errors.Wrapf(err, "cannot retrieve page: %f", page)
		}

		c.images = append(c.images, results.Images...)
	}

	log.WithFields(log.Fields{
		"len":     len(c.images),
		"profile": c.profile,
	}).Info("found amis")

	return nil
}

// New Create new aws client and query data for all given profiles
func New(ctx context.Context, profiles []string, enableCache bool) Client {
	accounts := Client{}

	for _, profile := range profiles {
		accounts[profile] = &Account{
			ec2:     ec2Client(ctx, profile),
			ecr:     ecrClient(ctx, profile),
			ctx:     ctx,
			profile: profile,
		}
	}

	if enableCache {
		cached := cache.Load()
		err := json.Unmarshal(cached, &accounts)
		if err == nil {
			return accounts
		}
	}

	var wg sync.WaitGroup
	for profile, c := range accounts {
		data := map[string]func() error{
			"amis":      c.amis,
			"ecrImages": c.findECRImages,
		}

		wg.Add(len(data))

		for name, dataFunction := range data {
			log.WithFields(log.Fields{
				"name":         name,
				"profile":      profile,
				"dataFunction": dataFunction,
			}).Debug("finding resources")

			go func(name string, function func() error) {
				defer wg.Done()

				err := function()
				if err != nil {
					log.WithFields(log.Fields{
						"err":     err,
						"name":    name,
						"profile": profile,
					}).Error("cannot retrieve data")

					return
				}

				log.WithFields(log.Fields{
					"name":    name,
					"profile": profile,
				}).Trace("finished retrieving data")
			}(name, dataFunction)
		}
	}

	wg.Wait()

	cache.Write(accounts)
	return accounts
}
