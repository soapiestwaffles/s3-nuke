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
			name: "s3 API mock",
			client: CloudwatchAPIMock{
				options: cloudwatch.Options{},
				t:       t,
			},
		},
		{
			name:   "aws s3 service",
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

// =================

func (s CloudwatchAPIMock) GetMetricData(ctx context.Context,
	params *cloudwatch.GetMetricDataInput,
	optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {

	output := &cloudwatch.GetMetricDataOutput{
		Messages: []types.MessageData{},
		MetricDataResults: []types.MetricDataResult{{
			Id:         aws.String("id1234"),
			Label:      aws.String("somelabel"),
			Messages:   []types.MessageData{},
			StatusCode: "Complete",
			Timestamps: []time.Time{},
			Values:     []float64{},
		}},
		NextToken:      aws.String("abc123def456"),
		ResultMetadata: middleware.Metadata{},
	}

	return output, nil
}

func (s CloudwatchAPIMockFail) GetMetricData(ctx context.Context,
	params *cloudwatch.GetMetricDataInput,
	optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {

	return nil, errors.New("simulated error case")
}
