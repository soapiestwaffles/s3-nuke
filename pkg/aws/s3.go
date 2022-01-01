package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Service defines functions related to S3 operations
type S3Service interface {

	// ListBuckets loads buckets into memory
	GetAllBuckets(ctx context.Context) ([]Bucket, error)
}

// Bucket contains information about an S3 bucket
type Bucket struct {
	CreationDate *time.Time
	Name         *string
}

// S3API defines the interface for AWS S3 SDK functions
type S3API interface {
	ListBuckets(ctx context.Context,
		params *s3.ListBucketsInput,
		optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
}

type S3ServiceOption func(s *s3Service)

type s3Service struct {
	client      S3API
	awsEndpoint string
}

// NewS3Service returns an initialized S3Service
func NewS3Service(opts ...S3ServiceOption) S3Service {
	svc := &s3Service{}
	for _, opt := range opts {
		opt(svc)
	}

	if svc.client == nil {
		svc.client = newS3Client(svc.awsEndpoint)
	}

	return svc
}

// WithS3API should be used if you want to initalize your own S3 client (such as in cases of a mock S3 client for testing)
// This cannot be used with WithAWSEndpoint
func WithS3API(s3Client S3API) S3ServiceOption {
	return func(s *s3Service) {
		s.client = s3Client
	}
}

// WithAWSEndpoint sets endpoint to be used by the AWS client
// This cannot be used with WithS3API
func WithAWSEndpoint(awsEndpoint string) S3ServiceOption {
	return func(s *s3Service) {
		s.awsEndpoint = awsEndpoint
	}
}

func (s *s3Service) GetAllBuckets(ctx context.Context) ([]Bucket, error) {
	result, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	buckets := make([]Bucket, 0)
	for _, b := range result.Buckets {
		buckets = append(buckets,
			Bucket{
				CreationDate: b.CreationDate,
				Name:         b.Name,
			},
		)
	}

	return buckets, nil
}

func newS3Client(awsEndpoint string) *s3.Client {
	// Initialize AWS S3 Client
	cfg, err := newConfig(awsEndpoint)
	if err != nil {
		return nil
	}

	return s3.NewFromConfig(cfg)
}

func newConfig(awsEndpoint string) (aws.Config, error) {
	var cfg aws.Config
	var err error
	if awsEndpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID: "aws",
				URL:         awsEndpoint,
			}, nil
		})
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithEndpointResolverWithOptions(customResolver))
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	return cfg, err
}
