package config

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
)

// New creates a new aws.Config with custom endpoint resolver and region set
func New(region string) (aws.Config, error) {
	retrier := func() aws.Retryer {
		return retry.NewAdaptiveMode()
	}

	return config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithRetryer(retrier))
}
