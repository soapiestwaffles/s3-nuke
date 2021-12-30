package aws

import (
	"context"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go/aws"
)

type S3APIMock struct {
	options s3.Options
}

func TestNewS3Service(t *testing.T) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		t.Errorf("error creating aws sdk default config: %v", err)
	}

	tests := []struct {
		name     string
		s3Client S3API
	}{
		{
			name: "s3 API mock",
			s3Client: S3APIMock{
				options: s3.Options{},
			},
		},
		{
			name:     "aws s3 service",
			s3Client: s3.NewFromConfig(cfg),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewS3Service(tt.s3Client)
			if got == nil {
				t.Errorf("NewS3Service() returned nil")
			}

			val := reflect.ValueOf(got).Elem()

			if val.Type().Field(0).Name != "client" {
				t.Errorf("NewS3Service() did not return s3Service struct containing field `client`")
			}

		})
	}
}

func Test_s3Service_GetAllBuckets(t *testing.T) {
	s3 := NewS3Service(&S3APIMock{})

	result, err := s3.GetAllBuckets(context.Background())
	if err != nil {
		t.Errorf("error while GetAllBuckets(), %v", err)
		return
	}

	for i, b := range result {
		t.Logf("GetAllBuckets(): bucket name [%d]: %s", i, *b.Name)
	}

	if *result[0].Name != "bucket1" || *result[1].Name != "bucket2" {
		t.Errorf("error getting bucket name results")
	}

}

// =================

func (s S3APIMock) ListBuckets(ctx context.Context,
	params *s3.ListBucketsInput,
	optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {

	buckets := []types.Bucket{
		{Name: aws.String("bucket1")},
		{Name: aws.String("bucket2")},
	}

	output := &s3.ListBucketsOutput{
		Buckets: buckets,
	}

	return output, nil
}
