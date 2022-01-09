package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

type CloudwatchService interface {
}

// =====

// S3API defines the interface for AWS S3 SDK functions
type CloudwatchAPI interface {
	GetMetricData(ctx context.Context,
		params *cloudwatch.GetMetricDataInput,
		optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
}
