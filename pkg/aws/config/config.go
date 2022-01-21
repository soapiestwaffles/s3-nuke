package config

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// New creates a new aws.Config with custom endpoint resolver and region set
func New(region string, awsEndpoint string) (aws.Config, error) {
	var cfg aws.Config
	var err error
	if awsEndpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           awsEndpoint,
				SigningRegion: region,
			}, nil
		})
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithEndpointResolverWithOptions(customResolver))
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	}

	return cfg, err
}
