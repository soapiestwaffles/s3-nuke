package workers

import (
	"context"
	"errors"

	"github.com/soapiestwaffles/s3-nuke/pkg/aws/s3"
)

type ObjectStack struct {
	Queue []s3.ObjectIdentifier
}

func (o *ObjectStack) Push(object s3.ObjectIdentifier) {
	o.Queue = append(o.Queue, object)
}

func (o *ObjectStack) Reset() {
	o.Queue = nil
}

func (o *ObjectStack) Len() int {
	return len(o.Queue)
}

func s3DeleteFromChannel(ctx context.Context, s3svc s3.Service, bucket string, input chan s3.ObjectIdentifier) (int, error) {
	deleteCounter := 0
	objs := ObjectStack{}

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

	err := flush()
	if err != nil {
		return deleteCounter, err
	}

	return deleteCounter, nil
}
