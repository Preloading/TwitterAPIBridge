package twitterv1

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/nfnt/resize"
)

func InitServer() {
	app := fiber.New()

	// Initialize default config
	app.Use(logger.New())

	// Custom middleware to log request details
	// app.Use(func(c *fiber.Ctx) error {
	// 	fmt.Println("Request Method:", c.Method())
	// 	fmt.Println("Request URL:", c.OriginalURL())
	// 	fmt.Println("Post Body:", string(c.Body()))
	// 	fmt.Println("Headers:", string(c.Request().Header.Header()))
	// 	fmt.Println()
	// 	return c.Next()
	// })

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// Auth
	app.Post("/oauth/access_token", access_token)

	// Posting
	app.Post("/1/statuses/update.json", status_update)

	// Timelines
	app.Get("/1/statuses/home_timeline.json", home_timeline)

	// Users
	app.Get("/1/users/show.xml", user_info)

	// Trends
	app.Get("/1/trends/:woeid.json", trends_woeid)

	// CDN Downscaler
	app.Get("/cdn/img", CDNDownscaler)

	app.Listen(":3000")
}

// https://developer.x.com/en/docs/authentication/api-reference/access_token
// and
// https://web.archive.org/web/20120708225149/https://dev.twitter.com/docs/oauth/xauth
func access_token(c *fiber.Ctx) error {
	// Parse the form data
	//sendErrorCodes := c.FormValue("send_error_codes")
	authMode := c.FormValue("x_auth_mode")
	authPassword := c.FormValue("x_auth_password")
	authUsername := c.FormValue("x_auth_username")

	if authMode == "client_auth" {
		res, err := blueskyapi.Authenticate(authUsername, authPassword)
		if err != nil {
			fmt.Println("Error:", err)
			return c.SendStatus(401)
		}
		fmt.Println("AccessJwt:", res.AccessJwt)
		fmt.Println("RefreshJwt:", res.RefreshJwt)
		fmt.Println("User ID:", res.DID)
		return c.SendString(fmt.Sprintf("oauth_token=%s&oauth_token_secret=%s&user_id=%s&screen_name=twitterapi&x_auth_expires=900", res.AccessJwt, res.RefreshJwt, bridge.BlueSkyToTwitterID(res.DID).String()))
		// TODO: add x_auth_expires
	}
	// This is a problem from when I actually get this connected to bluesky
	return c.SendStatus(501)
}

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/post/statuses/update
func status_update(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	// Define a regular expression to match the oauth_token
	re := regexp.MustCompile(`oauth_token="([^"]+)"`)
	matches := re.FindStringSubmatch(authHeader)

	if len(matches) < 2 {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	oauthToken := matches[1]

	status := c.FormValue("status")
	trim_user := c.FormValue("trim_user")
	in_reply_to_status_id := c.FormValue("in_reply_to_status_id")

	fmt.Println("Status:", status)
	fmt.Println("TrimUser:", trim_user)
	fmt.Println("InReplyToStatusID:", in_reply_to_status_id)

	if err := blueskyapi.UpdateStatus(oauthToken, status); err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update status")
	}

	// TODO: Implement this

	return c.SendString("Not implemented")
}

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/get/statuses/home_timeline
func home_timeline(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	// Define a regular expression to match the oauth_token
	re := regexp.MustCompile(`oauth_token="([^"]+)"`)
	matches := re.FindStringSubmatch(authHeader)

	if len(matches) < 2 {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	oauthToken := matches[1]

	err, res := blueskyapi.GetTimeline(oauthToken)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch timeline")
	}

	tweets := []bridge.Tweet{}

	for _, item := range res.Feed {
		tweet := bridge.Tweet{
			Coordinates:     nil,
			Favourited:      item.Post.Viewer.Like,
			CreatedAt:       bridge.TwitterTimeConverter(item.Post.Record.CreatedAt),
			Truncated:       false,
			Text:            item.Post.Record.Text,
			Annotations:     nil,
			Contributors:    nil,
			ID:              *bridge.BlueSkyToTwitterID(item.Post.CID),
			Geo:             nil,
			Place:           nil,
			InReplyToUserID: 0,
			User: bridge.TwitterUser{
				Name:                      item.Post.Author.DisplayName,
				ProfileSidebarBorderColor: "",
				ProfileBackgroundTile:     false,
				ProfileSidebarFillColor:   "",
				CreatedAt:                 bridge.TwitterTimeConverter(item.Post.Author.Associated.CreatedAt),
				ProfileImageURL:           "http://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape(item.Post.Author.Avatar) + "&width=128&height=128",
				Location:                  "",
				ProfileLinkColor:          "",
				FollowRequestSent:         false,
				URL:                       "",
				FavouritesCount:           0,
				ScreenName:                item.Post.Author.Handle,
				ContributorsEnabled:       false,
				UtcOffset:                 -28800,
				ID:                        *bridge.BlueSkyToTwitterID(item.Post.Author.DID),
				ProfileUseBackgroundImage: true,
				ProfileTextColor:          "333333",
				Protected:                 false,
				FollowersCount:            200,
				Lang:                      "en",
				Notifications:             false,
				TimeZone:                  "Pacific Time (US & Canada)",
				Verified:                  false,
				ProfileBackgroundColor:    "C0DEED",
				GeoEnabled:                true,
				Description:               "",
				FriendsCount:              100,
				StatusesCount:             333,
				ProfileBackgroundImageURL: "http://a0.twimg.com/images/themes/theme1/bg.png",
				Following:                 false,
			},
		}
		tweets = append(tweets, tweet)
	}

	return c.JSON(tweets)

	// return c.JSON([]bridge.Tweet{
	// 	{
	// 		Coordinates:     nil,
	// 		Favourited:      false,
	// 		CreatedAt:       "Wed Sep 01 00:00:00 +0000 2021",
	// 		Truncated:       false,
	// 		Text:            "Will it blow up if there's a @ in the name",
	// 		Annotations:     nil,
	// 		Contributors:    nil,
	// 		ID:              4,
	// 		Geo:             nil,
	// 		Place:           nil,
	// 		InReplyToUserID: 0,
	// 		User: bridge.TweetUser{
	// 			Name:                      "Preloading",
	// 			ProfileSidebarBorderColor: "C0DEED",
	// 			ProfileBackgroundTile:     false,
	// 			ProfileSidebarFillColor:   "DDEEF6",
	// 			CreatedAt:                 "Wed Sep 01 00:00:00 +0000 2021",
	// 			ProfileImageURL:           "https://cdn.bsky.app/img/avatar_thumbnail/plain/did:plc:khcyntihpu7snjszuojjgjc4/bafkreignfoswre6f2ehujkifewpk2xdlrqhfhraloaoixjf5dommpzjxeq@png",
	// 			Location:                  "San Francisco",
	// 			ProfileLinkColor:          "0084B4",
	// 			FollowRequestSent:         false,
	// 			URL:                       "http://dev.twitter.com",
	// 			FavouritesCount:           8,
	// 			ContributorsEnabled:       false,
	// 			UtcOffset:                 -28800,
	// 			ID:                        2,
	// 			ProfileUseBackgroundImage: true,
	// 			ProfileTextColor:          "333333",
	// 			Protected:                 false,
	// 			FollowersCount:            200,
	// 			Lang:                      "en",
	// 			Notifications:             false,
	// 			TimeZone:                  "Pacific Time (US & Canada)",
	// 			Verified:                  false,
	// 			ProfileBackgroundColor:    "C0DEED",
	// 			GeoEnabled:                true,
	// 			Description:               "A developer just looking to make some cool stuff",
	// 			FriendsCount:              100,
	// 			StatusesCount:             333,
	// 			ProfileBackgroundImageURL: "http://a0.twimg.com/images/themes/theme1/bg.png",
	// 			Following:                 false,
	// 			ScreenName:                "preloading",
	// 		},
	// 		Source: "web",
	// 	},
	// })

}

