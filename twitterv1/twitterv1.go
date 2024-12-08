package twitterv1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func InitServer() {
	app := fiber.New()

	// Initialize default config
	app.Use(logger.New())

	// // Custom middleware to log request details
	// app.Use(func(c *fiber.Ctx) error {
	// 	fmt.Println("Request Method:", c.Method())
	// 	fmt.Println("Request URL:", c.OriginalURL())
	// 	fmt.Println("Post Body:", string(c.Body()))
	// 	fmt.Println("Headers:", string(c.Request().Header.Header()))
	// 	fmt.Println()
	// 	return c.Next()
	// })

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// Auth
	app.Post("/oauth/access_token", access_token)

	// Interactions
	app.Post("/1/statuses/update.json", status_update)
	app.Post("/1/statuses/retweet/:id.json", retweet)

	// Posts
	app.Get("/1/statuses/home_timeline.json", home_timeline)
	app.Get("/1/statuses/show/:id.json", GetStatusFromId)

	// Users
	app.Get("/1/users/show.xml", user_info)

	// Trends
	app.Get("/1/trends/:woeid.json", trends_woeid)

	// Setings
	app.Get("/1/account/settings.xml", GetSettings)
	app.Get("/1/account/push_destinations/device.xml", PushDestinations)

	// CDN Downscaler
	app.Get("/cdn/img", CDNDownscaler)

	app.Listen(":3000")
}
