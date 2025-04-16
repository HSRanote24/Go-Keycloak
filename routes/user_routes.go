package routes

import (
	"go-keycloack/handlers"

	"github.com/gofiber/fiber/v2"
)

func RegisterUserRoutes(app *fiber.App, userHandler *handlers.UserHandler) {
	user := app.Group("/user")
	user.Post("/login", userHandler.HandleLogin)
	user.Post("/create", userHandler.HandleUserCreation)
	user.Get(":id", userHandler.HandleGetUser)
	user.Put(":id", userHandler.HandleUpdateUser)
	user.Delete(":id", userHandler.HandleDeleteUser)
}
