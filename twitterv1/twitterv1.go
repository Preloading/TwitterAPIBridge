package twitterv1

import (
	"fmt"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

type Tweet struct {
	Coordinates interface{} `json:"coordinates"` // I do not think anything implients this in modern day
	Favourited  bool        `json:"favorited"`
	CreatedAt   string      `json:"created_at"`
	Truncated   bool        `json:"truncated"`
	// lets agree for now that entities don't exist. that seems like a lot of effort
	Text            string      `json:"text"`
	Annotations     interface{} `json:"annotations"`  // Unknown
	Contributors    interface{} `json:"contributors"` // Unknown
	ID              int         `json:"id"`
	Geo             interface{} `json:"geo"`                 // I do not think anything impliments this in modern day
	Place           interface{} `json:"place"`               // Unknown
	InReplyToUserID int         `json:"in_reply_to_user_id"` // Unknown, but guessing int
	User            TweetUser   `json:"user"`
	Source          string      `json:"source"`
}

type TweetUser struct {
	Name                      string `json:"name"`
	ProfileSidebarBorderColor string `json:"profile_sidebar_border_color"` // Hex color (w/o hashtag)
	ProfileBackgroundTile     bool   `json:"profile_background_tile"`
	ProfileSidebarFillColor   string `json:"profile_sidebar_fill_color"` // Hex color (w/o hashtag)
	CreatedAt                 string `json:"created_at"`
	ProfileImageURL           string `json:"profile_image_url"`
	Location                  string `json:"location"`
	ProfileLinkColor          string `json:"profile_link_color"` // Hex color (w/o hashtag)
	FollowRequestSent         bool   `json:"follow_request_sent"`
	URL                       string `json:"url"`
	FavouritesCount           int    `json:"favourites_count"`
	ContributorsEnabled       bool   `json:"contributors_enabled"`
	UtcOffset                 int    `json:"utc_offset"`
	ID                        int    `json:"id"`
	ProfileUseBackgroundImage bool   `json:"profile_use_background_image"`
	ProfileTextColor          string `json:"profile_text_color"` // Hex color (w/o hashtag)
	Protected                 bool   `json:"protected"`
	FollowersCount            int    `json:"followers_count"`
	Lang                      string `json:"lang"`
	Notifications             bool   `json:"notifications"`
	TimeZone                  string `json:"time_zone"` // oh god it's in text form aaaa
	Verified                  bool   `json:"verified"`
	ProfileBackgroundColor    string `json:"profile_background_color"` // Hex color (w/o hashtag)
	GeoEnabled                bool   `json:"geo_enabled"`              // No clue what this does
	Description               string `json:"description"`
	FriendsCount              int    `json:"friends_count"`
	StatusesCount             int    `json:"statuses_count"`
	ProfileBackgroundImageURL string `json:"profile_background_image_url"`
	Following                 bool   `json:"following"`
	ScreenName                string `json:"screen_name"`
}

func InitServer() {
	app := fiber.New()

	// Initialize default config
	app.Use(logger.New())

	// Custom middleware to log request details
	app.Use(func(c *fiber.Ctx) error {
		fmt.Println("Request Method:", c.Method())
		fmt.Println("Request URL:", c.OriginalURL())
		fmt.Println("Post Body:", string(c.Body()))
		return c.Next()
	})

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

	app.Listen(":3000")
}

// https://developer.x.com/en/docs/authentication/api-reference/access_token
func access_token(c *fiber.Ctx) error {
	// Parse the form data
	sendErrorCodes := c.FormValue("send_error_codes")
	authMode := c.FormValue("x_auth_mode")
	authPassword := c.FormValue("x_auth_password")
	authUsername := c.FormValue("x_auth_username")

	fmt.Println("SendErrorCodes:", sendErrorCodes)
	fmt.Println("AuthMode:", authMode)
	fmt.Println("AuthPassword:", authPassword)
	fmt.Println("AuthUsername:", authUsername)

	if authMode == "client_auth" {
		res, err := blueskyapi.Authenticate(authUsername, authPassword)
		if err != nil {
			fmt.Println("Error:", err)
			return c.SendStatus(401)
		}
		return c.SendString(fmt.Sprintf("oauth_token=%s&oauth_token_secret=%s&user_id=%s&screen_name=twitterapi", res.OAuthToken, res.OAuthTokenSecret, res.UserID))
	}
	// This is a problem from when I actually get this connected to bluesky
	return c.SendStatus(501)
}

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/post/statuses/update
func status_update(c *fiber.Ctx) error {
	status := c.FormValue("status")
	trim_user := c.FormValue("trim_user")
	in_reply_to_status_id := c.FormValue("in_reply_to_status_id")

	fmt.Println("Status:", status)
	fmt.Println("TrimUser:", trim_user)
	fmt.Println("InReplyToStatusID:", in_reply_to_status_id)

	// TODO: Implement this

	return c.SendString("Not implemented")
}

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/get/statuses/home_timeline
func home_timeline(c *fiber.Ctx) error {

	return c.JSON([]Tweet{
		{
			Coordinates:     nil,
			Favourited:      false,
			CreatedAt:       "Wed Sep 01 00:00:00 +0000 2021",
			Truncated:       false,
			Text:            "Will it blow up if there's a @ in the name",
			Annotations:     nil,
			Contributors:    nil,
			ID:              4,
			Geo:             nil,
			Place:           nil,
			InReplyToUserID: 0,
			User: TweetUser{
				Name:                      "Preloading",
				ProfileSidebarBorderColor: "C0DEED",
				ProfileBackgroundTile:     false,
				ProfileSidebarFillColor:   "DDEEF6",
				CreatedAt:                 "Wed Sep 01 00:00:00 +0000 2021",
				ProfileImageURL:           "https://cdn.bsky.app/img/avatar_thumbnail/plain/did:plc:khcyntihpu7snjszuojjgjc4/bafkreignfoswre6f2ehujkifewpk2xdlrqhfhraloaoixjf5dommpzjxeq@png",
				Location:                  "San Francisco",
				ProfileLinkColor:          "0084B4",
				FollowRequestSent:         false,
				URL:                       "http://dev.twitter.com",
				FavouritesCount:           8,
				ContributorsEnabled:       false,
				UtcOffset:                 -28800,
				ID:                        2,
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
				Description:               "A developer just looking to make some cool stuff",
				FriendsCount:              100,
				StatusesCount:             333,
				ProfileBackgroundImageURL: "http://a0.twimg.com/images/themes/theme1/bg.png",
				Following:                 false,
				ScreenName:                "preloading",
			},
			Source: "web",
		},
	})

}

func user_info(c *fiber.Ctx) error {
	screen_name := c.Params("screen_name")
	user := TweetUser{
		Name:                      "Preloading",
		ProfileSidebarBorderColor: "C0DEED",
		ProfileBackgroundTile:     false,
		ProfileSidebarFillColor:   "DDEEF6",
		CreatedAt:                 "Wed Sep 01 00:00:00 +0000 2021",
		ProfileImageURL:           "https://cdn.bsky.app/img/avatar_thumbnail/plain/did:plc:khcyntihpu7snjszuojjgjc4/bafkreignfoswre6f2ehujkifewpk2xdlrqhfhraloaoixjf5dommpzjxeq@png",
		Location:                  "San Francisco",
		ProfileLinkColor:          "0084B4",
		FollowRequestSent:         false,
		URL:                       "http://dev.twitter.com",
		FavouritesCount:           8,
		ContributorsEnabled:       false,
		UtcOffset:                 -28800,
		ID:                        2,
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
		Description:               "A developer just looking to make some cool stuff",
		FriendsCount:              100,
		StatusesCount:             333,
		ProfileBackgroundImageURL: "http://a0.twimg.com/images/themes/theme1/bg.png",
		Following:                 false,
		ScreenName:                screen_name,
	}

	fmt.Println("ScreenName:", screen_name)
	return c.XML(user)
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
