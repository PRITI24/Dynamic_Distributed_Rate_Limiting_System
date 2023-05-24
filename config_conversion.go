package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v2"
)

type Configuration struct {
	RateLimits []RateLimit `yaml:"rateLimits"`
}

type RateLimit struct {
	APIKey    string           `yaml:"apiKey"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
	mutex     sync.Mutex
}

type EndpointConfig struct {
	Path         string    `yaml:"path"`
	RPM          int       `yaml:"rpm"`
	TPM          int       `yaml:"tpm"`
	LastRequest  time.Time `yaml:"-"`
	RequestCount int       `yaml:"-"`
}

type RateLimiter struct {
	apiKeyLimits map[string]*EndpointConfig
	mutex        sync.Mutex
}

type ReservationRequest struct {
	ClientID       string `json:"clientID"`
	Tokens         int    `json:"tokens"`
	Requests       int    `json:"requests"`
	APIKey         string `json:"apiKey"`
	TargetEndpoint string `json:"targetEndpoint"`
}

var limiter *RateLimiter

func main() {
	config, err := readConfigFile("config.yaml")
	if err != nil {
		fmt.Println("Error reading configuration file:", err)
		return
	}

	limiter = NewRateLimiter(config.RateLimits)

	// Access and use the configuration data as needed
	fmt.Println("Rate Limits:", config.RateLimits)

	// Example usage
	apiKey := "API_KEY_1"
	endpointPath := "/api/endpoint1"

	// Check if the request is allowed based on the rate limits
	if limiter.AllowRequest(apiKey, endpointPath) {
		// Process the request
		fmt.Println("Request allowed")
	} else {
		fmt.Println("Request blocked")
	}

	app := fiber.New()
	app.Post("/reserve", func(c *fiber.Ctx) error {

		var request ReservationRequest

		if err := c.BodyParser(&request); err != nil {
			fmt.Println("didn't read")
			return c.SendStatus(http.StatusBadRequest)
		}

		// Extract request parameters
		clientID := request.ClientID
		tokens := request.Tokens
		requests := request.Requests
		apiKey := request.APIKey
		targetEndpoint := request.TargetEndpoint

		// Perform rate limiting and reservation logic
		reservation := limiter.Reserve(clientID, tokens, requests, apiKey, targetEndpoint)

		if reservation.Allowed {
			// Reservation successful, return 200 OK response
			return c.JSON(reservation)
		} else {
			// Reservation failed, return 429 Too Many Requests response
			return c.SendStatus(http.StatusTooManyRequests)
		}
	})

	app.Listen(":8080")
}

func readConfigFile(filePath string) (*Configuration, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Configuration
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func NewRateLimiter(rateLimits []RateLimit) *RateLimiter {
	limiter := &RateLimiter{
		apiKeyLimits: make(map[string]*EndpointConfig),
	}

	// Initialize rate limits for each API key and endpoint
	for _, rateLimit := range rateLimits {
		for _, endpoint := range rateLimit.Endpoints {
			key := fmt.Sprintf("%s-%s", rateLimit.APIKey, endpoint.Path)
			limiter.apiKeyLimits[key] = &EndpointConfig{
				Path:         endpoint.Path,
				RPM:          endpoint.RPM,
				TPM:          endpoint.TPM,
				LastRequest:  time.Now(),
				RequestCount: 0,
			}
		}
	}

	return limiter
}

func (limiter *RateLimiter) AllowRequest(apiKey, endpointPath string) bool {
	key := fmt.Sprintf("%s-%s", apiKey, endpointPath)

	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	// Retrieve the rate limits for the given API key and endpoint
	limits, ok := limiter.apiKeyLimits[key]
	if !ok {
		// API key or endpoint not found, allow the request by default
		return true
	}

	// Calculate the time elapsed since the last request
	elapsed := time.Since(limits.LastRequest)

	// Reset the request count if the elapsed time exceeds one minute
	if elapsed >= time.Minute {
		limits.RequestCount = 0
	}

	// Check if the request count exceeds the limits
	if limits.RequestCount >= limits.RPM || limits.RequestCount >= limits.TPM {
		return false
	}

	// Increment the request count and update the last request time
	limits.RequestCount++
	limits.LastRequest = time.Now()

	return true
}

func (limiter *RateLimiter) Reserve(clientID string, tokens, requests int, apiKey, endpointPath string) *Reservation {
	key := fmt.Sprintf("%s-%s", apiKey, endpointPath)

	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	// Retrieve the rate limits for the given API key and endpoint
	limits, ok := limiter.apiKeyLimits[key]
	if !ok {
		return &Reservation{
			Allowed:            false,
			ReservedTokens:     0,
			ReservedRequests:   0,
			RemainingTokens:    0,
			RemainingRequests:  0,
			TargetEndpointPath: "",
		}
	}

	// Calculate the time elapsed since the last request
	elapsed := time.Since(limits.LastRequest)

	// Reset the request count if the elapsed time exceeds one minute
	if elapsed >= time.Minute {
		limits.RequestCount = 0
	}

	// Check if the reservation exceeds the limits
	if limits.RequestCount+requests > limits.RPM || limits.RequestCount+requests > limits.TPM {
		return &Reservation{
			Allowed:            false,
			ReservedTokens:     0,
			ReservedRequests:   0,
			RemainingTokens:    limits.RPM - limits.RequestCount,
			RemainingRequests:  limits.TPM - limits.RequestCount,
			TargetEndpointPath: "",
		}
	}

	// Increment the request count and update the last request time
	limits.RequestCount += requests
	limits.LastRequest = time.Now()

	// Create the reservation
	reservation := &Reservation{
		Allowed:            true,
		ReservedTokens:     tokens,
		ReservedRequests:   requests,
		RemainingTokens:    limits.RPM - limits.RequestCount,
		RemainingRequests:  limits.TPM - limits.RequestCount,
		TargetEndpointPath: endpointPath,
	}

	// Process the reservation based on the priority class
	switch apiKey {
	case "API_KEY_1":
		// Process immediately
		go limiter.processReservation(reservation)
	case "API_KEY_2":
		// Process after a delay
		time.AfterFunc(time.Second*5, func() {
			limiter.processReservation(reservation)
		})
	case "API_KEY_3":
		// Process in the background
		go limiter.processReservationInBackground(reservation)
	default:
		// Process immediately
		limiter.processReservation(reservation)
	}

	return reservation
}

func (limiter *RateLimiter) processReservation(reservation *Reservation) {
	// Perform the desired processing based on the reservation
	fmt.Println("Processing reservation:", reservation)
}

func (limiter *RateLimiter) processReservationInBackground(reservation *Reservation) {
	// Perform the desired background processing based on the reservation
	fmt.Println("Processing reservation in background:", reservation)
}

type Reservation struct {
	Allowed            bool   `json:"allowed"`
	ReservedTokens     int    `json:"reservedTokens"`
	ReservedRequests   int    `json:"reservedRequests"`
	RemainingTokens    int    `json:"remainingTokens"`
	RemainingRequests  int    `json:"remainingRequests"`
	TargetEndpointPath string `json:"targetEndpointPath"`
}

func reserveHandler(c *fiber.Ctx) error {
	var request ReservationRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(http.StatusBadRequest).SendString("Error parsing request body")
	}

	// Extract request parameters
	clientID := request.ClientID
	tokens := request.Tokens
	requests := request.Requests
	apiKey := request.APIKey
	targetEndpoint := request.TargetEndpoint

	// Perform rate limiting and reservation logic
	reservation := limiter.Reserve(clientID, tokens, requests, apiKey, targetEndpoint)

	if reservation.Allowed {
		// Reservation successful, return 200 OK response
		return c.Status(http.StatusOK).JSON(reservation)
	} else {
		// Reservation failed, return 429 Too Many Requests response
		return c.Status(http.StatusTooManyRequests).JSON(reservation)
	}
}
