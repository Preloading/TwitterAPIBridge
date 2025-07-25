package twitterv1

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
)

var (
	configData *config.Config
)

func InitServer(config *config.Config) {
	configData = config
	blueskyapi.InitConfig(configData)
	engine := html.New("./static", ".html")
	app := fiber.New(fiber.Config{
		//DisablePreParseMultipartForm: true,
		ProxyHeader: func() string {
			if configData.UseXForwardedFor {
				return fiber.HeaderXForwardedFor
			}
			return ""
		}(),
		Views: engine,
	})

	// Initialize default config
	app.Use(logger.New())

	// Custom middleware to log request details
	if config.DeveloperMode {
		app.Use(func(c *fiber.Ctx) error {
			// fmt.Println("Request Method:", c.Method())
			fmt.Println("Request URL:", c.OriginalURL())
			// fmt.Println("Post Body:", string(c.Body()))
			// fmt.Println("Headers:", string(c.Request().Header.Header()))
			// fmt.Println()
			return c.Next()
		})
	}

	// app.Get("/", func(c *fiber.Ctx) error {
	// 	return c.SendString("Hello, World!")
	// Serve static files from the "static" folder
	app.Static("/favicon.ico", "./static/favicon.ico")
	app.Static("/robots.txt", "./static/robots.txt")
	app.Static("/static", "./static")

	// Serve /
	app.Get("/", func(c *fiber.Ctx) error {
		// Render index within layouts/nested/main within layouts/nested/base
		return c.Render("index", fiber.Map{
			"DeveloperMode": config.DeveloperMode,
			"NotConfigured": configData.CdnURL == "http://127.0.0.1:3000",
			"PrefixedURL": func() string {
				if c.Hostname() == "twitterbridge.loganserver.net" {
					return "https://twb.preloading.dev"
				}
				if c.Hostname() == "testtwitterbridge.loganserver.net" {
					return "https://ttwb.preloading.dev"
				}
				return "https://" + c.Hostname()
			},
			"UnPrefixedURL": func() string {
				if c.Hostname() == "twitterbridge.loganserver.net" {
					return "twb.preloading.dev"
				}
				if c.Hostname() == "testtwitterbridge.loganserver.net" {
					return "ttwb.preloading.dev"
				}
				return c.Hostname()
			},
			"Version": config.Version,
		}, "index")
	})

	// Auth
	app.Post("/oauth/access_token", access_token)
	app.Get("/1/account/verify_credentials.:filetype", VerifyCredentials)
	app.Get("/account/verify_credentials.:filetype", VerifyCredentials)

	// Tweeting
	app.Post("/1/statuses/update.:filetype", status_update)
	app.Post("/1/statuses/update_with_media.:filetype", status_update_with_media)

	// Interactions
	app.Post("/1/statuses/retweet/:id.:filetype", retweet)
	app.Post("/1/favorites/create/:id.:filetype", favourite)
	app.Post("/1/favorites/destroy/:id.:filetype", Unfavourite)
	app.Post("/1/statuses/destroy/:id.:filetype", DeleteTweet)

	// Posts
	app.Get("/1/statuses/home_timeline.:filetype", home_timeline)
	app.Get("/1/statuses/user_timeline.:filetype", user_timeline)
	app.Get("/1/statuses/mentions.:filetype", mentions_timeline)
	app.Get("/1/favorites/toptweets.:filetype", hot_post_timeline)
	app.Get("/1/statuses/media_timeline.:filetype", media_timeline)
	app.Get("/1/statuses/show/:id.:filetype", GetStatusFromId)
	app.Get("/i/statuses/:id/activity/summary.:filetype", TweetInfo)
	app.Get("/1/related_results/show/:id.:filetype", RelatedResults)

	// Users
	app.Get("/1/users/show.:filetype", user_info)
	app.Get("/1/users/show/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		lastDotIndex := strings.LastIndex(path, ".")

		if lastDotIndex == -1 {
			// No file extension found
			c.Locals("handle", path)
			c.Locals("filetype", "json") // Default to JSON
		} else {
			c.Locals("handle", path[:lastDotIndex])
			c.Locals("filetype", path[lastDotIndex+1:])
		}

		return user_info(c)
	})
	app.Get("/1/users/lookup.:filetype", UsersLookup)
	app.Post("/1/users/lookup.:filetype", UsersLookup)
	app.Get("/1/friendships/lookup.:filetype", UserRelationships)
	app.Get("/1/friendships/show.:filetype", GetUsersRelationship)
	app.Get("/1/favorites/:id.:filetype", likes_timeline)
	app.Post("/1/friendships/create.:filetype", FollowUser)
	app.Post("/1/friendships/destroy.:filetype", UnfollowUserForm)
	app.Post("/1/friendships/destroy/:id.:filetype", UnfollowUserParams)
	app.Get("/1/followers.:filetype", GetFollowers)
	app.Get("/1/friends.:filetype", GetFollows)
	app.Get("/1/statuses/followers.:filetype", GetStatusesFollowers)
	app.Get("/1/statuses/friends.:filetype", GetStatusesFollows)

	app.Get("/1/users/recommendations.:filetype", GetSuggestedUsers)
	app.Get("/1/users/profile_image", UserProfileImage)

	// Connect
	app.Get("/1/users/search.:filetype", UserSearch)
	app.Get("/i/search/typeahead.:filetype", SearchAhead)
	app.Get("/i/activity/about_me.:filetype", GetMyActivity)

	// Discover
	app.Get("/1/trends/:woeid.:filetype", trends_woeid)
	app.Get("/1/trends/current.:filetype", trends_woeid)
	app.Get("/1/users/suggestions.:filetype", SuggestedTopics)
	app.Get("/1/users/suggestions/:slug.:filetype", GetTopicSuggestedUsers)
	app.Get("/i/search.:filetype", InternalSearch)
	app.Get("/i/discovery.:filetype", discovery)

	// Lists
	app.Get("/1/lists.:filetype", GetUsersLists)
	app.Get("/1/:user/lists.:filetype", GetUsersLists)
	app.Get("/1/lists/statuses.:filetype", list_timeline)
	app.Get("/1/:user/lists/:slug/statuses.:filetype", list_timeline)
	app.Get("/1/lists/members.:filetype", GetListMembers)
	app.Get("/1/:user/:list/members.:filetype", GetListMembers)

	app.Get("/1/lists/subscriptions.:filetype", GetUsersLists)       // This doesn't actually exist on bluesky, but here's something similar enough. Lists made by you.
	app.Get("/1/:user/lists/subscriptions.:filetype", GetUsersLists) // Well, if i'm to get technical, you can subscribe to moderation lists, but not the lists this expects.

	// Account / Settings
	app.Post("/1/account/update_profile.:filetype", UpdateProfile)
	app.Post("/1/account/update_profile_image.:filetype", UpdateProfilePicture)
	app.Get("/1/account/settings.:filetype", GetSettings)
	app.Get("/1/account/push_destinations/device.:filetype", PushDestinations)

	// Legal cuz why not?
	app.Get("/1/legal/tos.:filetype", TOS)
	app.Get("/1/legal/privacy.:filetype", PrivacyPolicy)

	// CDN Downscaler
	app.Get("/cdn/img", CDNDownscaler)
	app.Get("/cdn/img/bsky/:did/:link", CDNDownscaler)
	app.Get("/cdn/img/bsky/:did/:link.:filetype", CDNDownscaler)
	app.Get("/cdn/vid/bsky/:did/:link", CDNVideoProxy)
	app.Get("/cdn/img/bsky/:did/:link/:size", CDNDownscaler)

	// Shortcut
	app.Get("/img/:ref", RedirectToLink)

	// misc
	app.Get("/mobile_client_api/decider/:path", MobileClientApiDecider)

	app.Listen(fmt.Sprintf(":%d", config.ServerPort))
}

// misc
func MobileClientApiDecider(c *fiber.Ctx) error {
	return c.SendString("")
}

func EncodeAndSend(c *fiber.Ctx, data interface{}) error {
	encodeType := c.Params("filetype")
	if encodeType == "" {
		encodeType = c.Locals("filetype").(string)
	}
	switch encodeType {
	case "xml":
		// Encode the data to XML
		var buf bytes.Buffer
		enc := xml.NewEncoder(&buf)
		enc.Indent("", "  ")
		if err := enc.Encode(data); err != nil {
			fmt.Println("Error encoding XML:", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode into XML!")
		}

		// Add custom XML header
		xmlContent := buf.Bytes()
		customHeader := []byte(`<?xml version="1.0" encoding="UTF-8"?>`)
		xmlContent = append(customHeader, xmlContent...)

		return c.SendString(string(xmlContent))
	case "json", "":
		encoded, err := json.Marshal(data)
		if err != nil {
			fmt.Println("Error:", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode into json!")
		}
		return c.SendString(string(encoded))
	default:
		return c.Status(fiber.StatusInternalServerError).SendString("Invalid file type!")
	}

}
