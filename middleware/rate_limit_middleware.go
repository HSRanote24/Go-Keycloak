package middleware

import (
	"context"
	"fmt"
	"time"

	"go-keycloack/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

func InitValkey() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	info, err := redisClient.Info(ctx, "server").Result()
	if err != nil {
		fmt.Printf("Failed to connect to Valkey: %v\n", err)
	} else {
		fmt.Printf("Connected to Valkey. Server info:\n%s\n", info)
	}
}

func RateLimitLogin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		type LoginRequest struct {
			Username string `json:"username"`
		}
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil || req.Username == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
		}
		key := fmt.Sprintf("login_rate:%s", req.Username)
		ctx := context.Background()
		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Rate limiter error"})
		}
		if count == 1 {
			redisClient.Expire(ctx, key, 60*time.Second)
		}
		if count > 5 {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Too many login attempts. Please try again later."})
		}
		return c.Next()
	}
}

func RateLimitAll() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var userID string
		// Try to extract user from JWT (for authenticated requests)
		bearer := c.Get("Authorization")
		if len(bearer) > 7 && bearer[:7] == "Bearer " {
			token := bearer[7:]
			claims, err := utils.ParseJWT(token)
			if err == nil && claims["sub"] != nil {
				userID = claims["sub"].(string)
			}
		}
		// For login, extract username from body
		if userID == "" && c.Path() == "/login" && c.Method() == fiber.MethodPost {
			type LoginRequest struct {
				Username string `json:"username"`
			}
			var req LoginRequest
			if err := c.BodyParser(&req); err == nil && req.Username != "" {
				userID = req.Username
			}
		}
		// Fallback to IP address
		if userID == "" {
			userID = c.IP()
		}
		key := fmt.Sprintf("rate:%s:%s", userID, c.Path())
		ctx := context.Background()
		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Rate limiter error"})
		}
		if count == 1 {
			redisClient.Expire(ctx, key, 60*time.Second)
		}
		if count > 5 {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Too many requests. Please try again later."})
		}
		return c.Next()
	}
}
