package yfrog

import "github.com/gofiber/fiber/v2"

func ConfigureRoutes(app *fiber.App) {
	app.Post("/yfrog/api/xauth_upload")
}
