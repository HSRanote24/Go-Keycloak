package middleware

import (
	"go-keycloack/utils"

	"github.com/gofiber/fiber/v2"
)

func KeycloakAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := utils.ExtractAndValidateBearerToken(c)
		if err != nil {
			return err
		}
		// Optionally, validate token with Keycloak introspection endpoint here
		// For now, just pass through
		return c.Next()
	}
}