func user_info(c *fiber.Ctx) error {
	screen_name := c.Query("screen_name")
	authHeader := c.Get("Authorization")

	// if authHeader == "" {
	// 	return c.Status(fiber.StatusUnauthorized).SendString("Authorization header missing")
	// }

	// Define a regular expression to match the oauth_token
	re := regexp.MustCompile(`oauth_token="([^"]+)"`)
	matches := re.FindStringSubmatch(authHeader)

	if len(matches) < 2 {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	oauthToken := matches[1]

	userinfo, err := blueskyapi.GetUserInfo(oauthToken, screen_name)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	// user := bridge.TweetUser{
	// 	Name:                      "Preloading",
	// 	ProfileSidebarBorderColor: "C0DEED",
	// 	ProfileBackgroundTile:     false,
	// 	ProfileSidebarFillColor:   "DDEEF6",
	// 	CreatedAt:                 "Wed Sep 01 00:00:00 +0000 2021",
	// 	ProfileImageURL:           "https://cdn.bsky.app/img/avatar_thumbnail/plain/did:plc:khcyntihpu7snjszuojjgjc4/bafkreignfoswre6f2ehujkifewpk2xdlrqhfhraloaoixjf5dommpzjxeq@png",
	// 	Location:                  "San Francisco",
	// 	ProfileLinkColor:          "0084B4",
	// 	FollowRequestSent:         false,
	// 	URL:                       "http://dev.twitter.com",
	// 	FavouritesCount:           8,
	// 	ContributorsEnabled:       false,
	// 	UtcOffset:                 -28800,
	// 	ID:                        2,
	// 	ProfileUseBackgroundImage: true,
	// 	ProfileTextColor:          "333333",
	// 	Protected:                 false,
	// 	FollowersCount:            200,
	// 	Lang:                      "en",
	// 	Notifications:             false,
	// 	TimeZone:                  "Pacific Time (US & Canada)",
	// 	Verified:                  false,
	// 	ProfileBackgroundColor:    "C0DEED",
	// 	GeoEnabled:                true,
	// 	Description:               "A developer just looking to make some cool stuff",
	// 	FriendsCount:              100,
	// 	StatusesCount:             333,
	// 	ProfileBackgroundImageURL: "http://a0.twimg.com/images/themes/theme1/bg.png",
	// 	Following:                 false,
	// 	ScreenName:                screen_name,
	// }
	return c.XML(userinfo)
}

// https://web.archive.org/web/20120313235613/https://dev.twitter.com/docs/api/1/get/trends/%3Awoeid
// For now, we will be pretending WOEID doesn't exist
func trends_woeid(c *fiber.Ctx) error {
	type Trends struct {
		Created string `json:"created_at"`
		Trends  []struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			Promoted    bool   `json:"promoted"`
			Query       string `json:"query"`
			TweetVolume int    `json:"tweet_volume"`
		} `json:"trends"`
		AsOf      string `json:"as_of"`
		Locations []struct {
			Name  string `json:"name"`
			WOEID int    `json:"woeid"`
		} `json:"locations"`
	}

	return c.JSON(Trends{
		Created: "2021-09-01T00:00:00Z",
		Trends: []struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			Promoted    bool   `json:"promoted"`
			Query       string `json:"query"`
			TweetVolume int    `json:"tweet_volume"`
		}{
			{
				Name:        "Trending Topic",
				URL:         "https://twitter.com/search?q=%22Trending%20Topic%22",
				Promoted:    false,
				Query:       "%22Trending%20Topic%22",
				TweetVolume: 10000,
			},
		},
		AsOf: "2021-09-01T00:00:00Z",
		Locations: []struct {
			Name  string `json:"name"`
			WOEID int    `json:"woeid"`
		}{
			{
				Name:  "Worldwide",
				WOEID: 1,
			},
		},
	})
}

func CDNDownscaler(c *fiber.Ctx) error {
	imageURL := c.Query("url")
	if !strings.HasPrefix(imageURL, "https://cdn.bsky.app/img/") { // Later maybe lift these restrictions?
		return c.SendStatus(fiber.StatusBadRequest)
	}

	widthStr := c.Query("width")
	heightStr := c.Query("height")

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
