package main

import (
	"go-keycloack/handlers"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	app := fiber.New()

	loginHandler := &handlers.LoginHandler{}
	userCreationHandler := &handlers.UserCreationHandler{}

	app.Post("/login", loginHandler.HandleLogin)
	app.Post("/users", userCreationHandler.HandleUserCreation)

	if err := app.Listen(":3000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
