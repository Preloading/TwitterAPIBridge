package twitterv1

import (
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/gofiber/fiber/v2"
	"github.com/nfnt/resize"
)

func CDNDownscaler(c *fiber.Ctx) error {
	imageURL := c.Query("url")
	widthStr := c.Query("width")
	heightStr := c.Query("height")
	resizeOption := c.Query("resize")
	maintainAspect := false

	// Handle URL unescaping first if it's not a direct DID request
	if c.Params("did") != "" {
		did := c.Params("did")
		link := c.Params("link")
		size := c.Params("size")

		// Remove any file extension from link
		link = strings.TrimSuffix(link, filepath.Ext(link))

		imageURL = "https://cdn.bsky.app/img/feed_thumbnail/plain/" + did + "/" + link + "@jpeg"

		// If size is provided as a path parameter, treat it as a suffix
		if size != "" {
			if strings.HasPrefix(size, "mobile") || strings.HasPrefix(size, "web") || strings.HasPrefix(size, "ipad") {
				size = "/" + size
			} else {
				size = ":" + size
			}
			// This will be processed by the suffix handling below
			imageURL = imageURL + size
		}
	} else {
		unescapedURL, err := url.QueryUnescape(imageURL)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid URL")
		}
		imageURL = unescapedURL
	}

	// Check suffixes and set dimensions before any other URL validation
	suffixes := map[string]struct {
		width  string
		height string
		aspect bool
	}{
		":large":          {"", "", true},
		":small":          {"340", "480", true},
		":medium":         {"600", "1200", true},
		":thumb":          {"150", "150", false},
		":profile_bigger": {"73", "73", false},
		":profile_normal": {"48", "48", false},
		":profile_mini":   {"24", "24", false},
		"/mobile_retina":  {"620", "320", false},
		"/mobile":         {"320", "160", false},
		"/ipad":           {"626", "313", false},
		"/ipad_retina":    {"1252", "626", false},
		"/web":            {"520", "260", false},
		"/web_retina":     {"1040", "520", false},
	}

	// Check for suffixes and apply dimensions
	for suffix, dims := range suffixes {
		if strings.HasSuffix(imageURL, suffix) {
			imageURL = strings.TrimSuffix(imageURL, suffix)
			if dims.width != "" {
				widthStr = dims.width
				heightStr = dims.height
			}
			maintainAspect = dims.aspect
			break
		}
	}

	// Validate URL after suffix removal
	if c.Params("did") == "" && !strings.HasPrefix(imageURL, "https://cdn.bsky.app/img/") {
		return c.SendStatus(fiber.StatusBadRequest)
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

	if maintainAspect {
		w, h := img.Bounds().Dx(), img.Bounds().Dy()
		if w > h {
			w = width
			h = int(float64(width) * float64(h) / float64(w))
		} else {
			h = height
			w = int(float64(height) * float64(w) / float64(h))
		}
		width = w
		height = h
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

func UserProfileImage(c *fiber.Ctx) error {
	// auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	screen_name := c.Query("screen_name")
	if screen_name == "" {
		return c.Status(fiber.StatusBadRequest).SendString("screen_name is required")
	}
	// size := c.Query("size")
	// if size == "" {
	// 	size = "normal"
	// }
	// cdn_size := ":profile_bigger"

	// switch size {
	// case "normal":
	// 	cdn_size = ":profile_normal"
	// case "original":
	// 	cdn_size = ":large"
	// }

	userinfo, err := blueskyapi.GetUserInfo(*pds, *oauthToken, screen_name, false)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	//c.Redirect("https://cdn.bsky.app/img/" + screen_name + ":profile_bigger")
	return c.Redirect(userinfo.ProfileImageURL)
}
