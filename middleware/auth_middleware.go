package middleware

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func KeycloakAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token == "" || !strings.HasPrefix(token, "Bearer ") {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or invalid token"})
		}
		// Optionally, validate token with Keycloak introspection endpoint here
		// For now, just pass through
		return c.Next()
	}
}
