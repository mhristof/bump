package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	log "github.com/sirupsen/logrus"
)

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
