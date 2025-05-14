package ratelimiter

import (
	"testing"
	"time"

	"github.com/yourusername/ratelimiter/internal/config"
)

func TestRateLimiter_Reserve(t *testing.T) {
	// Test configuration
	rateLimits := []config.RateLimit{
		{
			APIKey: "API_KEY_1",
			Endpoints: []config.EndpointConfig{
				{
					Path: "/test",
					RPM:  10,
					TPM:  100,
				},
			},
		},
	}

	limiter := New(rateLimits)

	tests := []struct {
		name           string
		clientID       string
		tokens         int
		requests       int
		apiKey         string
		targetEndpoint string
		wantAllowed    bool
	}{
		{
			name:           "Valid request within limits",
			clientID:       "client1",
			tokens:         50,
			requests:       5,
			apiKey:         "API_KEY_1",
			targetEndpoint: "/test",
			wantAllowed:    true,
		},
		{
			name:           "Request exceeds RPM",
			clientID:       "client1",
			tokens:         50,
			requests:       11,
			apiKey:         "API_KEY_1",
			targetEndpoint: "/test",
			wantAllowed:    false,
		},
		{
			name:           "Request exceeds TPM",
			clientID:       "client1",
			tokens:         101,
			requests:       5,
			apiKey:         "API_KEY_1",
			targetEndpoint: "/test",
			wantAllowed:    false,
		},
		{
			name:           "Invalid API key",
			clientID:       "client1",
			tokens:         50,
			requests:       5,
			apiKey:         "INVALID_KEY",
			targetEndpoint: "/test",
			wantAllowed:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reservation := limiter.Reserve(
				tt.clientID,
				tt.tokens,
				tt.requests,
				tt.apiKey,
				tt.targetEndpoint,
			)

			if reservation.Allowed != tt.wantAllowed {
				t.Errorf("Reserve() allowed = %v, want %v", reservation.Allowed, tt.wantAllowed)
			}

			// Add a small delay to avoid rate limit conflicts between tests
			time.Sleep(100 * time.Millisecond)
		})
	}
}

func TestRateLimiter_ResetAfterOneMinute(t *testing.T) {
	rateLimits := []config.RateLimit{
		{
			APIKey: "API_KEY_1",
			Endpoints: []config.EndpointConfig{
				{
					Path: "/test",
					RPM:  10,
					TPM:  100,
				},
			},
		},
	}

	limiter := New(rateLimits)

	// Make initial requests
	for i := 0; i < 5; i++ {
		reservation := limiter.Reserve("client1", 10, 1, "API_KEY_1", "/test")
		if !reservation.Allowed {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}

	// Simulate time passing (> 1 minute)
	key := "API_KEY_1-/test"
	state := limiter.apiKeyLimits[key]
	state.LastRequest = time.Now().Add(-2 * time.Minute)

	// Make another request
	reservation := limiter.Reserve("client1", 10, 1, "API_KEY_1", "/test")
	if !reservation.Allowed {
		t.Error("Expected request to be allowed after counter reset")
	}

	if reservation.RemainingRequests != 9 {
		t.Errorf("Expected 9 remaining requests, got %d", reservation.RemainingRequests)
	}
}
