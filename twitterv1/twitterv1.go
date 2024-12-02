package twitterv1

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func InitServer() {
	app := fiber.New()

	// Initialize default config
	app.Use(logger.New())

	// Custom middleware to log request details
	app.Use(func(c *fiber.Ctx) error {
		fmt.Println("Request Method:", c.Method())
		fmt.Println("Request URL:", c.OriginalURL())
		fmt.Println("Post Body:", string(c.Body()))
		return c.Next()
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// Auth
	app.Post("/oauth/access_token", access_token)

	// Trends
	app.Get("/1/trends/:woeid.json", trends_woeid)

	app.Listen(":3000")
}

// https://developer.x.com/en/docs/authentication/api-reference/access_token
func access_token(c *fiber.Ctx) error {
	// Parse the form data
	sendErrorCodes := c.FormValue("send_error_codes")
	authMode := c.FormValue("x_auth_mode")
	authPassword := c.FormValue("x_auth_password")
	authUsername := c.FormValue("x_auth_username")

	fmt.Println("SendErrorCodes:", sendErrorCodes)
	fmt.Println("AuthMode:", authMode)
	fmt.Println("AuthPassword:", authPassword)
	fmt.Println("AuthUsername:", authUsername)

	return c.SendString("oauth_token=6253282-eWudHldSbIaelX7swmsiHImEL4KinwaGloHANdrY&oauth_token_secret=2EEfA6BG5ly3sR3XjE0IBSnlQu4ZrUzPiYTmrkVU&user_id=6253282&screen_name=twitterapi")
}

// https://web.archive.org/web/20120313235613/https://dev.twitter.com/docs/api/1/get/trends/%3Awoeid
func trends_woeid(c *fiber.Ctx) error {

}
