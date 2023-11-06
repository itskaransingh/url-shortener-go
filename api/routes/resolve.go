package routes

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/itskaransingh/url-shortener/database"
)

func ResolveUrl(c *fiber.Ctx) error {
	url := c.Params("url")
	redisClient := database.CreateClient(0)
	defer redisClient.Close()

	value, err := redisClient.Get(database.Ctx, url).Result()
	if err == redis.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Shorten URL Not Found",
		})
	} else if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Db Not connected",
		})
	}

	redisIncrement := database.CreateClient(1)
	defer redisIncrement.Close()

	_ = redisIncrement.Incr(database.Ctx, "Counter")

	return c.Redirect(value, 301)
}
