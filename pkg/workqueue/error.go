package workqueue

import (
	"errors"
	"time"

	"google.golang.org/grpc/status"
)

// NoRetryDetails marks the error as non-retriable with the given reason.
// If this error is returned to the dispatcher, it will not requeue the key.
func NonRetriableError(err error, reason string) error {
	// NonRetriableError is a marker error that indicates that the key should not be retried.
	if err == nil {
		// No error, nothing to do.
		return nil
	}

	// We ignore ok - usually happens when the error is not a gRPC error.
	s, _ := status.FromError(err)
	s, derr := s.WithDetails(&NoRetryDetails{
		Message: reason,
	})
	if derr != nil {
		// This shouldn't generally happen since this should only happen if the details aren't a protobuf message,
		// but if it does, we return the original error.
		return err
	}

	return s.Err()
}

// GetNonRetriableDetails extracts the NoRetryDetails from the error if it exists.
// If the error is nil or does not contain NoRetryDetails, it returns nil.
func GetNonRetriableDetails(err error) *NoRetryDetails {
	if err == nil {
		return nil
	}

	s, ok := status.FromError(err)
	if !ok {
		return nil
	}

	for _, detail := range s.Details() {
		if nrd, ok := detail.(*NoRetryDetails); ok {
			return nrd
		}
	}
	return nil
}

// requeueError is a special error type that indicates the work item should be
// requeued with a specific delay.
type requeueError struct {
	delay   time.Duration
	isError bool
}

// Error implements the error interface.
func (e *requeueError) Error() string {
	if e.isError {
		return "requeue requested (error)"
	}
	return "requeue requested (polling)"
}

// RequeueAfter returns an error that indicates the work item should be requeued
// after the specified delay for normal polling scenarios.
// Use RetryAfter for error/retry scenarios.
func RequeueAfter(delay time.Duration) error {
	return &requeueError{
		delay:   delay,
		isError: false,
	}
}

// RetryAfter returns an error that indicates the work item should be retried
// after the specified delay due to an error condition requiring retry with backoff.
// Use RequeueAfter for normal polling scenarios.
func RetryAfter(delay time.Duration) error {
	return &requeueError{
		delay:   delay,
		isError: true,
	}
}

// GetRequeueDelay extracts the requeue delay from an error if it's a requeue error.
// Returns the delay, whether it's a requeue error, and whether it's an error scenario (vs polling).
// If the error is not a requeue error, returns (0, false, false).
func GetRequeueDelay(err error) (time.Duration, bool, bool) {
	if err == nil {
		return 0, false, false
	}
	var re *requeueError
	if errors.As(err, &re) {
		return re.delay, true, re.isError
	}
	return 0, false, false
}
