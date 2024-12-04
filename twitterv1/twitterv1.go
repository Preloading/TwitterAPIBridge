package twitterv1

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math/big"
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

	// Posts
	app.Get("/1/statuses/home_timeline.json", home_timeline)
	app.Get("/1/statuses/show/:id.json", GetStatusFromId)

	// Users
	app.Get("/1/users/show.xml", user_info)

	// Trends
	app.Get("/1/trends/:woeid.json", trends_woeid)

	// Setings
	app.Get("/1/account/settings.xml", GetSettings)

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
		tweets = append(tweets, TranslatePostToTweet(item.Post, item.Reply.Parent.URI, item.Reply.Parent.Author.DID))
	}

	return c.JSON(tweets)

}

func GetStatusFromId(c *fiber.Ctx) error {
	encodedId := c.Params("id")
	idBigInt, ok := new(big.Int).SetString(encodedId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	uri := bridge.TwitterIDToBlueSky(idBigInt)

	fmt.Println("URI: " + uri)
	authHeader := c.Get("Authorization")
	fmt.Println("Auth: " + authHeader)
	// Define a regular expression to match the oauth_token
	re := regexp.MustCompile(`oauth_token="([^"]+)"`)
	matches := re.FindStringSubmatch(authHeader)

	if len(matches) < 2 {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	oauthToken := matches[1]

	err, thread := blueskyapi.GetPost(oauthToken, uri)

	if err != nil {
		return err
	}

	return c.JSON(TranslatePostToTweet(thread.Post, "", ""))
}

func TranslatePostToTweet(tweet blueskyapi.Post, replyMsgBskyURI string, replyUserBskyId string) bridge.Tweet {
	tweetEntities := bridge.Entities{
		Hashtags:     nil,
		Urls:         nil,
		UserMentions: []bridge.UserMention{},
		Media:        []bridge.Media{},
	}

	id := 1
	for _, image := range tweet.Record.Embed.Images {
		// Process each image
		// fmt.Println("Image:", "http://10.0.0.77:3000/cdn/img/?url="+url.QueryEscape("https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:"+item.Post.Author.DID+"/"+image.Image.Ref.Link+"/@jpeg"))
		tweetEntities.Media = append(tweetEntities.Media, bridge.Media{
			ID:       *big.NewInt(int64(id)),
			IDStr:    strconv.Itoa(id),
			MediaURL: "http://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape("https://cdn.bsky.app/img/feed_thumbnail/plain/"+tweet.Author.DID+"/"+image.Image.Ref.Link+"/@jpeg"),
			// MediaURLHttps: "https://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape("https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:"+image.Image.Ref.Link+"@jpeg"),
		})
		id++
	}
	for _, faucet := range tweet.Record.Facets {
		// I haven't seen this exceed 1 element yet
		// if len(faucet.Features) > 1 {
		// fmt.Println("Faucet with more than 1 feature found!")
		// faucetJSON, err := json.Marshal(faucet)
		// if err != nil {
		// 	fmt.Println("Error encoding faucet to JSON:", err)
		// } else {
		// 	fmt.Println("Faucet JSON:", string(faucetJSON))
		// }
		// // }
		// fmt.Println(faucet.Features[0].Type)
		switch faucet.Features[0].Type {
		case "app.bsky.richtext.facet#mention":
			fmt.Println("we found a mention")
			tweetEntities.UserMentions = append(tweetEntities.UserMentions, bridge.UserMention{
				Name:       "test",
				ScreenName: "test",
				//ScreenName: item.Post.Record.Text[faucet.Index.ByteStart+1 : faucet.Index.ByteEnd],
				ID: *bridge.BlueSkyToTwitterID(faucet.Features[0].Did),
				Indices: []int{
					faucet.Index.ByteStart,
					faucet.Index.ByteEnd,
				},
			})
		}

	}

	convertedTweet := bridge.Tweet{
		Coordinates:  nil,
		Favourited:   tweet.Viewer.Like,
		CreatedAt:    bridge.TwitterTimeConverter(tweet.Record.CreatedAt),
		Truncated:    false,
		Text:         tweet.Record.Text,
		Entities:     tweetEntities,
		Annotations:  nil,
		Contributors: nil,
		ID:           *bridge.BlueSkyToTwitterID(tweet.URI),
		Geo:          nil,
		Place:        nil,
		InReplyToUserID: func() *big.Int {
			id := bridge.BlueSkyToTwitterID(replyUserBskyId)
			if id.Cmp(big.NewInt(0)) == 0 {
				fmt.Println("null")
				return nil
			}
			return id
		}(),
		User: bridge.TwitterUser{
			Name:                      tweet.Author.DisplayName,
			ProfileSidebarBorderColor: "",
			ProfileBackgroundTile:     false,
			ProfileSidebarFillColor:   "",
			CreatedAt:                 bridge.TwitterTimeConverter(tweet.Author.Associated.CreatedAt),
			ProfileImageURL:           "http://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape(tweet.Author.Avatar) + "&width=128&height=128",
			Location:                  "",
			ProfileLinkColor:          "",
			FollowRequestSent:         false,
			URL:                       "",
			FavouritesCount:           0,
			ScreenName:                tweet.Author.Handle,
			ContributorsEnabled:       false,
			UtcOffset:                 -28800,
			ID:                        *bridge.BlueSkyToTwitterID(tweet.URI),
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
			// FriendsCount:              100,
			// StatusesCount:             333,
			ProfileBackgroundImageURL: "http://a0.twimg.com/images/themes/theme1/bg.png",
			Following:                 false,
		},
		Source: "Bluesky",
		InReplyToStatusID: func() *big.Int {
			id := bridge.BlueSkyToTwitterID(replyMsgBskyURI) // hack, later probably do this more efficently
			if id.Cmp(big.NewInt(0)) == 0 {
				fmt.Println("null")
				return nil
			}
			return bridge.BlueSkyToTwitterID(replyMsgBskyURI)
		}(),
	}
	return convertedTweet
}

func user_info(c *fiber.Ctx) error {
	screen_name := c.Query("screen_name")
	authHeader := c.Get("Authorization")

	// Define a regular expression to match the oauth_token
	re := regexp.MustCompile(`oauth_token="([^"]+)"`)
	matches := re.FindStringSubmatch(authHeader)

	oauthToken := ""

	if len(matches) < 2 {
		//return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
		// This request supports without a token
		oauthToken = ""
	} else {
		oauthToken = matches[1]
	}

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

func Search(c *fiber.Ctx) error {
	q := c.Query("q")
	fmt.Println("Search query:", q)
	return c.SendStatus(fiber.StatusNotImplemented)
}

func PushDestinations(c *fiber.Ctx) error {
	// This request is /1/account/push_destinations/device.xml, and i cannot find any info on this.
	return c.SendStatus(fiber.StatusNotImplemented)
}

// TODO
// 99% sure this relies on PushDestinations, which i have no data on.
func GetSettings(c *fiber.Ctx) error {
	return c.XML(bridge.Config{
		SleepTime: bridge.SleepTime{
			EndTime:   nil,
			Enabled:   true,
			StartTime: nil,
		},
		TrendLocation: []bridge.TrendLocation{
			{
				Name:  "Worldwide",
				Woeid: 1,
				PlaceType: bridge.PlaceType{
					Name: "Supername",
					Code: 19,
				},
				Country:     "",
				URL:         "http://where.yahooapis.com/v1/place/1",
				CountryCode: nil,
			},
		},
		Language:            "en",
		AlwaysUseHttps:      false,
		DiscoverableByEmail: true,
		TimeZone: bridge.TimeZone{
			Name:       "Pacific Time (US & Canada)",
			TzinfoName: "America/Los_Angeles",
			UtcOffset:  -28800,
		},
		GeoEnabled: true,
	})

}

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
		widthStr = "320"
		heightStr = ""
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
