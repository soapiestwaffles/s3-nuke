package workers

import (
	"context"
	"reflect"

	"github.com/soapiestwaffles/s3-nuke/pkg/aws/s3"
)

type objectStack struct {
	Queue []s3.ObjectIdentifier
}

// Push adds an s3.ObjectIdentifier to the stack
func (o *objectStack) Push(object s3.ObjectIdentifier) {
	o.Queue = append(o.Queue, object)
}

// Reset resets the object stack
func (o *objectStack) Reset() {
	o.Queue = nil
}

// Len returns the current size of the stack
func (o *objectStack) Len() int {
	return len(o.Queue)
}

// FindMissingFrom will compare the two slices of ObjectIdentifier and return the difference of the two
func (o *objectStack) FindMissingFrom(objects []s3.ObjectIdentifier) []s3.ObjectIdentifier {
	result := make([]s3.ObjectIdentifier, 0)
OUTER:
	for i := 0; i < len(o.Queue); i++ {
		for x := 0; x < len(objects); x++ {
			if reflect.DeepEqual(o.Queue[i], objects[x]) {
				continue OUTER
			}
		}

		result = append(result, o.Queue[i])
	}

	return result
}

// S3DeleteFromChannel deletes object versions (s3.ObjectIdentifier) from `input` channel.
// if `progress` channel is available, it will be sent counts of deleted items
//
// returns:
//   `int` - total number of objects deleted during `input` channel lifetime
//   `[]s3.ObjectIdentifier` - object list of items that were queued but didn't get deleted via s3svc.DeleteObjects
//   `error` - non-nil if errors were encountered
func S3DeleteFromChannel(ctx context.Context, s3svc s3.Service, bucket string, input <-chan s3.ObjectIdentifier, progress chan<- int, failures chan<- []s3.ObjectIdentifier) (int, error) {
	deleteCounter := 0
	objs := objectStack{}

	// returns
	// `[]s3.ObjectIdentifier`` - objects that were queued but didn't actually get deleted
	// `error`` - error message if an unrecoverable error occured
	flush := func() error {
		queueCount := objs.Len()
		deleteResult, err := s3svc.DeleteObjects(ctx, bucket, objs.Queue)
		if err != nil {
			return err
		}
		deleteCount := len(deleteResult)

		deleteCounter += deleteCount
		if progress != nil {
			progress <- deleteCount
		}
		if deleteCount != queueCount && failures != nil {
			deleteFailures := objs.FindMissingFrom(deleteResult)
			failures <- deleteFailures
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
func S3QueueObjectVersions(ctx context.Context, s3svc s3.Service, bucket string, output chan<- s3.ObjectIdentifier) (int, error) {
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
