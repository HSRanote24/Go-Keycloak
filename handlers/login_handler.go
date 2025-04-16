package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/gofiber/fiber/v2"
)

type LoginHandler struct{}

func (h *LoginHandler) HandleLogin(c *fiber.Ctx) error {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var loginReq LoginRequest
	if err := c.BodyParser(&loginReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	keycloakBaseURL := os.Getenv("KEYCLOAK_BASE_URL")
	if len(keycloakBaseURL) > 0 && keycloakBaseURL[len(keycloakBaseURL)-1] == '/' {
		keycloakBaseURL = keycloakBaseURL[:len(keycloakBaseURL)-1]
	}

	realm := os.Getenv("REALM")
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	keycloakURL := keycloakBaseURL + "/realms/" + realm + "/protocol/openid-connect/token"

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("grant_type", "password")
	data.Set("username", loginReq.Username)
	data.Set("password", loginReq.Password)

	resp, err := http.Post(keycloakURL, "application/x-www-form-urlencoded", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to connect to Keycloak: " + err.Error(),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Keycloak error: " + string(body),
		})
	}

	tokenResponse, err := parseResponseBody(resp)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to parse response",
		})
	}

	return c.Status(fiber.StatusOK).JSON(tokenResponse)
}

func parseResponseBody(resp *http.Response) (map[string]interface{}, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsedBody map[string]interface{}
	if err := json.Unmarshal(body, &parsedBody); err != nil {
		return nil, err
	}
	return parsedBody, nil
}
