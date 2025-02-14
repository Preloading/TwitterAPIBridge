package twitterv1

import (
	"crypto/sha1"
	"encoding/hex"

	"github.com/Preloading/TwitterAPIBridge/db_controller"
	"github.com/gofiber/fiber/v2"
)

// CreateShortLink generates a short link for the given original URL
func CreateShortLink(originalPath string) (string, error) {
	hash := sha1.New()
	hash.Write([]byte(originalPath))
	shortCode := hex.EncodeToString(hash.Sum(nil))[:8]

	err := db_controller.StoreShortLink(shortCode, originalPath)
	if err != nil {
		return "", err
	}

	return shortCode, nil
}

// RedirectToLink redirects to the original URL using the short code
func RedirectToLink(c *fiber.Ctx) error {

	shortCode := c.Params("ref")
	originalURL, err := db_controller.GetOriginalURL(shortCode)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Could not find image!")
	}

	return c.Redirect(configData.CdnURL+originalURL, fiber.StatusMovedPermanently)
}
