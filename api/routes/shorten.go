package routes

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/itskaransingh/url-shortener/database"
	"github.com/itskaransingh/url-shortener/helpers"
)

type Request struct {
	Url          string        `json:"url"`
	CoustomShort string        `json:"short"`
	Expiry       time.Duration `json:"expiry"`
}

type Response struct {
	Url                 string        `json:"url"`
	CoustomShort        string        `json:"short"`
	Expiry              time.Duration `json:"expiry"`
	XRateLimitRemaining int           `json:"rate_limit_remaining"`
	XRateLimitReset     time.Duration `json:"rate_limit_reset"`
}

func ShortenUrl(c *fiber.Ctx) error {
	body := new(Request)

	err := c.BodyParser(&body) // Parse the request body into struct

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to parse request body",
		})
	}

	//Implement rate limiting

	redisClientForRateLimiting := database.CreateClient(1)
	defer redisClientForRateLimiting.Close()

	value, err := redisClientForRateLimiting.Get(database.Ctx, c.IP()).Result()
	if err == redis.Nil {
		redisClientForRateLimiting.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second)
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Db Not connected",
		})
	} else {
		intValue, _ := strconv.Atoi(value)
		fmt.Println(intValue)

		if intValue <= 0 {
			limit, _ := redisClientForRateLimiting.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":            "Rate limit exceeded",
				"rate_limit_reset": limit / time.Nanosecond / time.Minute,
			})
		}
	}

	// Check if URL is valid
	if !govalidator.IsURL(body.Url) {
		return c.Status(fiber.StatusBadRequest).JSON((fiber.Map{
			"error": "Invalid URL",
		}))
	}

	//Remove domian error
	if !helpers.CheckDomainError(body.Url) {
		return c.Status(fiber.StatusBadRequest).JSON((fiber.Map{
			"error": "Invalid domain",
		}))
	}

	//Enforce http

	body.Url = helpers.EnforceHTTP(body.Url)

	//Assigning shortUrl

	short := body.CoustomShort

	if short == "" {
		short = uuid.New().String()[:6]
	}

	redisClient := database.CreateClient(0)
	defer redisClient.Close()

	err = redisClient.Get(database.Ctx, short).Err()

	if err == redis.Nil {
		if body.Expiry == 0 {
			body.Expiry = 24
		}

		redisClient.Set(database.Ctx, short, body.Url, body.Expiry*3600*time.Second)
	} else {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "The short url already exists",
		})
	}

	// Returning Response
	response := Response{
		Url:                 body.Url,
		CoustomShort:        os.Getenv("DOMAIN") + "/" + short,
		Expiry:              body.Expiry * 3600 * time.Second,
		XRateLimitRemaining: 10,
		XRateLimitReset:     30 * time.Minute,
	}

	redisClientForRateLimiting.Decr(database.Ctx, c.IP())

	val, _ := redisClientForRateLimiting.Get(database.Ctx, c.IP()).Result()
	response.XRateLimitRemaining, _ = strconv.Atoi(val)

	ttl, _ := redisClientForRateLimiting.TTL(database.Ctx, c.IP()).Result()
	fmt.Println(ttl)
	response.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	return c.Status(fiber.StatusOK).JSON(response)
}
