package twitterv1

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/gofiber/fiber/v2"
	"github.com/h2non/bimg"
)

var httpClient = &http.Client{}

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
		":profile_bigger": {"128", "128", false},
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

	resp, err := httpClient.Get(imageURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch image")
	}
	defer resp.Body.Close()

	if resizeOption == "none" || (width == 0 && height == 0) {
		c.Response().Header.Set("Cache-Control", "public, max-age=1209600")
		c.Response().Header.Set("Content-Type", resp.Header.Get("Content-Type"))
		imgBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to recieve image")
		}
		return c.Send(imgBytes)
	}

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to recieve image")
	}

	imgMetadata, err := bimg.Metadata(imgBytes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("bad img")
	}

	if maintainAspect {
		w, h := imgMetadata.Size.Width, imgMetadata.Size.Height
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

	o := bimg.Options{}

	if width > 0 || height > 0 {
		switch resizeOption {
		case "fit":
			o = bimg.Options{
				Height: height,
				Width:  width,
			}
		case "crop":
			o = bimg.Options{
				Height: height,
				Width:  width,
				Crop:   true,
			}
		case "none":
			c.Set("Content-Type", "image/"+imgMetadata.Type)

			c.Response().Header.Set("Cache-Control", "public, max-age=1209600")
			return c.Send(imgBytes)
			// Do nothing
		default:
			o = bimg.Options{
				Height: height,
				Width:  width,
			}
		}
	}

	imgScaled, err := bimg.Resize(imgBytes, o)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode image")
	}

	c.Set("Content-Type", "image/"+imgMetadata.Type)

	c.Response().Header.Set("Cache-Control", "public, max-age=1209600")
	return c.Send(imgScaled)
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
	c.Set("Cache-Control", "public, max-age=900") // 15 minutes
	return c.Redirect(userinfo.ProfileImageURL)
}

// This is here because it doesn't just want a direct link to the m3u8 file.
// So we make an extremely basic site that just includes the video, and maybe the alt text if i care enough
func CDNVideoProxy(c *fiber.Ctx) error {
	video_url := "https://video.bsky.app/watch/" + c.Params("did") + "/" + c.Params("link") + "/720p/video.m3u8" // 720p on an iphone 2g oh god
	thumbnail_url := "https://video.cdn.bsky.app/hls/" + c.Params("did") + "/" + c.Params("link") + "/thumbnail.jpg"

	c.Context().SetContentType("text/html")

	c.Set("Cache-Control", "public, max-age=604800")

	return c.SendString(fmt.Sprintf(videoTemplate, thumbnail_url, video_url))
}

var videoTemplate = `<meta content="width=device-width,initial-scale=1"name=viewport><title>Bluesky Video</title><style>*{margin:0;padding:0;width:100%%;height:100%%}</style><body><script src=https://cdn.jsdelivr.net/npm/hls.js@1></script><video autoplay="autoplay" controls id=v poster=%s src=%s></video><script>var v=document.getElementById("v");if(v.canPlayType("application/vnd.apple.mpegurl"));else if(Hls.isSupported()){var h=new Hls;h.loadSource(v.src),h.attachMedia(v)}</script>`
