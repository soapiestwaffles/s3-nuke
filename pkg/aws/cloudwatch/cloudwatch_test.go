package cloudwatch

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/davecgh/go-spew/spew"
)

type CloudwatchAPIMock struct {
	options cloudwatch.Options
	t       *testing.T
}

type CloudwatchAPIMockFail struct {
	options cloudwatch.Options
	t       *testing.T
}

var now time.Time = time.Now()
var cloudwatchTimestamps []time.Time = []time.Time{
	now,
	now.Add(-time.Hour * 24),
	now.Add(-time.Hour * 48),
	now.Add(-time.Hour * 72),
}

func TestNewService(t *testing.T) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		t.Errorf("error creating aws sdk default config: %v", err)
	}

	tests := []struct {
		name   string
		client CloudwatchAPI
	}{
		{
			name: "cloudwatch API mock",
			client: CloudwatchAPIMock{
				options: cloudwatch.Options{},
				t:       t,
			},
		},
		{
			name:   "aws cloudwatch service",
			client: cloudwatch.NewFromConfig(cfg),
		},
		{
			name:   "nil test",
			client: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewService(WithAPI(tt.client))
			if got == nil {
				t.Errorf("cloudwatch.NewService() returned nil when it wasn't supposed to")
			} else {
				val := reflect.ValueOf(got).Elem()

				if val.Type().Field(0).Name != "client" {
					t.Errorf("cloudwatch.NewService() did not return service struct containing field `client`")
				}
			}
		})
	}
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

