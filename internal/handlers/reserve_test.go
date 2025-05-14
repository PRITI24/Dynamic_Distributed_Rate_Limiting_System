package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/ratelimiter/internal/config"
	"github.com/yourusername/ratelimiter/internal/ratelimiter"
)

func TestReserveHandler_Handle(t *testing.T) {
	// Test configuration
	rateLimits := []config.RateLimit{
		{
			APIKey: "API_KEY_1",
			Endpoints: []config.EndpointConfig{
				{
					Path: "/api/endpoint1",
					RPM:  100,
					TPM:  10,
				},
			},
		},
	}

	// Initialize components
	limiter := ratelimiter.New(rateLimits)
	handler := NewReserveHandler(limiter)
	app := fiber.New()
	app.Post("/reserve", handler.Handle)

	tests := []struct {
		name           string
		request        ReserveRequest
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "Valid request",
			request: ReserveRequest{
				ClientID:       "test-client",
				Tokens:         5,
				Requests:       1,
				APIKey:         "API_KEY_1",
				TargetEndpoint: "/api/endpoint1",
			},
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response struct {
					Allowed bool `json:"allowed"`
				}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if !response.Allowed {
					t.Error("Expected request to be allowed")
				}
			},
		},
		{
			name: "Missing client ID",
			request: ReserveRequest{
				Tokens:         5,
				Requests:       1,
				APIKey:         "API_KEY_1",
				TargetEndpoint: "/api/endpoint1",
			},
			expectedStatus: fiber.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Error != "ClientID is required" {
					t.Errorf("Expected error message about missing ClientID, got: %s", response.Error)
				}
			},
		},
		{
			name: "Invalid API key",
			request: ReserveRequest{
				ClientID:       "test-client",
				Tokens:         5,
				Requests:       1,
				APIKey:         "INVALID_KEY",
				TargetEndpoint: "/api/endpoint1",
			},
			expectedStatus: fiber.StatusTooManyRequests,
			checkResponse: func(t *testing.T, body []byte) {
				var response struct {
					Allowed bool `json:"allowed"`
				}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Allowed {
					t.Error("Expected request to be denied")
				}
			},
		},
		{
			name: "Exceed RPM limit",
			request: ReserveRequest{
				ClientID:       "test-client",
				Tokens:         5,
				Requests:       150,
				APIKey:         "API_KEY_1",
				TargetEndpoint: "/api/endpoint1",
			},
			expectedStatus: fiber.StatusTooManyRequests,
			checkResponse: func(t *testing.T, body []byte) {
				var response struct {
					Allowed bool `json:"allowed"`
				}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Allowed {
					t.Error("Expected request to be denied")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			reqBody, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Create test request
			req := httptest.NewRequest("POST", "/reserve", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Perform request
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}

			// Check status code
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			// Check response
			tt.checkResponse(t, body)
		})
	}
}
