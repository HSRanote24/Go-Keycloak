package utils

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ExtractAndValidateBearerToken extracts the Bearer token from the Authorization header and validates its format.
// Returns the token string if valid, or a Fiber error response if invalid.
func ExtractAndValidateBearerToken(c *fiber.Ctx) (string, error) {
	tokenHeader := c.Get("Authorization")
	if tokenHeader == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "Authorization token is missing in the header.")
	}
	tokenParts := strings.Split(tokenHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "Invalid Authorization token format. Ensure the token is a Bearer token.")
	}
	tokenPayload := strings.Split(tokenParts[1], ".")
	if len(tokenPayload) != 3 {
		return "", fiber.NewError(fiber.StatusUnauthorized, "Malformed JWT token. Ensure the token is correctly generated.")
	}
	return tokenParts[1], nil
}
