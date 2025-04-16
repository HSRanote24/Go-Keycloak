package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"

	"go-keycloack/models"
	"go-keycloack/services"

	"github.com/go-playground/validator/v10"
	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	// Add Cassandra and Keycloak clients here when integrating
}

func (h *UserHandler) HandleLogin(c *fiber.Ctx) error {
	type LoginRequest struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		FirstName string `json:"firstname"`
		LastName  string `json:"lastname"`
	}
	var loginReq LoginRequest
	if err := c.BodyParser(&loginReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	keycloakBaseURL := os.Getenv("KEYCLOAK_BASE_URL")
	if keycloakBaseURL == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "KEYCLOAK_BASE_URL not set"})
	}
	if keycloakBaseURL[len(keycloakBaseURL)-1] == '/' {
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to connect to Keycloak: " + err.Error()})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Keycloak error: " + string(body)})
	}

	var tokenResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to parse Keycloak response"})
	}

	// Ensure user exists in Cassandra (create if not)
	user, err := services.GetUserByUsername(loginReq.Username)
	if err != nil || user == nil {
		// Create user in Cassandra
		user = &models.User{Username: loginReq.Username, FirstName: loginReq.FirstName, LastName: loginReq.LastName}
		if err := services.CreateUser(user); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user in Cassandra"})
		}
	}

	return c.JSON(tokenResponse)
}

func (h *UserHandler) HandleUserCreation(c *fiber.Ctx) error {
	type UserCreationRequest struct {
		Username  string `json:"username" validate:"required,min=3,max=32"`
		Password  string `json:"password" validate:"required,min=6"`
		Email     string `json:"email" validate:"required,email"`
		FirstName string `json:"firstname" validate:"required,min=1,max=50"`
		LastName  string `json:"lastname" validate:"required,min=1,max=50"`
	}
	var userReq UserCreationRequest
	if err := c.BodyParser(&userReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	// Validate input
	validate := validator.New()
	if err := validate.Struct(userReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Get admin token for Keycloak
	adminToken, err := getKeycloakAdminToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get admin token: " + err.Error()})
	}

	keycloakBaseURL := os.Getenv("KEYCLOAK_BASE_URL")
	realm := os.Getenv("REALM")
	usersURL := keycloakBaseURL + "/admin/realms/" + realm + "/users"

	userBody := map[string]interface{}{
		"username":  userReq.Username,
		"email":     userReq.Email,
		"enabled":   true,
		"firstName": userReq.FirstName,
		"lastName":  userReq.LastName,
		"credentials": []map[string]interface{}{
			{"type": "password", "value": userReq.Password, "temporary": false},
		},
	}
	userJSON, _ := json.Marshal(userBody)

	req, err := http.NewRequest("POST", usersURL, bytes.NewBuffer(userJSON))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user creation request"})
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user in Keycloak"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to create user in Keycloak: " + string(body)})
	}

	// Create user in Cassandra
	user := &models.User{Username: userReq.Username, Email: userReq.Email, FirstName: userReq.FirstName, LastName: userReq.LastName}
	if err := services.CreateUser(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User created in Keycloak but failed in Cassandra"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "User registered successfully"})
}

// Helper to get Keycloak admin token
func getKeycloakAdminToken() (string, error) {
	keycloakBaseURL := os.Getenv("KEYCLOAK_BASE_URL")
	realm := os.Getenv("REALM")
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	adminUser := os.Getenv("ADMIN_USERNAME")
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if keycloakBaseURL == "" || realm == "" || clientID == "" || clientSecret == "" || adminUser == "" || adminPass == "" {
		return "", fiber.NewError(fiber.StatusInternalServerError, "Missing Keycloak admin credentials")
	}
	tokenURL := keycloakBaseURL + "/realms/" + realm + "/protocol/openid-connect/token"
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("grant_type", "password")
	data.Set("username", adminUser)
	data.Set("password", adminPass)
	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fiber.NewError(fiber.StatusUnauthorized, string(body))
	}
	var tokenResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	token, ok := tokenResp["access_token"].(string)
	if !ok {
		return "", fiber.NewError(fiber.StatusInternalServerError, "No access_token in response")
	}
	return token, nil
}

func (h *UserHandler) HandleGetUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := gocql.ParseUUID(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid UUID"})
	}
	user, err := services.GetUserByID(id)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}
	return c.JSON(user)
}

func (h *UserHandler) HandleUpdateUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := gocql.ParseUUID(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid UUID"})
	}
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}
	if err := services.UpdateUser(id, &user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Update failed"})
	}
	return c.JSON(user)
}

func (h *UserHandler) HandleDeleteUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := gocql.ParseUUID(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid UUID"})
	}

	// Check if user exists first
	user, err := services.GetUserByID(id)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	if err := services.DeleteUser(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Delete failed"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// HandleGetAllUsers returns all users from the database
func (h *UserHandler) HandleGetAllUsers(c *fiber.Ctx) error {
	users, err := services.GetAllUsers()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch users"})
	}
	return c.JSON(users)
}
