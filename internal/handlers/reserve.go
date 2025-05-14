package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/ratelimiter/internal/ratelimiter"
)

// ReserveRequest represents the incoming request structure
type ReserveRequest struct {
	ClientID       string `json:"clientID"`
	Tokens         int    `json:"tokens"`
	Requests       int    `json:"requests"`
	APIKey         string `json:"apiKey"`
	TargetEndpoint string `json:"targetEndpoint"`
}

// ReserveResponse represents the API response structure
type ReserveResponse struct {
	Status struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
	Data struct {
		Allowed            bool   `json:"allowed"`
		ReservedTokens     int    `json:"reservedTokens"`
		ReservedRequests   int    `json:"reservedRequests"`
		RemainingTokens    int    `json:"remainingTokens"`
		RemainingRequests  int    `json:"remainingRequests"`
		TargetEndpointPath string `json:"targetEndpointPath"`
	} `json:"data"`
}

// ErrorResponse represents the error response structure
type ErrorResponse struct {
	Status struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
	Error string `json:"error"`
}

// ReserveHandler handles rate limit reservation requests
type ReserveHandler struct {
	limiter *ratelimiter.RateLimiter
}

// NewReserveHandler creates a new ReserveHandler instance
func NewReserveHandler(limiter *ratelimiter.RateLimiter) *ReserveHandler {
	return &ReserveHandler{
		limiter: limiter,
	}
}

// Handle processes the reservation request
func (h *ReserveHandler) Handle(c *fiber.Ctx) error {
	var request ReserveRequest
	if err := c.BodyParser(&request); err != nil {
		errResp := ErrorResponse{}
		errResp.Status.Code = fiber.StatusBadRequest
		errResp.Status.Message = "Error"
		errResp.Error = "Invalid request format"
		return sendJSONResponse(c, fiber.StatusBadRequest, errResp)
	}

	// Validate request
	if err := h.validateRequest(&request); err != nil {
		errResp := ErrorResponse{}
		errResp.Status.Code = fiber.StatusBadRequest
		errResp.Status.Message = "Error"
		errResp.Error = err.Error()
		return sendJSONResponse(c, fiber.StatusBadRequest, errResp)
	}

	// Process reservation
	reservation := h.limiter.Reserve(
		request.ClientID,
		request.Tokens,
		request.Requests,
		request.APIKey,
		request.TargetEndpoint,
	)

	response := ReserveResponse{}
	response.Status.Code = fiber.StatusOK
	response.Status.Message = "Success"
	response.Data.Allowed = reservation.Allowed
	response.Data.ReservedTokens = reservation.ReservedTokens
	response.Data.ReservedRequests = reservation.ReservedRequests
	response.Data.RemainingTokens = reservation.RemainingTokens
	response.Data.RemainingRequests = reservation.RemainingRequests
	response.Data.TargetEndpointPath = reservation.TargetEndpointPath

	if !reservation.Allowed {
		response.Status.Code = fiber.StatusTooManyRequests
		response.Status.Message = "Rate limit exceeded"
		return sendJSONResponse(c, fiber.StatusTooManyRequests, response)
	}

	return sendJSONResponse(c, fiber.StatusOK, response)
}

// validateRequest performs basic validation on the request
func (h *ReserveHandler) validateRequest(request *ReserveRequest) error {
	if request.ClientID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "ClientID is required")
	}
	if request.APIKey == "" {
		return fiber.NewError(fiber.StatusBadRequest, "APIKey is required")
	}
	if request.TargetEndpoint == "" {
		return fiber.NewError(fiber.StatusBadRequest, "TargetEndpoint is required")
	}
	if request.Tokens < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Tokens must be non-negative")
	}
	if request.Requests < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Requests must be non-negative")
	}
	return nil
}

// sendJSONResponse sends a JSON response with proper formatting
func sendJSONResponse(c *fiber.Ctx, status int, data interface{}) error {
	c.Set("Content-Type", "application/json")

	// Convert to pretty JSON
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	return c.Status(status).Send(jsonData)
}
