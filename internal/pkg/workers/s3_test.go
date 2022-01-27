package workers

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/soapiestwaffles/s3-nuke/pkg/aws/s3"
)

var (
	key     = "test"
	version = "version"
)

func TestObjectStack_Push(t *testing.T) {
	type args struct {
		object s3.ObjectIdentifier
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add",
			args: args{
				object: s3.ObjectIdentifier{
					Key:       &key,
					VersionID: &version,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &ObjectStack{}
			o.Push(tt.args.object)

			if o.Queue[0].Key != &key || o.Queue[0].VersionID != &version {
				t.Errorf("ObjectStack.Push() push failed, queue did not contain correct item")
			}
		})
	}
}

func TestObjectStack_Reset(t *testing.T) {
	t.Run("reset", func(t *testing.T) {
		o := &ObjectStack{}
		o.Push(s3.ObjectIdentifier{
			Key:       &key,
			VersionID: &version,
		})

		if o.Len() != 1 {
			t.Errorf("ObjectStack.Reset() could not set up ObjectStack for test")
		}
		o.Reset()

		if o.Len() > 0 {
			t.Errorf("ObjectStack.Reset() failed, Len() > 0")
		}
	})
}

func TestObjectStack_Len(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			name: "5",
			want: 5,
		},
		{
			name: "10",
			want: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &ObjectStack{}
			for i := 0; i < tt.want; i++ {
				o.Push(s3.ObjectIdentifier{
					Key:       &key,
					VersionID: &version,
				})
			}
			if got := o.Len(); got != tt.want {
				t.Errorf("ObjectStack.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_s3DeleteFromChannel(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	s3svc := S3ServiceMock{}
	tests := []struct {
		name    string
		bucket  string
		want    int
		wantErr bool
	}{
		{
			name:    "sub-1000 count",
			bucket:  "testbucket",
			want:    500,
			wantErr: false,
		},
		{
			name:    "1000 count",
			bucket:  "testbucket",
			want:    1000,
			wantErr: false,
		},
		{
			name:    "5000 count",
			bucket:  "testbucket",
			want:    5000,
			wantErr: false,
		},
		{
			name:    "random count 1",
			bucket:  "testbucket",
			want:    rand.Intn(9000),
			wantErr: false,
		},
		{
			name:    "random count 2",
			bucket:  "testbucket",
			want:    rand.Intn(9000),
			wantErr: false,
		},
		{
			name:    "random count 3",
			bucket:  "testbucket",
			want:    rand.Intn(9000),
			wantErr: false,
		},
		{
			name:    "random count 4",
			bucket:  "testbucket",
			want:    rand.Intn(9000),
			wantErr: false,
		},
		{
			name:    "random count 5",
			bucket:  "testbucket",
			want:    rand.Intn(9000),
			wantErr: false,
		},
		{
			name:    "failure",
			bucket:  "failbucket",
			want:    2000,
			wantErr: true,
		},
		{
			name:    "wrong count",
			bucket:  "wrongcount",
			want:    2000,
			wantErr: true,
		},
		{
			name:    "failure on final flush",
			bucket:  "wrongcountnon1000",
			want:    4321,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testChannel := make(chan s3.ObjectIdentifier, 10000)
			go func() {
				for i := 0; i < tt.want; i++ {
					testChannel <- s3.ObjectIdentifier{
						Key:       &key,
						VersionID: &version,
					}
				}
				close(testChannel)
			}()

			got, err := s3DeleteFromChannel(context.TODO(), s3svc, tt.bucket, testChannel)
			if (err != nil) != tt.wantErr {
				t.Errorf("s3DeleteFromQueue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("s3DeleteFromQueue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ====

type S3ServiceMock struct {
}

func (s S3ServiceMock) DeleteObjects(ctx context.Context, bucketName string, objects []s3.ObjectIdentifier) ([]s3.ObjectIdentifier, error) {
	switch bucketName {
	case "failbucket":
		return nil, errors.New("simulated failure")
	case "wrongcount":
		o := append(objects, s3.ObjectIdentifier{
			Key:       &key,
			VersionID: &version,
		})
		return o, nil
	case "wrongcountnon1000":
		if len(objects) != 1000 {
			return nil, errors.New("simulated failure")
		}
		return objects, nil
	}

	return objects, nil
}

func (s S3ServiceMock) GetAllBuckets(ctx context.Context) ([]s3.Bucket, error) {
	return []s3.Bucket{}, nil
}

func (s S3ServiceMock) CreateBucketSimple(ctx context.Context, bucketName string, region string, versioned bool) error {
	return nil
}

func (s S3ServiceMock) PutObjectSimple(ctx context.Context, bucketName string, keyName string, body io.Reader) (*string, *string, error) {
	return nil, nil, nil
}

func (s S3ServiceMock) GetBucektRegion(ctx context.Context, bucketName string) (string, error) {
	return "", nil
}

func (s S3ServiceMock) ListObjects(ctx context.Context, bucketName string, continuationToken *string, prefix *string) ([]string, *string, error) {
	return []string{}, nil, nil
}

func (s S3ServiceMock) ListObjectVersions(ctx context.Context, bucketName string, keyMarker *string, versionIDMarker *string, prefix *string) ([]s3.ObjectVersion, *string, *string, error) {
	return []s3.ObjectVersion{}, nil, nil, nil
}
