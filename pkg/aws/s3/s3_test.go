package s3

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
)

type S3APIMock struct {
	options s3.Options
	t       *testing.T
}

type S3APIMockFail struct {
	options s3.Options
	t       *testing.T
}

func TestNewService(t *testing.T) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO())
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
			got := NewService(WithS3API(tt.s3Client))
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
		s3 := NewService(WithS3API(&S3APIMock{options: s3.Options{}}))

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
		s3 := NewService(WithS3API(&S3APIMockFail{options: s3.Options{}}))
		_, err := s3.GetAllBuckets(context.Background())
		if err == nil {
			t.Errorf("expected to get error")
			return
		}
	})
}

func TestWithAWSEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		awsEndpoint string
	}{
		{
			name:        "with endpoint",
			awsEndpoint: "http://test.com:1234",
		},
		{
			name:        "with empty endpoint",
			awsEndpoint: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &s3Service{}
			f := WithAWSEndpoint(tt.awsEndpoint)
			f(svc)
			if svc.awsEndpoint != tt.awsEndpoint {
				t.Errorf("WithAWSEndpoint(): set awsEndpoint to %s, but got %s", tt.awsEndpoint, svc.awsEndpoint)
			}
		})
	}
}

func TestWithRegion(t *testing.T) {
	tests := []struct {
		name   string
		region string
	}{
		{
			name:   "us-west-2",
			region: "us-west-2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &s3Service{}
			f := WithRegion(tt.region)
			f(svc)
			if svc.region != tt.region {
				t.Errorf("WithRegion(): set region to %s, but got %s", tt.region, svc.region)
			}
		})
	}
}

func Test_s3Service_CreateBucketSimple(t *testing.T) {
	s3Mock := S3APIMock{
		options: s3.Options{},
		t:       t,
	}
	s3MockFail := S3APIMockFail{
		options: s3.Options{},
		t:       t,
	}
	type args struct {
		ctx        context.Context
		bucketName string
		versioned  bool
		region     string
	}
	tests := []struct {
		name    string
		client  S3API
		args    args
		wantErr bool
	}{
		{
			name:   "create new bucket not versioned",
			client: s3Mock,
			args: args{
				ctx:        context.TODO(),
				bucketName: "testbucket",
				versioned:  false,
				region:     "us-west-2",
			},
			wantErr: false,
		},
		{
			name:   "fail create new bucket not versioned",
			client: s3MockFail,
			args: args{
				ctx:        context.TODO(),
				bucketName: "testbucket",
				versioned:  false,
				region:     "us-west-2",
			},
			wantErr: true,
		},
		{
			name:   "create new bucket versioned",
			client: s3Mock,
			args: args{
				ctx:        context.TODO(),
				bucketName: "testbucket",
				versioned:  true,
				region:     "us-west-2",
			},
			wantErr: false,
		},
		{
			name:   "fail create new bucket versioned",
			client: s3MockFail,
			args: args{
				ctx:        context.TODO(),
				bucketName: "testbucket",
				versioned:  true,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			s := NewService(
				WithS3API(tt.client))

			if err := s.CreateBucketSimple(tt.args.ctx, tt.args.bucketName, tt.args.region, tt.args.versioned); (err != nil) != tt.wantErr {
				t.Errorf("s3Service.CreateBucketSimple() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_s3Service_PutObjectSimple(t *testing.T) {
	s3Mock := S3APIMock{
		options: s3.Options{},
		t:       t,
	}
	s3MockFail := S3APIMockFail{
		options: s3.Options{},
		t:       t,
	}

	reader1 := strings.NewReader("🧪 This is a test body 1")
	reader2 := strings.NewReader("🧪 This is a test body 2")
	type args struct {
		ctx        context.Context
		bucketName string
		keyName    string
		body       io.Reader
	}
	tests := []struct {
		name    string
		client  S3API
		args    args
		wantErr bool
	}{
		{
			name:   "put file no error",
			client: s3Mock,
			args: args{
				ctx:        context.TODO(),
				bucketName: "testbucket",
				keyName:    "testkey",
				body:       reader1,
			},
			wantErr: false,
		},
		{
			name:   "put file error",
			client: s3MockFail,
			args: args{
				ctx:        context.TODO(),
				bucketName: "testbucket",
				keyName:    "testkey",
				body:       reader2,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			s := NewService(
				WithS3API(tt.client))

			_, _, err := s.PutObjectSimple(tt.args.ctx, tt.args.bucketName, tt.args.keyName, tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("s3Service.PutObjectSimple() error = %v, wantErr %v", err, tt.wantErr)
				return
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

func (s S3APIMock) CreateBucket(ctx context.Context,
	params *s3.CreateBucketInput,
	optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	s.t.Logf("created bucket: [%s]", *params.Bucket)
	return &s3.CreateBucketOutput{
		Location: params.Bucket,
	}, nil
}

func (s S3APIMockFail) CreateBucket(ctx context.Context,
	params *s3.CreateBucketInput,
	optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	s.t.Logf("created bucket: [%s], with failure", *params.Bucket)
	return nil, errors.New("simulated error case")
}

func (s S3APIMock) PutBucketVersioning(ctx context.Context,
	params *s3.PutBucketVersioningInput,
	optFns ...func(*s3.Options)) (*s3.PutBucketVersioningOutput, error) {
	s.t.Logf("version bucket: [%s], status: [%s]", *params.Bucket, params.VersioningConfiguration.Status)
	return &s3.PutBucketVersioningOutput{}, nil
}

func (s S3APIMockFail) PutBucketVersioning(ctx context.Context,
	params *s3.PutBucketVersioningInput,
	optFns ...func(*s3.Options)) (*s3.PutBucketVersioningOutput, error) {
	s.t.Logf("version bucket: [%s], status: [%s], with failure", *params.Bucket, params.VersioningConfiguration.Status)
	return nil, errors.New("simulated error case")
}

func (s S3APIMock) PutObject(ctx context.Context,
	params *s3.PutObjectInput,
	optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	s.t.Logf("put object: [%s], bucket: [%s]", *params.Key, *params.Bucket)

	buf := make([]byte, 1024)
	for {
		n, err := params.Body.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if n > 0 {
			s.t.Logf("read from body (1024bytes):\n%s", string(buf[:n]))
		}
	}

	return &s3.PutObjectOutput{
		BucketKeyEnabled: false,
		ETag:             aws.String("123456789ABCDEF"),
		VersionId:        aws.String("123456789ABCDEF"),
		ResultMetadata:   middleware.Metadata{},
	}, nil
}

func (s S3APIMockFail) PutObject(ctx context.Context,
	params *s3.PutObjectInput,
	optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	s.t.Logf("put object: [%s], bucket: [%s], with failure", *params.Key, *params.Bucket)

	buf := make([]byte, 1024)
	for {
		n, err := params.Body.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if n > 0 {
			s.t.Logf("read from body (1024bytes):\n%s", string(buf[:n]))
		}
	}

	return nil, errors.New("simulated error case")
}
