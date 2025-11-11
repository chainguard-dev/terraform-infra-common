package sdk

import (
	"net/http"

	"github.com/chainguard-dev/terraform-infra-common/pkg/httpratelimit"
)

// SecondaryRateLimitWaiter wraps the shared rate limiting transport
// for backward compatibility with existing SDK users.
//
// Deprecated: Use httpratelimit.Transport directly for new code.
type SecondaryRateLimitWaiter = httpratelimit.Transport

// NewSecondaryRateLimitWaiterClient creates a new HTTP client with rate limiting enabled.
// This function maintains backward compatibility with existing SDK users.
//
// Deprecated: Use httpratelimit.NewClient directly for new code.
func NewSecondaryRateLimitWaiterClient(base http.RoundTripper) *http.Client {
	if base == nil {
		base = http.DefaultTransport
	}

	return &http.Client{
		Transport: httpratelimit.NewTransport(base),
	}
}

// Header constants for backward compatibility.
//
// Deprecated: Use constants from httpratelimit package directly.
const (
	HeaderRetryAfter          = httpratelimit.HeaderRetryAfter
	HeaderXRateLimitReset     = httpratelimit.HeaderXRateLimitReset
	HeaderXRateLimitRemaining = httpratelimit.HeaderXRateLimitRemaining
)
