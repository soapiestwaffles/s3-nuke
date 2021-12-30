package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Service defines functions related to S3 operations
type S3Service interface {

	// ListBuckets loads buckets (up to the first 500) into memory
	// TODO: Make bucket name retrieval limit more dynamic instead of capped at 500
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

type s3Service struct {
	client S3API
}

// NewS3Service returns an initialized S3Service
func NewS3Service(s3Client S3API) S3Service {
	if s3Client == nil {
		return nil
	}
	return &s3Service{client: s3Client}
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
