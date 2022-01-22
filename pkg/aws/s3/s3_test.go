package s3

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/google/uuid"
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
			svc := &service{}
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
			svc := &service{}
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

func Test_service_GetBucektRegion(t *testing.T) {
	s3Mock := S3APIMock{
		options: s3.Options{},
		t:       t,
	}
	s3MockFail := S3APIMockFail{
		options: s3.Options{},
		t:       t,
	}

	type fields struct {
		client      S3API
		awsEndpoint string
		region      string
	}
	tests := []struct {
		name       string
		fields     fields
		bucketName string
		want       string
		wantErr    bool
	}{
		{
			name: "get bucket region us-west-2",
			fields: fields{
				client:      s3Mock,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			bucketName: "testbucket",
			want:       "us-west-2",
			wantErr:    false,
		},
		{
			name: "get bucket region us-east-1",
			fields: fields{
				client:      s3Mock,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			bucketName: "testbucket-us-east-1",
			want:       "us-east-1",
			wantErr:    false,
		},
		{
			name: "get bucket fail",
			fields: fields{
				client:      s3MockFail,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			bucketName: "testbucket-us-east-1",
			want:       "",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				client:      tt.fields.client,
				awsEndpoint: tt.fields.awsEndpoint,
				region:      tt.fields.region,
			}
			got, err := s.GetBucektRegion(context.TODO(), tt.bucketName)
			if (err != nil) != tt.wantErr {
				t.Errorf("service.GetBucektRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("service.GetBucektRegion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_service_ListObjects(t *testing.T) {
	s3Mock := S3APIMock{
		options: s3.Options{},
		t:       t,
	}
	s3MockFail := S3APIMockFail{
		options: s3.Options{},
		t:       t,
	}

	finalToken := "third-token"
	thirdToken := "second-token"

	type fields struct {
		client      S3API
		awsEndpoint string
		region      string
	}
	type args struct {
		ctx               context.Context
		bucketName        string
		continuationToken *string
		prefix            *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		want1   *string
		wantErr bool
	}{
		{
			name: "list objects",
			fields: fields{
				client:      s3Mock,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:               context.TODO(),
				bucketName:        "testbucket",
				continuationToken: nil,
			},
			wantErr: false,
		},
		{
			name: "fail",
			fields: fields{
				client:      s3MockFail,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:               context.TODO(),
				bucketName:        "testbucket",
				continuationToken: nil,
			},
			wantErr: true,
		},
		{
			name: "list objects with final token",
			fields: fields{
				client:      s3Mock,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:               context.TODO(),
				bucketName:        "testbucket",
				continuationToken: &finalToken,
			},
			wantErr: false,
		},
		{
			name: "list objects with third token",
			fields: fields{
				client:      s3Mock,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:               context.TODO(),
				bucketName:        "testbucket",
				continuationToken: &thirdToken,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				client:      tt.fields.client,
				awsEndpoint: tt.fields.awsEndpoint,
				region:      tt.fields.region,
			}
			continuationToken := tt.args.continuationToken
			for {
				keys, token, err := s.ListObjects(tt.args.ctx, tt.args.bucketName, continuationToken, tt.args.prefix)
				t.Logf("service.ListObjects() returned num keys %d, token %v", len(keys), token)
				if (err != nil) != tt.wantErr {
					t.Errorf("service.ListObjects() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if tt.wantErr {
					break
				}
				if continuationToken == nil && token == nil {
					t.Errorf("service.ListObjects() error, wanted continationToken but got nil")
					return
				}
				if len(keys) < 1 {
					t.Errorf("service.ListObjects() error, keys is empty")
					return
				}

				if token != nil {
					continuationToken = token
				} else {
					break
				}
			}
		})
	}
}

func Test_service_ListObjectVersions(t *testing.T) {
	s3Mock := S3APIMock{
		options: s3.Options{},
		t:       t,
	}
	s3MockFail := S3APIMockFail{
		options: s3.Options{},
		t:       t,
	}

	finalToken := "third-token"
	thirdToken := "second-token"

	type fields struct {
		client      S3API
		awsEndpoint string
		region      string
	}
	type args struct {
		ctx             context.Context
		bucketName      string
		keyMarker       *string
		versionIDMarker *string
		prefix          *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "list object versions",
			fields: fields{
				client:      s3Mock,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:             context.TODO(),
				bucketName:      "testbucket",
				keyMarker:       nil,
				versionIDMarker: nil,
				prefix:          nil,
			},
			wantErr: false,
		},
		{
			name: "list object versions with third token",
			fields: fields{
				client:      s3Mock,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:             context.TODO(),
				bucketName:      "testbucket",
				keyMarker:       &thirdToken,
				versionIDMarker: aws.String("randomVersionIDMarker"),
				prefix:          nil,
			},
			wantErr: false,
		},
		{
			name: "list object versions with final token",
			fields: fields{
				client:      s3Mock,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:             context.TODO(),
				bucketName:      "testbucket",
				keyMarker:       &finalToken,
				versionIDMarker: aws.String("randomVersionIDMarker"),
				prefix:          nil,
			},
			wantErr: false,
		},
		{
			name: "fail",
			fields: fields{
				client:      s3MockFail,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:             context.TODO(),
				bucketName:      "testbucket",
				keyMarker:       nil,
				versionIDMarker: nil,
				prefix:          nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				client:      tt.fields.client,
				awsEndpoint: tt.fields.awsEndpoint,
				region:      tt.fields.region,
			}
			keyMarkerToken := tt.args.keyMarker
			versionIDMarkerToken := tt.args.versionIDMarker
			for {
				versions, keyMarker, versionIDMarker, err := s.ListObjectVersions(tt.args.ctx, tt.args.bucketName, keyMarkerToken, versionIDMarkerToken, tt.args.prefix)
				if (err != nil) != tt.wantErr {
					t.Errorf("service.ListObjectVersions() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if tt.wantErr {
					break
				}
				t.Logf("service.ListObjectVersions() returned num versions %d, keyMarker %v, versionIDMarker %v",
					len(versions), keyMarker, versionIDMarker)
				if keyMarkerToken == nil && keyMarker == nil {
					t.Errorf("service.ListObjectVersions() error, wanted keyMarkerToken but got nil")
					return
				}
				if versionIDMarkerToken == nil && versionIDMarker == nil {
					t.Errorf("service.ListObjectVersions() error, wanted versionIDMarker but got nil")
					return
				}
				if len(versions) < 1 {
					t.Errorf("service.ListObjectVersions() error, versions is empty")
				}

				if keyMarker != nil {
					keyMarkerToken = keyMarker
				} else {
					break
				}

				if versionIDMarker != nil {
					versionIDMarkerToken = versionIDMarker
				} else {
					break
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

func (s S3APIMock) GetBucketLocation(ctx context.Context,
	params *s3.GetBucketLocationInput,
	optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {

	s.t.Logf("get bucket location [%s]", *params.Bucket)

	if *params.Bucket == "testbucket-us-east-1" {
		return &s3.GetBucketLocationOutput{
			LocationConstraint: "",
			ResultMetadata:     middleware.Metadata{},
		}, nil
	}

	return &s3.GetBucketLocationOutput{
		LocationConstraint: types.BucketLocationConstraintUsWest2,
		ResultMetadata:     middleware.Metadata{},
	}, nil

}

func (s S3APIMockFail) GetBucketLocation(ctx context.Context,
	params *s3.GetBucketLocationInput,
	optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {

	s.t.Logf("get bucket location [%s]", *params.Bucket)

	return nil, errors.New("simulated error case")

}

func (s S3APIMock) ListObjectsV2(ctx context.Context,
	params *s3.ListObjectsV2Input,
	optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {

	rand.Seed(time.Now().Unix())

	if params.ContinuationToken != nil {
		s.t.Logf("list objects: bucket [%s] continuationToken [%s]", *params.Bucket, *params.ContinuationToken)
	} else {
		s.t.Logf("list objects: bucket [%s]", *params.Bucket)
	}

	returnValue := s3.ListObjectsV2Output{
		CommonPrefixes:    []types.CommonPrefix{},
		Contents:          []types.Object{},
		ContinuationToken: params.ContinuationToken,
		// Delimiter:         new(string),
		EncodingType: "",
		// IsTruncated:       false,
		MaxKeys: params.MaxKeys,
		Name:    params.Bucket,
		Prefix:  new(string),
	}

	if params.ContinuationToken == nil {
		// first call
		returnValue.NextContinuationToken = aws.String("first-token")
		returnValue.IsTruncated = true
	} else {
		switch *params.ContinuationToken {
		case "first-token":
			returnValue.NextContinuationToken = aws.String("second-token")
			returnValue.IsTruncated = true
		case "second-token":
			returnValue.NextContinuationToken = aws.String("third-token")
			returnValue.IsTruncated = true
		default:
			returnValue.NextContinuationToken = nil
			returnValue.IsTruncated = false
		}
	}

	itemCount := rand.Intn(500) + 1
	lastModified := time.Now()
	for i := 0; i < itemCount; i++ {
		returnValue.Contents = append(returnValue.Contents, types.Object{
			ETag:         aws.String(uuid.NewString()),
			Key:          aws.String(uuid.NewString()),
			LastModified: &lastModified,
			Owner:        &types.Owner{},
			Size:         rand.Int63n(1000000),
			StorageClass: "StandardStorage",
		})
	}
	returnValue.KeyCount = int32(itemCount)

	return &returnValue, nil
}

func (s S3APIMockFail) ListObjectsV2(ctx context.Context,
	params *s3.ListObjectsV2Input,
	optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {

	s.t.Logf("get bucket location [%s]", *params.Bucket)

	return nil, errors.New("simulated error case")
}

func (s S3APIMock) ListObjectVersions(ctx context.Context,
	params *s3.ListObjectVersionsInput,
	optFns ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error) {

	if params.KeyMarker != nil && params.VersionIdMarker != nil {
		s.t.Logf("get object versions [%s], keyMarker [%s], versionIDMarker [%s]", *params.Bucket, *params.KeyMarker, *params.VersionIdMarker)
	} else if params.KeyMarker != nil {
		s.t.Logf("get object versions [%s], keyMarker [%s]", *params.Bucket, *params.KeyMarker)
	} else if params.VersionIdMarker != nil {
		s.t.Logf("get object versions [%s], versionIDMarker [%s]", *params.Bucket, *params.VersionIdMarker)
	} else {
		s.t.Logf("get object versions [%s]", *params.Bucket)
	}

	returnValue := s3.ListObjectVersionsOutput{
		IsTruncated:         false,
		KeyMarker:           params.KeyMarker,
		MaxKeys:             1000,
		Name:                params.Bucket,
		Prefix:              params.Prefix,
		VersionIdMarker:     params.VersionIdMarker,
		Versions:            []types.ObjectVersion{},
		NextVersionIdMarker: aws.String("versionIdMarker"),
	}

	if params.KeyMarker == nil {
		// first call
		returnValue.NextKeyMarker = aws.String("first-token")
		returnValue.IsTruncated = true
	} else {
		switch *params.KeyMarker {
		case "first-token":
			returnValue.NextKeyMarker = aws.String("second-token")
			returnValue.IsTruncated = true
		case "second-token":
			returnValue.NextKeyMarker = aws.String("third-token")
			returnValue.IsTruncated = true
		default:
			returnValue.NextKeyMarker = nil
			returnValue.IsTruncated = false
		}
	}

	itemCount := rand.Intn(500) + 1
	lastModified := time.Now()
	for i := 0; i < itemCount; i++ {
		returnValue.Versions = append(returnValue.Versions, types.ObjectVersion{
			ETag:         aws.String(uuid.NewString()),
			IsLatest:     false,
			Key:          aws.String(uuid.NewString()),
			LastModified: &lastModified,
			Owner:        &types.Owner{},
			Size:         12345,
			StorageClass: "StandardStorage",
			VersionId:    aws.String(uuid.NewString()),
		})
	}

	for i := 0; i < itemCount; i++ {
		returnValue.DeleteMarkers = append(returnValue.DeleteMarkers, types.DeleteMarkerEntry{
			IsLatest:     false,
			Key:          aws.String(uuid.NewString()),
			LastModified: &lastModified,
			Owner:        &types.Owner{},
			VersionId:    aws.String(uuid.NewString()),
		})
	}

	return &returnValue, nil
}

func (s S3APIMockFail) ListObjectVersions(ctx context.Context,
	params *s3.ListObjectVersionsInput,
	optFns ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error) {

	s.t.Logf("get object versions [%s]", *params.Bucket)

	return nil, errors.New("simulated error case")
}

func (s S3APIMock) DeleteObjects(ctx context.Context,
	params *s3.DeleteObjectsInput,
	optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	rand.Seed(time.Now().UnixNano())

	s.t.Logf("delete objects bucket [%s], num objects [%d]", *params.Bucket, len(params.Delete.Objects))

	returnValue := s3.DeleteObjectsOutput{
		Deleted:        []types.DeletedObject{},
		Errors:         []types.Error{},
		RequestCharged: "",
		ResultMetadata: middleware.Metadata{},
	}

	for _, object := range params.Delete.Objects {
		returnValue.Deleted = append(returnValue.Deleted, types.DeletedObject{
			DeleteMarker:          rand.Uint64()&(1<<63) == 0,
			DeleteMarkerVersionId: aws.String(uuid.NewString()),
			Key:                   object.Key,
			VersionId:             object.VersionId,
		})
	}

	return &returnValue, nil

}

func (s S3APIMockFail) DeleteObjects(ctx context.Context,
	params *s3.DeleteObjectsInput,
	optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {

	s.t.Logf("delete objects bucket [%s], num objects [%d]", *params.Bucket, len(params.Delete.Objects))

	return nil, errors.New("simulated error case")
}
