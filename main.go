package main

import (
	"go-keycloack/config"
	"go-keycloack/handlers"
	"go-keycloack/middleware"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	config.InitCassandra()
	defer config.Session.Close()

	middleware.InitValkey() // Initialize Valkey (Redis-compatible) connection

	app := fiber.New()

	app.Use(middleware.RateLimitAll()) // Apply rate limiting to all routes

	userHandler := &handlers.UserHandler{}

	// Public endpoints
	app.Post("/login", userHandler.HandleLogin)
	app.Post("/users", userHandler.HandleUserCreation)

	// Protected endpoints
	app.Use(middleware.KeycloakAuthMiddleware())
	app.Get("/users", userHandler.HandleGetAllUsers)
	app.Get("/users/:id", userHandler.HandleGetUser)
	app.Put("/users/:id", userHandler.HandleUpdateUser)
	app.Delete("/users/:id", userHandler.HandleDeleteUser)

	if err := app.Listen(":3000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
