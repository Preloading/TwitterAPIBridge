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
			}(),
			"UnPrefixedURL": func() string {
				if c.Hostname() == "twitterbridge.loganserver.net" {
					return "twb.preloading.dev"
				}
				if c.Hostname() == "testtwitterbridge.loganserver.net" {
					return "ttwb.preloading.dev"
				}
				return c.Hostname()
			}(),
			"Version": config.Version,
		}, "index")
	})

	// Auth
	app.Post("/oauth/access_token", access_token)
	AddV1Path(app.Get, "/account/verify_credentials.:filetype", VerifyCredentials)

	// OAUTH
	app.Post("/oauth/request_token", RequestToken)

	// Tweeting
	AddV1Path(app.Post, "/statuses/update.:filetype", status_update)
	AddV1Path(app.Post, "/statuses/update_with_media.:filetype", status_update_with_media)

	// Interactions
	AddV1Path(app.Post, "/statuses/retweet/:id.:filetype", retweet)
	AddV1Path(app.Post, "/favorites/create/:id.:filetype", favourite)
	AddV1Path(app.Post, "/favorites/destroy/:id.:filetype", Unfavourite)
	AddV1Path(app.Post, "/statuses/destroy/:id.:filetype", DeleteTweet)

	// Posts
	AddV1Path(app.Get, "/statuses/home_timeline.:filetype", home_timeline)
	AddV1Path(app.Get, "/statuses/user_timeline.:filetype", user_timeline)
	AddV1Path(app.Get, "/statuses/mentions.:filetype", mentions_timeline)
	AddV1Path(app.Get, "/favorites/toptweets.:filetype", hot_post_timeline)
	AddV1Path(app.Get, "/tatuses/media_timeline.:filetype", media_timeline)
	AddV1Path(app.Get, "/statuses/show/:id.:filetype", GetStatusFromId)
	app.Get("/i/statuses/:id/activity/summary.:filetype", TweetInfo)
	AddV1Path(app.Get, "/related_results/show/:id.:filetype", RelatedResults)

	// Users
	AddV1Path(app.Get, "/users/show.:filetype", user_info)
	AddV1Path(app.Get, "/users/show/*", func(c *fiber.Ctx) error {
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
	AddV1Path(app.Get, "/users/lookup.:filetype", UsersLookup)
	AddV1Path(app.Post, "/users/lookup.:filetype", UsersLookup)
	AddV1Path(app.Get, "/friendships/lookup.:filetype", UserRelationships)
	AddV1Path(app.Get, "/friendships/show.:filetype", GetUsersRelationship)
	AddV1Path(app.Get, "/favorites/:id.:filetype", likes_timeline)
	AddV1Path(app.Post, "/friendships/create.:filetype", FollowUser)
	AddV1Path(app.Post, "/friendships/destroy.:filetype", UnfollowUserForm)
	AddV1Path(app.Post, "/friendships/destroy/:id.:filetype", UnfollowUserParams)
	AddV1Path(app.Get, "/followers.:filetype", GetFollowers)
	AddV1Path(app.Get, "/friends.:filetype", GetFollows)
	AddV1Path(app.Get, "/statuses/followers.:filetype", GetStatusesFollowers)
	AddV1Path(app.Get, "/statuses/friends.:filetype", GetStatusesFollows)
	app.Get("/i/device_following/ids.:filetype", GetFollowingIds)

	AddV1Path(app.Get, "/users/recommendations.:filetype", GetSuggestedUsers)
	AddV1Path(app.Get, "/users/profile_image", UserProfileImage)

	// Connect
	AddV1Path(app.Get, "/users/search.:filetype", UserSearch)
	app.Get("/i/search/typeahead.:filetype", SearchAhead)
	app.Get("/i/activity/about_me.:filetype", GetMyActivity)

	// Discover
	AddV1Path(app.Get, "/trends/:woeid.:filetype", trends_woeid)
	AddV1Path(app.Get, "/trends/current.:filetype", trends_woeid)
	AddV1Path(app.Get, "/users/suggestions.:filetype", SuggestedTopics)
	AddV1Path(app.Get, "/users/suggestions/:slug.:filetype", GetTopicSuggestedUsers)
	app.Get("/i/search.:filetype", InternalSearch)
	app.Get("/i/discovery.:filetype", discovery)

	// Lists
	AddV1Path(app.Get, "/lists.:filetype", GetUsersLists)
	AddV1Path(app.Get, "/:user/lists.:filetype", GetUsersLists)
	AddV1Path(app.Get, "/lists/statuses.:filetype", list_timeline)
	AddV1Path(app.Get, "/:user/lists/:slug/statuses.:filetype", list_timeline)
	AddV1Path(app.Get, "/lists/members.:filetype", GetListMembers)
	AddV1Path(app.Get, "/:user/:list/members.:filetype", GetListMembers)

	AddV1Path(app.Get, "/lists/subscriptions.:filetype", GetUsersLists)       // This doesn't actually exist on bluesky, but here's something similar enough. Lists made by you.
	AddV1Path(app.Get, "/:user/lists/subscriptions.:filetype", GetUsersLists) // Well, if i'm to get technical, you can subscribe to moderation lists, but not the lists this expects.

	// Account / Settings
	AddV1Path(app.Post, "/account/update_profile.:filetype", UpdateProfile)
	AddV1Path(app.Post, "/account/update_profile_image.:filetype", UpdateProfilePicture)
	AddV1Path(app.Get, "/account/settings.:filetype", GetSettings)

	// Push Notifications
	AddV1Path(app.Get, "/account/push_destinations/device.:filetype", DevicePushDestinations)
	AddV1Path(app.Post, "/account/push_destinations.:filetype", UpdatePushNotifications)
	AddV1Path(app.Post, "/account/push_destinations/destroy.:filetype", RemovePush)

	// Legal cuz why not?
	AddV1Path(app.Get, "/legal/tos.:filetype", TOS)
	AddV1Path(app.Get, "/legal/privacy.:filetype", PrivacyPolicy)

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

	go cleanupTempTokens()
	app.Listen(fmt.Sprintf(":%d", config.ServerPort))
}

func AddV1Path(function func(string, ...fiber.Handler) fiber.Router, url string, handler fiber.Handler) {
	function(url, handler)
	function(fmt.Sprintf("/1%s", url), handler)
}

// misc
func MobileClientApiDecider(c *fiber.Ctx) error {
	return c.SendString("") // todo maybe?
}

func EncodeAndSend(c *fiber.Ctx, data interface{}) error {
	var ok bool
	encodeType := c.Params("filetype")
	if encodeType == "" {
		encodeType, ok = c.Locals("filetype").(string)
		if !ok {
			encodeType = "json"
		}
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
