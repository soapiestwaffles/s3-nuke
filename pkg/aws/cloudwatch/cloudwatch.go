package cloudwatch

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/config"
)

// StorageType defines the name of an S3 storage class
type StorageType string

const (
	StandardStorage                StorageType = "StandardStorage"
	IntelligentTieringFAStorage    StorageType = "IntelligentTieringFAStorage"
	IntelligentTieringIAStorage    StorageType = "IntelligentTieringIAStorage"
	IntelligentTieringAAStorage    StorageType = "IntelligentTieringAAStorage"
	IntelligentTieringAIAStorage   StorageType = "IntelligentTieringAIAStorage"
	IntelligentTieringDAAStorage   StorageType = "IntelligentTieringDAAStorage"
	StandardIAStorage              StorageType = "StandardIAStorage"
	StandardIASizeOverhead         StorageType = "StandardIASizeOverhead"
	StandardIAObjectOverhead       StorageType = "StandardIAObjectOverhead"
	OneZoneIAStorage               StorageType = "OneZoneIAStorage"
	OneZoneIASizeOverhead          StorageType = "OneZoneIASizeOverhead"
	ReducedRedundancyStorage       StorageType = "ReducedRedundancyStorage"
	GlacierInstantRetrievalStorage StorageType = "GlacierInstantRetrievalStorage"
	GlacierStorage                 StorageType = "GlacierStorage"
	GlacierStagingStorage          StorageType = "GlacierStagingStorage"
	GlacierObjectOverhead          StorageType = "GlacierObjectOverhead"
	GlacierS3ObjectOverhead        StorageType = "GlacierS3ObjectOverhead"
	DeepArchiveStorage             StorageType = "DeepArchiveStorage"
	DeepArchiveObjectOverhead      StorageType = "DeepArchiveObjectOverhead"
	DeepArchiveS3ObjectOverhead    StorageType = "DeepArchiveS3ObjectOverhead"
	DeepArchiveStagingStorage      StorageType = "DeepArchiveStagingStorage"
)

// Service defines functions related to Cloudwatch operations
type Service interface {
	// GetS3ObjectCount returns the amount of objects in an S3 bucket at the time of the last cloudwatch metric for ALL storage types
	//
	// time units:
	//   `startTimeDiff` - hours
	//   `period`        - seconds
	GetS3ObjectCount(ctx context.Context, bucketName string, startTimeDiff int, period int32) (*S3ObjectCountResults, error)

	// GetS3ByteCount returns the amount of bytes in an S3 bucket at the time of the last cloudwatch metric for a given storage type
	//
	// see StorageType constants for type
	GetS3ByteCount(ctx context.Context, bucketName string, storageType StorageType, startTimeDiff int, period int32) (*S3ByteCountResults, error)
}

// S3ObjectCountResults contains the results from GetS3ObjectCount()
//
// Timestamps/values are ordered newest first at the 0th index and ordered descending
type S3ObjectCountResults struct {
	Timestamps []time.Time
	Values     []float64
}

// S3ByteCountResults contains the results from GetS3ByteCount()
//
// Timestamps/values are ordered newest first at the 0th index and ordered descending
type S3ByteCountResults struct {
	Timestamps []time.Time
	Values     []float64
}

// ServiceOption is used with NewService and configures the newly created s3Service
type ServiceOption func(s *service)

type service struct {
	client      CloudwatchAPI
	awsEndpoint string
	region      string
}

// NewService returns an initialized Cloudwatch service
func NewService(opts ...ServiceOption) Service {
	svc := &service{}
	for _, opt := range opts {
		opt(svc)
	}

	if svc.client == nil {
		if svc.region == "" {
			svc.client = newClient(os.Getenv("AWS_REGION"), svc.awsEndpoint)
		} else {
			svc.client = newClient(svc.region, svc.awsEndpoint)
		}
	}

	return svc
}

