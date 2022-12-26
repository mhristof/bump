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

type Client map[string]*Account

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

	cache.Write(accounts)
	return accounts
}

func (c *Client) Update(name string) string {
	return c.updateAMI(name)
}
