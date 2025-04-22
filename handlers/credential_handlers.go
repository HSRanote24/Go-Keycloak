package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// sendJSONRequest sends a JSON payload to a given URL and returns the response body
func sendJSONRequest(targetURL string, payload map[string]interface{}) (string, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return string(body), fmt.Errorf("request failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

// OnboardIssuerFiber handles onboarding an issuer (Fiber version)
func OnboardIssuerFiber(c *fiber.Ctx) error {
	var payload map[string]interface{}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid payload")
	}

	result, err := sendJSONRequest("http://localhost:7002/onboard/issuer", payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.SendString(result)
}

// IssueCredentialFiber handles issuing a credential (Fiber version)
func IssueCredentialFiber(c *fiber.Ctx) error {
	var payload map[string]interface{}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid payload")
	}

	result, err := sendJSONRequest("http://localhost:7002/openid4vc/jwt/issue", payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	offerURL, err := parseOfferURL(result)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Invalid offer URL: %v", err))
	}

	resp, err := http.Get(offerURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to fetch credential offer: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read offer body")
	}

	if resp.StatusCode != http.StatusOK {
		return c.Status(fiber.StatusUnauthorized).SendString(fmt.Sprintf("Credential fetch failed: status %d, body: %s", resp.StatusCode, string(body)))
	}

	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

// VerifyCredentialFiber handles verifying a credential (Fiber version)
func VerifyCredentialFiber(c *fiber.Ctx) error {
	var payload map[string]interface{}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid payload")
	}

	result, err := sendJSONRequest("http://localhost:7003/openid4vc/verify", payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.SendString(result)
}

// parseOfferURL parses the raw result from credential issuance to extract the valid offer URL.
func parseOfferURL(raw string) (string, error) {
	const prefix = "openid-credential-offer://"
	if !strings.HasPrefix(raw, prefix) {
		return "", fmt.Errorf("invalid credential offer URI format")
	}
	uriPart := strings.TrimPrefix(raw, prefix)
	u, err := url.Parse("http://" + uriPart) // Add 'http://' for parsing
	if err != nil {
		return "", fmt.Errorf("invalid URI: %w", err)
	}

	// Replace 'host.docker.internal' with 'localhost' if it exists
	if strings.Contains(u.Host, "host.docker.internal") {
		u.Host = strings.Replace(u.Host, "host.docker.internal", "localhost", 1)
	}

	query := u.Query().Get("credential_offer_uri")
	if query == "" {
		return "", fmt.Errorf("missing credential_offer_uri")
	}

	parsedQueryURL, err := url.Parse(query)
	if err != nil {
		return "", fmt.Errorf("invalid credential offer URL: %w", err)
	}

	if strings.Contains(parsedQueryURL.Host, "host.docker.internal") {
		parsedQueryURL.Host = strings.Replace(parsedQueryURL.Host, "host.docker.internal", "localhost", 1)
	}

	return parsedQueryURL.String(), nil
}
