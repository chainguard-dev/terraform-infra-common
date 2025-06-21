package workqueue

import "google.golang.org/grpc/status"

// NoRetryDetails marks the error as non-retriable with the given reason.
// If this error is returned to the dispatcher, it will not requeue the key.
func NonRetriableError(err error, reason string) error {
	// NonRetriableError is a marker error that indicates that the key should not be retried.
	if err == nil {
		// No error, nothing to do.
		return ni
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

	// Return the error string
	return s.Err(reason)
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
