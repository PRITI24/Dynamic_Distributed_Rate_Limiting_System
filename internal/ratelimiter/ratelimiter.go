package ratelimiter

import (
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/ratelimiter/internal/config"
)

// RateLimiter handles rate limiting logic
type RateLimiter struct {
	apiKeyLimits map[string]*EndpointState
	mutex        sync.RWMutex
}

// EndpointState tracks the state of an endpoint
type EndpointState struct {
	Path         string
	RPM          int
	TPM          int
	LastRequest  time.Time
	RequestCount int
	mutex        sync.Mutex
}

// Reservation represents a rate limit reservation response
type Reservation struct {
	Allowed            bool   `json:"allowed"`
	ReservedTokens     int    `json:"reservedTokens"`
	ReservedRequests   int    `json:"reservedRequests"`
	RemainingTokens    int    `json:"remainingTokens"`
	RemainingRequests  int    `json:"remainingRequests"`
	TargetEndpointPath string `json:"targetEndpointPath"`
}

// New creates a new RateLimiter instance
func New(rateLimits []config.RateLimit) *RateLimiter {
	limiter := &RateLimiter{
		apiKeyLimits: make(map[string]*EndpointState),
	}

	for _, rateLimit := range rateLimits {
		for _, endpoint := range rateLimit.Endpoints {
			key := fmt.Sprintf("%s-%s", rateLimit.APIKey, endpoint.Path)
			limiter.apiKeyLimits[key] = &EndpointState{
				Path:         endpoint.Path,
				RPM:         endpoint.RPM,
				TPM:         endpoint.TPM,
				LastRequest: time.Now(),
			}
		}
	}

	return limiter
}

// Reserve attempts to reserve capacity for requests and tokens
func (rl *RateLimiter) Reserve(clientID string, tokens, requests int, apiKey, targetEndpoint string) *Reservation {
	key := fmt.Sprintf("%s-%s", apiKey, targetEndpoint)

	rl.mutex.RLock()
	state, exists := rl.apiKeyLimits[key]
	rl.mutex.RUnlock()

	if !exists {
		return &Reservation{
			Allowed: false,
		}
	}

	state.mutex.Lock()
	defer state.mutex.Unlock()

	// Reset counters if a minute has passed
	if time.Since(state.LastRequest) >= time.Minute {
		state.RequestCount = 0
	}

	// Check if the reservation would exceed limits
	if state.RequestCount+requests > state.RPM || tokens > state.TPM {
		return &Reservation{
			Allowed:           false,
			RemainingTokens:   state.TPM - tokens,
			RemainingRequests: state.RPM - state.RequestCount,
		}
	}

	// Update state
	state.RequestCount += requests
	state.LastRequest = time.Now()

	// Process based on priority
	reservation := &Reservation{
		Allowed:            true,
		ReservedTokens:     tokens,
		ReservedRequests:   requests,
		RemainingTokens:    state.TPM - tokens,
		RemainingRequests:  state.RPM - state.RequestCount,
		TargetEndpointPath: targetEndpoint,
	}

	go rl.processReservation(apiKey, reservation)

	return reservation
}

// processReservation handles the reservation based on API key priority
func (rl *RateLimiter) processReservation(apiKey string, reservation *Reservation) {
	switch apiKey {
	case "API_KEY_1":
		// Process immediately
		rl.process(reservation)
	case "API_KEY_2":
		// Process after delay
		time.Sleep(5 * time.Second)
		rl.process(reservation)
	case "API_KEY_3":
		// Process in background
		go rl.process(reservation)
	default:
		// Process immediately
		rl.process(reservation)
	}
}

// process handles the actual processing of the reservation
func (rl *RateLimiter) process(reservation *Reservation) {
	// Implement actual processing logic here
	// This could include making API calls, processing data, etc.
	fmt.Printf("Processing reservation for endpoint: %s\n", reservation.TargetEndpointPath)
} 