package workers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/soapiestwaffles/s3-nuke/pkg/aws/s3"
)

var (
	key     = "test"
	version = "version"
	s3svc   = S3ServiceMock{}
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
			o := &objectStack{}
			o.Push(tt.args.object)

			if o.Queue[0].Key != &key || o.Queue[0].VersionID != &version {
				t.Errorf("ObjectStack.Push() push failed, queue did not contain correct item")
			}
		})
	}
}

func TestObjectStack_Reset(t *testing.T) {
	t.Run("reset", func(t *testing.T) {
		o := &objectStack{}
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
			o := &objectStack{}
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

func Test_objectStack_FindMissingFrom(t *testing.T) {
	q := make([]s3.ObjectIdentifier, 1000)
	z := make([]s3.ObjectIdentifier, 1000)
	x := make([]s3.ObjectIdentifier, 970)
	for i := 0; i < 1000; i++ {
		q[i] = s3.ObjectIdentifier{
			Key:       ptrString(fmt.Sprintf("key%d", i)),
			VersionID: ptrString(fmt.Sprintf("version%d", i)),
		}
		z[i] = s3.ObjectIdentifier{
			Key:       ptrString(fmt.Sprintf("key%d", i)),
			VersionID: ptrString(fmt.Sprintf("version%d", i)),
		}
	}

	wasRemoved := make([]s3.ObjectIdentifier, 30)
	randomIndicies := generateUniqueRandoms(30, 1000)
	for i := 0; i < 30; i++ {
		wasRemoved[i] = s3.ObjectIdentifier{
			Key:       ptrString(fmt.Sprintf("key%d", randomIndicies[i])),
			VersionID: ptrString(fmt.Sprintf("version%d", randomIndicies[i])),
		}
	}
	sort.Slice(wasRemoved, func(i, j int) bool {
		iKey, jKey := (*wasRemoved[i].Key)[3:], (*wasRemoved[j].Key)[3:]
		iVal, _ := strconv.Atoi(iKey)
		jVal, _ := strconv.Atoi(jKey)
		return iVal < jVal
	})

	addIndex := 0
GENERATEOUTER:
	for i := 0; i < 1000; i++ {
		for z := 0; z < 30; z++ {
			if i == randomIndicies[z] {
				continue GENERATEOUTER
			}
		}
		x[addIndex] = s3.ObjectIdentifier{
			Key:       ptrString(fmt.Sprintf("key%d", i)),
			VersionID: ptrString(fmt.Sprintf("version%d", i)),
		}
		addIndex++
	}

	type args struct {
		objects []s3.ObjectIdentifier
	}
	tests := []struct {
		name string
		args args
		want []s3.ObjectIdentifier
	}{
		{
			name: "all missing",
			args: args{
				objects: []s3.ObjectIdentifier{},
			},
			want: z,
		},
		{
			name: "none missing",
			args: args{
				objects: z,
			},
			want: []s3.ObjectIdentifier{},
		},
		{
			name: "random missing",
			args: args{
				objects: x,
			},
			want: wasRemoved,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := objectStack{
				Queue: q,
			}
			if got := o.FindMissingFrom(tt.args.objects); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("objectStack.FindMissingFrom() = got result (%d), want (%d)", len(got), len(tt.want))
				t.Errorf("=== GOT ===")
				for _, v := range got {
					t.Errorf("--- Key: %s, VersionID: %s\n", *v.Key, *v.VersionID)
				}
				t.Errorf("=== WANT ===")
				for _, v := range tt.want {
					t.Errorf("--- Key: %s, VersionID: %s\n", *v.Key, *v.VersionID)
				}
				return
			}
		})
	}
}

func Test_s3DeleteFromChannel(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	progress := make(chan int, 100)
	failures := make(chan []s3.ObjectIdentifier, 100)
	tests := []struct {
		name         string
		bucket       string
		progressChan chan int
		failuresChan chan []s3.ObjectIdentifier
		want         int
		wantErr      bool
	}{
		{
			name:         "sub-1000 count",
			bucket:       "testbucket",
			want:         500,
			progressChan: nil,
			failuresChan: nil,
			wantErr:      false,
		},
		{
			name:         "1000 count",
			bucket:       "testbucket",
			want:         1000,
			progressChan: nil,
			failuresChan: nil,
			wantErr:      false,
		},
		{
			name:         "5000 count",
			bucket:       "testbucket",
			want:         5000,
			progressChan: nil,
			failuresChan: nil,
			wantErr:      false,
		},
		{
			name:         "random count 1",
			bucket:       "testbucket",
			want:         rand.Intn(9000),
			progressChan: nil,
			failuresChan: nil,
			wantErr:      false,
		},
		{
			name:         "random count 2",
			bucket:       "testbucket",
			want:         rand.Intn(9000),
			progressChan: nil,
			failuresChan: nil,
			wantErr:      false,
		},
		{
			name:         "random count 3",
			bucket:       "testbucket",
			want:         rand.Intn(9000),
			progressChan: nil,
			failuresChan: nil,
			wantErr:      false,
		},
		{
			name:         "random count 4",
			bucket:       "testbucket",
			want:         rand.Intn(9000),
			progressChan: nil,
			failuresChan: nil,
			wantErr:      false,
		},
		{
			name:         "random count 5",
			bucket:       "testbucket",
			want:         rand.Intn(9000),
			progressChan: nil,
			failuresChan: nil,
			wantErr:      false,
		},
		{
			name:         "failure",
			bucket:       "failbucket",
			want:         2000,
			progressChan: nil,
			failuresChan: nil,
			wantErr:      true,
		},
		{
			name:         "failure on final flush",
			bucket:       "wrongcountnon1000",
			want:         4321,
			progressChan: nil,
			failuresChan: nil,
			wantErr:      true,
		},
		{
			name:         "progress channel",
			bucket:       "testbucket",
			want:         1234,
			progressChan: progress,
			failuresChan: nil,
			wantErr:      false,
		},
		{
			name:         "failure channel single",
			bucket:       "failurechanbucket",
			want:         50,
			progressChan: nil,
			failuresChan: failures,
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testChannel := make(chan s3.ObjectIdentifier, 10000)
			go func() {
				for i := 0; i < tt.want; i++ {
					testChannel <- s3.ObjectIdentifier{
						Key:       ptrString(fmt.Sprintf("key%d", i)),
						VersionID: ptrString(fmt.Sprintf("version%d", i)),
					}
				}
				close(testChannel)
			}()

			got, err := S3DeleteFromChannel(context.TODO(), s3svc, tt.bucket, testChannel, tt.progressChan, tt.failuresChan)
			if (err != nil) != tt.wantErr {
				t.Errorf("s3DeleteFromQueue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if tt.failuresChan != nil && tt.bucket == "failurechanbucket" {
				if got != (tt.want - 5) {
					t.Errorf("s3DeleteFromQueue() failed result for failures test, want %d, got %d", tt.want-5, got)
					return
				}
				close(tt.failuresChan)
				result := []s3.ObjectIdentifier{}
				for failures := range tt.failuresChan {
					result = append(result, failures...)
				}

				if len(result) != 5 {
					t.Errorf("s3DeleteFromQueue() failuresChan want %d, got %d", 5, len(result))
					return
				}

				for i := 0; i < 5; i++ {
					currentKey := fmt.Sprintf("key%d", i)
					currentVersion := fmt.Sprintf("version%d", i)
					if *result[i].Key != currentKey && *result[i].VersionID != currentVersion {
						t.Errorf("s3DeleteFromQueue() failuresChan found incorrect result in failures chan, want %s/%s, got %s/%s", currentKey, currentVersion, *result[i].Key, *result[i].VersionID)
						return
					}
				}

				return
			}

			if got != tt.want {
				t.Errorf("s3DeleteFromQueue() = %v, want %v", got, tt.want)
				return
			}

			if tt.progressChan != nil {
				close(tt.progressChan)
				progressCount := 0
				for deleted := range tt.progressChan {
					progressCount += deleted
				}
				if progressCount != tt.want {
					t.Errorf("s3DeleteFromQueue() progressChan want %d, got %d", tt.want, progressCount)
					return
				}
			}
		})
	}
}

func TestS3QueueObjectVersions(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	type args struct {
		bucket string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "list object versions",
			args: args{
				bucket: "randombucket",
			},
			wantErr: false,
		},
		{
			name: "failure on list object versions",
			args: args{
				bucket: "failbucket",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := make(chan s3.ObjectIdentifier, 5000)
			count, err := S3QueueObjectVersions(context.TODO(), s3svc, tt.args.bucket, output)
			if (err != nil) != tt.wantErr {
				t.Errorf("S3QueueObjectVersions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			t.Logf("S3QueueObjectVersions() got %d objects", count)
			close(output)

			objCount := 0
			for range output {
				objCount++
			}
			t.Logf("S3QueueObjectVersions() actual object count: %d", objCount)
			if objCount != count {
				t.Errorf("S3QueueObjectVersions() error channel object count and returned count do not match")
				return
			}

			if objCount != 4000 {
				t.Errorf("S3QueueObjectVersions() did not get expected amount of objects")
				return
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
	case "failurechanbucket":
		result := []s3.ObjectIdentifier{}
		result = append(result, objects[5:]...)
		return result, nil
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
	if bucketName == "failbucket" {
		return nil, nil, nil, errors.New("simulated failure")
	}

	keyMarkerStates := []string{
		"firstKey",
		"secondKey",
		"thirdKey",
	}

	versionMarkerStates := []string{
		"firstVersion",
		"secondVersion",
		"thirdVersion",
	}

	o := []s3.ObjectVersion{}
	for i := 0; i < 1000; i++ {
		o = append(o, s3.ObjectVersion{
			ObjectIdentifier: s3.ObjectIdentifier{
				Key:       &key,
				VersionID: &version,
			},
			IsDeleteMarker: false,
		})
	}

	if keyMarker == nil {
		return o, &keyMarkerStates[0], &versionMarkerStates[0], nil
	}
	switch *keyMarker {
	case keyMarkerStates[0]:
		return o, &keyMarkerStates[1], &versionMarkerStates[1], nil
	case keyMarkerStates[1]:
		return o, &keyMarkerStates[2], &versionMarkerStates[2], nil
	case keyMarkerStates[2]:
		return o, nil, nil, nil
	default:
		return o, &keyMarkerStates[0], &versionMarkerStates[0], nil
	}

}

func ptrString(s string) *string {
	return &s
}

func generateUniqueRandoms(size int, max int) []int {
	rand.Seed(time.Now().UnixNano())
	num := make([]int, size)
	for i := 0; i < size; i++ {
	WHILEFOR:
		for {
			n := rand.Intn(max)
			for x := 0; x < len(num); x++ {
				if num[x] == n {
					continue WHILEFOR
				}
			}
			num[i] = n
			break
		}
	}

	return num
}
