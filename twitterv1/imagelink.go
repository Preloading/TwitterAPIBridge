package twitterv1

import (
	"hash/fnv"
	"sync"

	"github.com/Preloading/TwitterAPIBridge/db_controller"
	"github.com/gofiber/fiber/v2"
)

// Simple cache for frequently accessed URLs
type urlCache struct {
	cache map[string]string
	mutex sync.RWMutex
}

var (
	base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	cache       = &urlCache{
		cache: make(map[string]string),
	}
)

// Get returns the cached shortcode for a URL
func (c *urlCache) Get(url string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	code, ok := c.cache[url]
	return code, ok
}

// Set caches a URL and its shortcode
func (c *urlCache) Set(url, code string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache[url] = code
}

func toBase62(num uint64) string {
	if num == 0 {
		return "0"
	}

	// Pre-allocate result with max possible length for uint64
	result := make([]byte, 11) // uint64 in base62 needs max 11 chars
	i := len(result) - 1

	for num > 0 && i >= 0 {
		result[i] = base62Chars[num%62]
		num /= 62
		i--
	}

	return string(result[i+1:])
}

func CreateShortLink(originalPath string, prefix string) (string, error) {
	// Check cache first
	if code, ok := cache.Get(originalPath); ok {
		return code, nil
	}

	// Create the URL somewhat predictably
	h := fnv.New64a()
	h.Write([]byte(originalPath))
	hash := h.Sum64()

	// Convert hash to base62
	shortCode := toBase62(hash)

	if len(shortCode) < 6 {
		shortCode = string(base62Chars[0]) + shortCode
	} else if len(shortCode) > 6 {
		shortCode = shortCode[:6]
	}

	shortCode = prefix + shortCode

	err := db_controller.StoreShortLink(shortCode, originalPath)
	if err != nil {
		return "", err
	}

	// Cache the result
	cache.Set(originalPath, shortCode)
	return shortCode, nil
}

func RedirectToLink(c *fiber.Ctx) error {

	shortCode := c.Params("ref")
	originalURL, err := db_controller.GetOriginalURL(shortCode)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Could not find image!")
	}

	return c.Redirect(configData.CdnURL+originalURL, fiber.StatusMovedPermanently)
}
