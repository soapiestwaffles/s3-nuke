package s3

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/config"
)

// Service defines functions related to S3 operations
type Service interface {
	// ListBuckets loads buckets into memory
	GetAllBuckets(ctx context.Context) ([]Bucket, error)

	// CreateBucketSimple creates a new, simple S3 bucket in the current/default region
	//
	// (This function is mainly used in s3-nuke for testing)
	CreateBucketSimple(ctx context.Context, bucketName string, region string, versioned bool) error

	// PutBucketObject puts an object in an S3 bucket
	//
	// returns Etag, VersionID, and Error
	// (This function is mainly used in s3-nuke for testing)
	PutObjectSimple(ctx context.Context, bucketName string, keyName string, body io.Reader) (*string, *string, error)

	// GetBucketRegion will return the region of a bucket
	GetBucektRegion(ctx context.Context, bucketName string) (string, error)
}

// Bucket contains information about an S3 bucket
type Bucket struct {
	CreationDate *time.Time
	Name         *string
}

// ServiceOption is used with NewS3Service and configures the newly created s3Service
type ServiceOption func(s *service)

type service struct {
	client      S3API
	awsEndpoint string
	region      string
}

// NewService returns an initialized S3Service
func NewService(opts ...ServiceOption) Service {
	svc := &service{}
	for _, opt := range opts {
		opt(svc)
	}

	if svc.client == nil {
		if svc.region == "" {
			svc.client = newS3Client(os.Getenv("AWS_REGION"), svc.awsEndpoint)
		} else {
			svc.client = newS3Client(svc.region, svc.awsEndpoint)
		}
	}

	return svc
}

// WithS3API should be used if you want to initialize your own S3 client (such as in cases of a mock S3 client for testing)
// This cannot be used with WithAWSEndpoint
func WithS3API(s3Client S3API) ServiceOption {
	return func(s *service) {
		s.client = s3Client
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

func (s *service) GetAllBuckets(ctx context.Context) ([]Bucket, error) {
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

func (s *service) CreateBucketSimple(ctx context.Context, bucketName string, region string, versioned bool) error {
	_, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucketName,
		ACL:    types.BucketCannedACLPrivate,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		},
	})
	if err != nil {
		return err
	}

	if versioned {
		_, err := s.client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
			Bucket: &bucketName,
			VersioningConfiguration: &types.VersioningConfiguration{
				Status: "Enabled",
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *service) PutObjectSimple(ctx context.Context, bucketName string, keyName string, body io.Reader) (*string, *string, error) {
	result, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &keyName,
		Body:   body,
	})

	if err != nil {
		return nil, nil, err
	}

	return result.ETag, result.VersionId, nil
}

func (s *service) GetBucektRegion(ctx context.Context, bucketName string) (string, error) {
	result, err := s.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: &bucketName,
	})

	if err != nil {
		return "", err
	}

	if result.LocationConstraint == "" {
		return "us-east-1", nil
	}

	return string(result.LocationConstraint), nil

}

func newS3Client(region string, awsEndpoint string) *s3.Client {
	// Initialize AWS S3 Client
	cfg, err := config.New(region, awsEndpoint)
	if err != nil {
		return nil
	}

	return s3.NewFromConfig(cfg)
}

// =====

// S3API defines the interface for AWS S3 SDK functions
type S3API interface {
	ListBuckets(ctx context.Context,
		params *s3.ListBucketsInput,
		optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)

	CreateBucket(ctx context.Context,
		params *s3.CreateBucketInput,
		optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)

	PutBucketVersioning(ctx context.Context,
		params *s3.PutBucketVersioningInput,
		optFns ...func(*s3.Options)) (*s3.PutBucketVersioningOutput, error)

	PutObject(ctx context.Context,
		params *s3.PutObjectInput,
		optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)

	GetBucketLocation(ctx context.Context,
		params *s3.GetBucketLocationInput,
		optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
}
