package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/ratelimiter/internal/config"
	"github.com/yourusername/ratelimiter/internal/handlers"
	"github.com/yourusername/ratelimiter/internal/ratelimiter"
)

func main() {
	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Initialize rate limiter
	limiter := ratelimiter.New(cfg.RateLimits)

	// Initialize Fiber app
	app := fiber.New()

	// Initialize handlers
	handler := handlers.NewReserveHandler(limiter)

	// Setup routes
	app.Post("/reserve", handler.Handle)

	// Start server
	port := ":8086"
	fmt.Printf("Server starting on port %s\n", port)
	if err := app.Listen(port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
