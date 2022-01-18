package cloudwatch

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/smithy-go/middleware"
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

// =================

func (s CloudwatchAPIMock) GetMetricData(ctx context.Context,
	params *cloudwatch.GetMetricDataInput,
	optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {

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