// WithAPI should be used if you want to initialize your own S3 client (such as in cases of a mock S3 client for testing)
// This cannot be used with WithAWSEndpoint
func WithAPI(client CloudwatchAPI) ServiceOption {
	return func(s *service) {
		s.client = client
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

func newClient(region string, awsEndpoint string) *cloudwatch.Client {
	// Initialize AWS S3 Client
	cfg, err := config.New(region, awsEndpoint)
	if err != nil {
		return nil
	}

	return cloudwatch.NewFromConfig(cfg)
}

func (s *service) GetS3ObjectCount(ctx context.Context, bucketName string, startTimeDiff int, period int32) (*S3ObjectCountResults, error) {
	returnValues := &S3ObjectCountResults{}

	var nextToken *string
	for {
		result, err := s.client.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
			EndTime:   aws.Time(time.Unix(time.Now().Unix(), 0)),
			StartTime: aws.Time(time.Unix(time.Now().Add(time.Duration(-startTimeDiff)*time.Hour).Unix(), 0)),
			NextToken: nextToken,
			MetricDataQueries: []types.MetricDataQuery{
				{
					Id:    aws.String("m1"),
					Label: aws.String("Number of objects"),
					// Period: aws.Int32(int32(period)),
					MetricStat: &types.MetricStat{
						Metric: &types.Metric{
							Namespace:  aws.String("AWS/S3"),
							MetricName: aws.String("NumberOfObjects"),
							Dimensions: []types.Dimension{
								{
									Name:  aws.String("BucketName"),
									Value: aws.String(bucketName),
								},
								{
									Name:  aws.String("StorageType"),
									Value: aws.String("AllStorageTypes"),
								},
							},
						},
						Period: aws.Int32(period),
						Stat:   aws.String("Average"),
					},
				},
			},
		})

		if err != nil {
			return nil, err
		}

		// Copy this set of results into our returnValues
		for _, r := range result.MetricDataResults {
			returnValues.Values = append(returnValues.Values, r.Values...)
			returnValues.Timestamps = append(returnValues.Timestamps, r.Timestamps...)
		}

		// Check if we have another set to load, if not, break the loop
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return returnValues, nil
}

func (s *service) GetS3ByteCount(ctx context.Context, bucketName string, storageType StorageType, startTimeDiff int, period int32) (*S3ByteCountResults, error) {
	returnValues := &S3ByteCountResults{}

	var nextToken *string
	for {
		result, err := s.client.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
			EndTime:   aws.Time(time.Unix(time.Now().Unix(), 0)),
			StartTime: aws.Time(time.Unix(time.Now().Add(time.Duration(-startTimeDiff)*time.Hour).Unix(), 0)),
			NextToken: nextToken,
			MetricDataQueries: []types.MetricDataQuery{
				{
					Id:    aws.String("m1"),
					Label: aws.String("Number of bytes"),
					// Period: aws.Int32(int32(period)),
					MetricStat: &types.MetricStat{
						Metric: &types.Metric{
							Namespace:  aws.String("AWS/S3"),
							MetricName: aws.String("BucketSizeBytes"),
							Dimensions: []types.Dimension{
								{
									Name:  aws.String("BucketName"),
									Value: aws.String(bucketName),
								},
								{
									Name:  aws.String("StorageType"),
									Value: aws.String(string(storageType)),
								},
							},
						},
						Period: aws.Int32(period),
						Stat:   aws.String("Average"),
					},
				},
			},
		})

		if err != nil {
			return nil, err
		}

		// Copy this set of results into our returnValues
		for _, r := range result.MetricDataResults {
			returnValues.Values = append(returnValues.Values, r.Values...)
			returnValues.Timestamps = append(returnValues.Timestamps, r.Timestamps...)
		}

		// Check if we have another set to load, if not, break the loop
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return returnValues, nil
}

// =====

// CloudwatchAPI defines the interface for AWS S3 SDK functions
type CloudwatchAPI interface {
	GetMetricData(ctx context.Context,
		params *cloudwatch.GetMetricDataInput,
		optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
}
