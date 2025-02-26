package twitterv1

import (
	"crypto/sha256"
	"math/big"
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

func toBase62(num *big.Int) string {
	base := big.NewInt(62)
	zero := big.NewInt(0)
	result := make([]byte, 7) // Changed to 7 to leave room for collision handling
	idx := 6

	temp := new(big.Int).Set(num)
	for idx >= 0 {
		mod := new(big.Int)
		temp.DivMod(temp, base, mod)
		result[idx] = base62Chars[mod.Int64()]
		idx--
		if temp.Cmp(zero) == 0 {
			break
		}
	}
	// Pad with '0' if necessary
	for i := idx; i >= 0; i-- {
		result[i] = base62Chars[0]
	}
	return string(result)
}

func CreateShortLink(originalPath string) (string, error) {
	// Check cache first
	if code, ok := cache.Get(originalPath); ok {
		return code, nil
	}

	// Generate SHA-256 hash of the URL
	hash := sha256.New()
	hash.Write([]byte(originalPath))
	hashBytes := hash.Sum(nil)

	// Convert first 8 bytes of hash to a big.Int
	num := new(big.Int).SetBytes(hashBytes[:8])
	shortCode := toBase62(num)

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
