package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

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
