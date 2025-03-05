package twitterv1

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/gofiber/fiber/v2"
	"github.com/nfnt/resize"
)

func CDNDownscaler(c *fiber.Ctx) error {
	imageURL := c.Query("url")

	if c.Params("did") != "" {
		fmt.Println(c.Params("did"))
		fmt.Println(c.Params("link"))
		fmt.Println(c.Params("filetype"))
		imageURL = "https://cdn.bsky.app/img/feed_thumbnail/plain/" + c.Params("did") + "/" + c.Params("link") + "@jpeg"
	} else {
		unescapedURL, err := url.QueryUnescape(imageURL)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid URL")
		}

		imageURL = unescapedURL
		if !strings.HasPrefix(imageURL, "https://cdn.bsky.app/img/") { // Later maybe lift these restrictions?
			return c.SendStatus(fiber.StatusBadRequest)
		}
	}

	widthStr := c.Query("width")
	heightStr := c.Query("height")
	resizeOption := c.Query("resize")

	maintainAspect := false

	// So twitter likes to do a stupid thing where it appends :small or :large to the end of tweet images, so we need to strip that, and use that for dimentions

	if strings.HasSuffix(imageURL, ":large") {
		imageURL = strings.TrimSuffix(imageURL, ":large")

		// We do know what large is, buuuut it seems to work fine if we give the raw image, and i think that's fiiiiiiine
		widthStr = ""
		heightStr = ""
		resizeOption = "none"
		maintainAspect = true

	}
	// https://web.archive.org/web/20120412055327/https://dev.twitter.com/docs/api/1/get/help/configuration

	if strings.HasSuffix(imageURL, ":small") {
		imageURL = strings.TrimSuffix(imageURL, ":small")

		widthStr = "340"
		heightStr = "480"
		maintainAspect = true
	}
	if strings.HasSuffix(imageURL, ":medium") {
		imageURL = strings.TrimSuffix(imageURL, ":medium")

		widthStr = "600"
		heightStr = "1200"
		maintainAspect = true
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

// This is here because it doesn't just want a direct link to the m3u8 file.
// So we make an extremely basic site that just includes the video, and maybe the alt text if i care enough
func CDNVideoProxy(c *fiber.Ctx) error {
	video_url := "https://video.bsky.app/watch/" + c.Params("did") + "/" + c.Params("link") + "/720p/video.m3u8" // 720p on an iphone 2g oh god
	thumbnail_url := "https://video.cdn.bsky.app/hls/" + c.Params("did") + "/" + c.Params("link") + "/thumbnail.jpg"

	c.Context().SetContentType("text/html")

	// Is the below minified? yup!
	// tbh, you could pretty easily just run it thru a prettier and it would make sense, but i'll explain it here:
	// It's a basic html page that includes the hls.js library, a video element, and a script that checks if the browser supports hls, and if it doesn't, it uses hls.js to play the video
	// why u hef to be mad golang warning thingy?
	return c.SendString(fmt.Sprintf(`
	<meta content="width=device-width,initial-scale=1"name=viewport><title>Bluesky Video</title><style>*{margin:0;padding:0;width:100%%;height:100%%}</style><body><script src=https://cdn.jsdelivr.net/npm/hls.js@1></script><video autoplay="autoplay" controls id=v poster=%s src=%s></video><script>var v=document.getElementById("v");if(v.canPlayType("application/vnd.apple.mpegurl"));else if(Hls.isSupported()){var h=new Hls;h.loadSource(v.src),h.attachMedia(v)}</script>
	`, thumbnail_url, video_url))
}
