package cloudwatch

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/config"
)

type Service interface {
}

// ServiceOption is used with NewS3Service and configures the newly created s3Service
type ServiceOption func(s *service)

type service struct {
	client      CloudwatchAPI
	awsEndpoint string
	region      string
}

// NewS3Service returns an initialized S3Service
func NewService(opts ...ServiceOption) Service {
	svc := &service{}
	for _, opt := range opts {
		opt(svc)
	}

	if svc.client == nil {
		if svc.region == "" {
			svc.client = newClient(os.Getenv("AWS_REGION"), svc.awsEndpoint)
		} else {
			svc.client = newClient(svc.region, svc.awsEndpoint)
		}
	}

	return svc
}

// WithAPI should be used if you want to initialize your own S3 client (such as in cases of a mock S3 client for testing)
// This cannot be used with WithAWSEndpoint
func WithAPI(client CloudwatchAPI) ServiceOption {
	return func(s *service) {
		s.client = client
	}
}

// WithAWSEndpoint sets endpoint to be used by the AWS client
// This cannot be used with WithS3API
func WithAWSEndpoint(awsEndpoint string) ServiceOption {
	return func(s *service) {
		s.awsEndpoint = awsEndpoint
	}
}

// WithRegion sets the AWS client region
func WithRegion(region string) ServiceOption {
	return func(s *service) {
		s.region = region
	}
}

func newClient(region string, awsEndpoint string) *cloudwatch.Client {
	// Initialize AWS S3 Client
	cfg, err := config.New(region, awsEndpoint)
	if err != nil {
		return nil
	}

	return cloudwatch.NewFromConfig(cfg)
}

// =====

// S3API defines the interface for AWS S3 SDK functions
type CloudwatchAPI interface {
	GetMetricData(ctx context.Context,
		params *cloudwatch.GetMetricDataInput,
		optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
}
