package twitterv1

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nfnt/resize"
)

func CDNDownscaler(c *fiber.Ctx) error {
	imageURL := c.Query("url")
	fmt.Println(imageURL)
	if !strings.HasPrefix(imageURL, "https://cdn.bsky.app/img/") { // Later maybe lift these restrictions?
		return c.SendStatus(fiber.StatusBadRequest)
	}

	widthStr := c.Query("width")
	heightStr := c.Query("height")

	// So twitter likes to do a stupid thing where it appends :small or :large to the end of tweet images, so we need to strip that, and use that for dimentions

	if strings.HasSuffix(imageURL, ":small") {
		imageURL = strings.TrimSuffix(imageURL, ":small")

		// TODO: Find what these values actually used to be
		// They used to be https://web.archive.org/web/20120412055327/https://dev.twitter.com/docs/api/1/get/help/configuration
		widthStr = "320"
		heightStr = ""
	}
	if strings.HasSuffix(imageURL, ":thumb") {
		imageURL = strings.TrimSuffix(imageURL, ":thumb")

		// TODO: Find what these values actually used to be
		widthStr = "150"
		heightStr = "150"

	}
	if strings.HasSuffix(imageURL, ":large") {
		imageURL = strings.TrimSuffix(imageURL, ":large")

		// TODO: Find what these values actually used to be
		widthStr = ""
		heightStr = ""

	}

	width, err := strconv.Atoi(widthStr)
	if err != nil {
		width = 0
	}
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		height = 0
	}

	resp, err := http.Get(imageURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch image")
	}
	defer resp.Body.Close()

	img, format, err := image.Decode(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to decode image")
	}

	if width > 0 || height > 0 {
		img = resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
	}

	c.Set("Content-Type", "image/"+format)
	switch format {
	case "jpeg":
		err = jpeg.Encode(c.Response().BodyWriter(), img, nil)
	case "png":
		err = png.Encode(c.Response().BodyWriter(), img)
	case "gif":
		err = gif.Encode(c.Response().BodyWriter(), img, nil)
	default:
		return c.Status(fiber.StatusInternalServerError).SendString("Unsupported image format")
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode image")
	}

	return nil
}
