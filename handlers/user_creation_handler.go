package handlers

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
)

type UserCreationHandler struct{}

type UserCreationRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func (h *UserCreationHandler) HandleUserCreation(c *fiber.Ctx) error {
	var userReq UserCreationRequest
	if err := c.BodyParser(&userReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload: " + err.Error(),
		})
	}

	adminToken := c.Get("Authorization")
	if adminToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization token is missing in the header.",
		})
	}

	tokenParts := strings.Split(adminToken, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Authorization token format. Ensure the token is a Bearer token.",
		})
	}

	tokenPayload := strings.Split(tokenParts[1], ".")
	if len(tokenPayload) != 3 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Malformed JWT token. Ensure the token is correctly generated.",
		})
	}

	decodedPayload, err := decodeBase64(tokenPayload[1])
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Error decoding token payload: " + err.Error(),
		})
	}

	var tokenData map[string]interface{}
	if err := json.Unmarshal([]byte(decodedPayload), &tokenData); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Error parsing token payload: " + err.Error(),
		})
	}

	roles := []interface{}{}
	if ra, ok := tokenData["realm_access"].(map[string]interface{}); ok {
		if r, ok := ra["roles"].([]interface{}); ok {
			roles = append(roles, r...)
		}
	}
	if resAccess, ok := tokenData["resource_access"].(map[string]interface{}); ok {
		if realmMgmt, ok := resAccess["realm-management"].(map[string]interface{}); ok {
			if r, ok := realmMgmt["roles"].([]interface{}); ok {
				roles = append(roles, r...)
			}
		}
	}

	hasManageUsersRole := false
	for _, role := range roles {
		if role == "manage-users" {
			hasManageUsersRole = true
			break
		}
	}

	if !hasManageUsersRole {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Authorization error: Token does not have the 'manage-users' role.",
		})
	}

	client := resty.New()
	keycloakBaseURL := os.Getenv("KEYCLOAK_BASE_URL")
	realm := os.Getenv("REALM")

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", adminToken).
		SetBody(map[string]interface{}{
			"username":      userReq.Username,
			"firstName":     userReq.FirstName,
			"lastName":      userReq.LastName,
			"email":         userReq.Email,
			"enabled":       true,
			"emailVerified": true,
			"credentials": []map[string]interface{}{
				{
					"type":      "password",
					"value":     userReq.Password,
					"temporary": false,
				},
			},
		}).
		Post(keycloakBaseURL + "/admin/realms/" + realm + "/users")

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to connect to Keycloak: " + err.Error(),
		})
	}

	if resp.IsError() {
		if resp.StatusCode() == fiber.StatusForbidden {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Keycloak authorization error: insufficient permissions to create users.",
			})
		}
		return c.Status(resp.StatusCode()).JSON(fiber.Map{
			"error": "Keycloak error: " + resp.String(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User created successfully",
	})
}

func decodeBase64(encoded string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
