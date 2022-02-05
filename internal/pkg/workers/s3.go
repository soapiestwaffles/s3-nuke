package workers

import (
	"context"
	"errors"

	"github.com/soapiestwaffles/s3-nuke/pkg/aws/s3"
)

type objectStack struct {
	Queue []s3.ObjectIdentifier
}

func (o *objectStack) Push(object s3.ObjectIdentifier) {
	o.Queue = append(o.Queue, object)
}

func (o *objectStack) Reset() {
	o.Queue = nil
}

func (o *objectStack) Len() int {
	return len(o.Queue)
}

// S3DeleteFromChannel deletes object versions (s3.ObjectIdentifier) from `input` channel.
// if `progress` channel is available, it will be sent counts of deleted items
//
// returns:
//   `int` - total number of objects deleted during `input` channel lifetime
//   `error` - non-nil if errors were encountered
func S3DeleteFromChannel(ctx context.Context, s3svc s3.Service, bucket string, input chan s3.ObjectIdentifier, progress chan int) (int, error) {
	deleteCounter := 0
	objs := objectStack{}

	flush := func() error {
		queueCount := objs.Len()
		deleteResult, err := s3svc.DeleteObjects(ctx, bucket, objs.Queue)
		if err != nil {
			return err
		}

		if len(deleteResult) != queueCount {
			return errors.New("queue count does not match actual deleted count")
		}
		deleteCounter += queueCount
		if progress != nil {
			progress <- queueCount
		}
		objs.Reset()

		return nil

	}

	for object := range input {
		objs.Push(object)

		if objs.Len() == 1000 {
			err := flush()
			if err != nil {
				return deleteCounter, err
			}
		}
	}

	if objs.Len() > 0 {
		err := flush()
		if err != nil {
			return deleteCounter, err
		}
	}

	return deleteCounter, nil
}

// S3QueueObjectVersions loads s3 object versions and queues them into the `output` channel
//
// returns:
//   `int` - total number of objects queued
//   `error` - not-nil if errors were encountered while retrieving object version list
func S3QueueObjectVersions(ctx context.Context, s3svc s3.Service, bucket string, output chan s3.ObjectIdentifier) (int, error) {
	var keyMarkerState, versionMarkerState *string
	queueCounter := 0

	for {
		objectVersions, keyMarker, versionMarker, err := s3svc.ListObjectVersions(ctx, bucket, keyMarkerState, versionMarkerState, nil)
		if err != nil {
			return queueCounter, err
		}
		for _, version := range objectVersions {
			output <- version.ObjectIdentifier
			queueCounter++
		}

		if keyMarker == nil && versionMarker == nil {
			break
		}
		keyMarkerState = keyMarker
		versionMarkerState = versionMarker
	}

	return queueCounter, nil
}