func Test_service_GetS3ObjectCount(t *testing.T) {
	cloudwatchMock := CloudwatchAPIMock{
		options: cloudwatch.Options{},
		t:       t,
	}
	cloudwatchMockFail := CloudwatchAPIMockFail{
		options: cloudwatch.Options{},
		t:       t,
	}
	type fields struct {
		client      CloudwatchAPI
		awsEndpoint string
		region      string
	}
	type args struct {
		ctx           context.Context
		bucketName    string
		startTimeDiff int
		period        int32
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *S3ObjectCountResults
		wantErr bool
	}{
		{
			name: "get bucket objects",
			fields: fields{
				client:      cloudwatchMock,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:           context.TODO(),
				bucketName:    "testbucket",
				startTimeDiff: 72,
				period:        60,
			},
			want: &S3ObjectCountResults{
				Timestamps: cloudwatchTimestamps,
				Values: []float64{
					10.0,
					20.0,
					30.0,
					40.0,
				},
			},
			wantErr: false,
		},
		{
			name: "fail",
			fields: fields{
				client:      cloudwatchMockFail,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:           context.TODO(),
				bucketName:    "failbucket",
				startTimeDiff: 72,
				period:        60,
			},
			want:    nil,
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
			got, err := s.GetS3ObjectCount(tt.args.ctx, tt.args.bucketName, tt.args.startTimeDiff, tt.args.period)
			if (err != nil) != tt.wantErr {
				t.Errorf("service.GetS3ObjectCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("service.GetS3ObjectCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_service_GetS3ByteCount(t *testing.T) {
	cloudwatchMock := CloudwatchAPIMock{
		options: cloudwatch.Options{},
		t:       t,
	}
	cloudwatchMockFail := CloudwatchAPIMockFail{
		options: cloudwatch.Options{},
		t:       t,
	}
	type fields struct {
		client      CloudwatchAPI
		awsEndpoint string
		region      string
	}
	type args struct {
		ctx           context.Context
		bucketName    string
		storageType   StorageType
		startTimeDiff int
		period        int32
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *S3ByteCountResults
		wantErr bool
	}{
		{
			name: "get bucket objects",
			fields: fields{
				client:      cloudwatchMock,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:           context.TODO(),
				bucketName:    "testbucket",
				storageType:   StandardStorage,
				startTimeDiff: 72,
				period:        60,
			},
			want: &S3ByteCountResults{
				Timestamps: cloudwatchTimestamps,
				Values: []float64{
					10.0,
					20.0,
					30.0,
					40.0,
				},
			},
			wantErr: false,
		},
		{
			name: "fail",
			fields: fields{
				client:      cloudwatchMockFail,
				awsEndpoint: "",
				region:      "us-west-2",
			},
			args: args{
				ctx:           context.TODO(),
				bucketName:    "failbucket",
				startTimeDiff: 72,
				period:        60,
			},
			want:    nil,
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
			got, err := s.GetS3ByteCount(tt.args.ctx, tt.args.bucketName, tt.args.storageType, tt.args.startTimeDiff, tt.args.period)
			if (err != nil) != tt.wantErr {
				t.Errorf("service.GetS3ObjectCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("service.GetS3ObjectCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =================

func (s CloudwatchAPIMock) GetMetricData(ctx context.Context,
	params *cloudwatch.GetMetricDataInput,
	optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {

	s.t.Logf("request metrics: %s", spew.Sdump(params))
	output := &cloudwatch.GetMetricDataOutput{
		Messages: []types.MessageData{},
		MetricDataResults: []types.MetricDataResult{{
			Id:         aws.String("id"),
			Label:      aws.String("sample data"),
			Messages:   []types.MessageData{},
			StatusCode: "Complete",
			Timestamps: cloudwatchTimestamps,
			Values: []float64{
				10.0,
				20.0,
				30.0,
				40.0,
			},
		}},
		NextToken:      nil,
		ResultMetadata: middleware.Metadata{},
	}

	return output, nil
}

func (s CloudwatchAPIMockFail) GetMetricData(ctx context.Context,
	params *cloudwatch.GetMetricDataInput,
	optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {

	return nil, errors.New("simulated error case")
}

// Test NewService with different scenarios to improve coverage
func TestNewService_Coverage(t *testing.T) {
	// Set AWS_REGION environment variable for testing
	originalRegion := os.Getenv("AWS_REGION")
	defer func() {
		if originalRegion != "" {
			if err := os.Setenv("AWS_REGION", originalRegion); err != nil {
				t.Errorf("Failed to restore AWS_REGION: %v", err)
			}
		} else {
			if err := os.Unsetenv("AWS_REGION"); err != nil {
				t.Errorf("Failed to unset AWS_REGION: %v", err)
			}
		}
	}()

	t.Run("with env region and no options", func(t *testing.T) {
		if err := os.Setenv("AWS_REGION", "us-east-1"); err != nil {
			t.Fatalf("Failed to set AWS_REGION: %v", err)
		}
		svc := NewService()
		if svc == nil {
			t.Errorf("NewService() returned nil")
		}
	})

	t.Run("with custom region option", func(t *testing.T) {
		if err := os.Unsetenv("AWS_REGION"); err != nil {
			t.Fatalf("Failed to unset AWS_REGION: %v", err)
		}
		svc := NewService(WithRegion("us-west-1"))
		if svc == nil {
			t.Errorf("NewService() returned nil")
		}
	})

	t.Run("with aws endpoint and region", func(t *testing.T) {
		if err := os.Unsetenv("AWS_REGION"); err != nil {
			t.Fatalf("Failed to unset AWS_REGION: %v", err)
		}
		svc := NewService(WithAWSEndpoint("http://localhost:4566"), WithRegion("us-east-1"))
		if svc == nil {
			t.Errorf("NewService() returned nil")
		}
	})
}

// Test pagination in GetS3ObjectCount
func TestGetS3ObjectCount_Pagination(t *testing.T) {
	mockWithPagination := CloudwatchAPIMockWithPagination{
		options: cloudwatch.Options{},
		t:       t,
	}

	s := &service{
		client:      mockWithPagination,
		awsEndpoint: "",
		region:      "us-west-2",
	}

	got, err := s.GetS3ObjectCount(context.TODO(), "testbucket", 72, 86400)
	if err != nil {
		t.Errorf("service.GetS3ObjectCount() error = %v", err)
		return
	}

	// Should have values from both pages
	if len(got.Values) < 2 {
		t.Errorf("service.GetS3ObjectCount() expected pagination results, got %d values", len(got.Values))
	}
}

// Test pagination in GetS3ByteCount  
func TestGetS3ByteCount_Pagination(t *testing.T) {
	mockWithPagination := CloudwatchAPIMockWithPagination{
		options: cloudwatch.Options{},
		t:       t,
	}

	s := &service{
		client:      mockWithPagination,
		awsEndpoint: "",
		region:      "us-west-2",
	}

	got, err := s.GetS3ByteCount(context.TODO(), "testbucket", StandardStorage, 72, 86400)
	if err != nil {
		t.Errorf("service.GetS3ByteCount() error = %v", err)
		return
	}

	// Should have values from both pages
	if len(got.Values) < 2 {
		t.Errorf("service.GetS3ByteCount() expected pagination results, got %d values", len(got.Values))
	}
}

// Mock that returns paginated results
type CloudwatchAPIMockWithPagination struct {
	options cloudwatch.Options
	t       *testing.T
}

func (s CloudwatchAPIMockWithPagination) GetMetricData(ctx context.Context,
	params *cloudwatch.GetMetricDataInput,
	optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {

	var nextToken *string
	var values []float64
	
	// Simulate pagination - first call has NextToken, second doesn't
	if params.NextToken == nil {
		nextToken = aws.String("page2")
		values = []float64{10.0}
	} else {
		nextToken = nil  // No more pages
		values = []float64{20.0}
	}

	output := &cloudwatch.GetMetricDataOutput{
		Messages: []types.MessageData{},
		MetricDataResults: []types.MetricDataResult{{
			Id:         aws.String("id"),
			Label:      aws.String("sample data"),
			Messages:   []types.MessageData{},
			StatusCode: "Complete",
			Timestamps: []time.Time{time.Now()},
			Values:     values,
		}},
		NextToken:      nextToken,
		ResultMetadata: middleware.Metadata{},
	}

	return output, nil
}
