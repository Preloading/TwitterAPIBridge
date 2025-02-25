package yfrog

import "github.com/gofiber/fiber/v2"

func MediaUpload(c *fiber.Ctx) error {
	return c.SendString("Media upload")
}
