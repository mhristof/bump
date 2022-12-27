package aws

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrTypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/mhristof/bump/cache"
	log "github.com/sirupsen/logrus"
)

type Account struct {
	Images         []types.Image
	ECRRepos       []ecrTypes.Repository
	ecrReposMutex  sync.Mutex
	ECRImages      []ecrTypes.ImageDetail
	ecrImagesMutex sync.Mutex
	ec2            *ec2.Client
	ecr            *ecr.Client
	ctx            context.Context
	profile        string
}

type Client struct {
	accounts map[string]*Account
	AMITags  []string
}

type NewInput struct {
	Ctx      context.Context
	Profiles []string
	Cache    bool
	AMITags  []string
}

// New Create new aws client and query data for all given profiles
func New(input *NewInput) Client {
	client := Client{
		AMITags: input.AMITags,
	}

	log.WithFields(log.Fields{
		"input": input,
	}).Debug("creating aws with input")

	for _, profile := range input.Profiles {
		client.accounts[profile] = &Account{
			ec2:     ec2Client(input.Ctx, profile),
			ecr:     ecrClient(input.Ctx, profile),
			ctx:     input.Ctx,
			profile: profile,
		}
	}

	if input.Cache {
		cached := cache.Load()
		err := json.Unmarshal(cached, &client.accounts)
		if err == nil {
			filtered := map[string]*Account{}

			for _, profile := range input.Profiles {
				filtered[profile] = client.accounts[profile]
			}

			client.accounts = filtered

			return client
		}
	}

	var wg sync.WaitGroup
	for profile, c := range client.accounts {
		data := map[string]func() error{
			"amis":      c.amis,
			"ECRImages": c.findECRImages,
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

	cache.Write(client.accounts)
	return client
}

func (c *Client) Update(name string) string {
	return c.updateAMI(name)
}
