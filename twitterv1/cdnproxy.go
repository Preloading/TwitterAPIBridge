package twitterv1

import (
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nfnt/resize"
)

func CDNDownscaler(c *fiber.Ctx) error {
	imageURL := c.Query("url")
	unescapedURL, err := url.QueryUnescape(imageURL)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid URL")
	}
	imageURL = unescapedURL

	if !strings.HasPrefix(imageURL, "https://cdn.bsky.app/img/") { // Later maybe lift these restrictions?
		return c.SendStatus(fiber.StatusBadRequest)
	}

	widthStr := c.Query("width")
	heightStr := c.Query("height")
	resizeOption := c.Query("resize")

	// So twitter likes to do a stupid thing where it appends :small or :large to the end of tweet images, so we need to strip that, and use that for dimentions

	if strings.HasSuffix(imageURL, ":large") {
		imageURL = strings.TrimSuffix(imageURL, ":large")

		// We do know what large is, buuuut it seems to work fine if we give the raw image, and i think that's fiiiiiiine
		widthStr = ""
		heightStr = ""
		resizeOption = "none"

	}
	// https://web.archive.org/web/20120412055327/https://dev.twitter.com/docs/api/1/get/help/configuration

	if strings.HasSuffix(imageURL, ":small") {
		imageURL = strings.TrimSuffix(imageURL, ":small")

		widthStr = "340"
		heightStr = "480"
	}
	if strings.HasSuffix(imageURL, ":medium") {
		imageURL = strings.TrimSuffix(imageURL, ":medium")

		widthStr = "600"
		heightStr = "1200"
	}
	if strings.HasSuffix(imageURL, ":thumb") {
		imageURL = strings.TrimSuffix(imageURL, ":thumb")

		widthStr = "150"
		heightStr = "150"

	}

	if strings.HasSuffix(imageURL, ":profile_bigger") {
		imageURL = strings.TrimSuffix(imageURL, ":profile_bigger")

		widthStr = "73"
		heightStr = "73"

	}
	if strings.HasSuffix(imageURL, ":profile_normal") {
		imageURL = strings.TrimSuffix(imageURL, ":profile_normal")

		widthStr = "48"
		heightStr = "48"

	}
	if strings.HasSuffix(imageURL, ":profile_mini") {
		imageURL = strings.TrimSuffix(imageURL, ":profile_normal")

		widthStr = "24"
		heightStr = "24"

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
		switch resizeOption {
		case "fit":
			img = resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
		case "crop":
			img = resize.Thumbnail(uint(width), uint(height), img, resize.Lanczos3)
		case "none":
			// Do nothing
		default:
			img = resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
		}
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
	// cache for 30 minutes
	c.Response().Header.Set("Cache-Control", "public, max-age="+strconv.Itoa(int(30*time.Minute.Seconds())))
	return nil
}
