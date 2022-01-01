package aws

import (
	"context"
	"errors"
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

type S3APIMockFail struct {
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
		{
			name:     "nil test",
			s3Client: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewS3Service(WithS3API(tt.s3Client))
			if got == nil {
				t.Errorf("NewS3Service() returned nil when it wasn't supposed to")
			} else {
				val := reflect.ValueOf(got).Elem()

				if val.Type().Field(0).Name != "client" {
					t.Errorf("NewS3Service() did not return s3Service struct containing field `client`")
				}
			}
		})
	}
}

func Test_s3Service_GetAllBuckets(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		s3 := NewS3Service(WithS3API(&S3APIMock{options: s3.Options{}}))

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
	})

	t.Run("fail", func(t *testing.T) {
		s3 := NewS3Service(WithS3API(&S3APIMockFail{options: s3.Options{}}))
		_, err := s3.GetAllBuckets(context.Background())
		if err == nil {
			t.Errorf("expected to get error")
			return
		}
	})
}

func Test_newConfig(t *testing.T) {
	tests := []struct {
		name        string
		awsEndpoint string
		wantErr     bool
	}{
		{
			name:        "endpoint not set",
			awsEndpoint: "",
			wantErr:     false,
		},
		{
			name:        "endpoint set",
			awsEndpoint: "http://test-endpoint.com:1234",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newConfig(tt.awsEndpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("newConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.awsEndpoint == "" {
				if got.EndpointResolverWithOptions != nil {
					t.Errorf("%s: newConfig() error, EndpointResolverWithOptions was set but was supposed to be nil", tt.name)
				}
			} else {
				if got.EndpointResolverWithOptions == nil {
					t.Errorf("%s: newConfig() error, EndpointResolveWithOptions was nil but was supposed to be set", tt.name)
				}
			}
		})
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

func (s S3APIMockFail) ListBuckets(ctx context.Context,
	params *s3.ListBucketsInput,
	optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {

	return nil, errors.New("simulated error case")
}
