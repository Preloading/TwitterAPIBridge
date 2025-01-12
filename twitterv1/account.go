package twitterv1

import (
	"fmt"
	"io"
	"strings"
	"sync"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/gofiber/fiber/v2"
)

// Mutex map to store mutexes for each user
var userMutexes = make(map[string]*sync.Mutex)
var mutexMapLock sync.Mutex

func getUserMutex(userID string) *sync.Mutex {
	mutexMapLock.Lock()
	defer mutexMapLock.Unlock()
	if _, exists := userMutexes[userID]; !exists {
		userMutexes[userID] = &sync.Mutex{}
	}
	return userMutexes[userID]
}

func PushDestinations(c *fiber.Ctx) error {
	// TODO: figure out what the hell this is supposed to do to make notifications not crash.
	old_udid := c.Query("old_udid")
	udid := c.Query("udid")
	// environment := c.Query("environment")

	// 	app_version=4.1.3&
	// device_model=iPhone&
	// device_name=
	// enabled_for=23&
	// environment=3&
	// language=en&
	// old_udid=d89b164326e0c50494438d5bd360988c53e672f0&
	// send_error_codes=true&
	// system_name=iPhone%20OS&
	// system_version=4.2.1&
	// token=
	// udid=291C3725-6221-4B96-A897-3436AE9D48DF

	return c.SendString(fmt.Sprintf(`
	<?xml version="1.0" encoding="UTF-8"?>
	<device>
		<device_model>iPhone</device_model>
		<device_name>iPhone</device_name>
		<enabled_for>23</enabled_for>
		<language>en</language>
		<send_error_codes>true</send_error_codes>
		<system_name>iPhone OS</system_name>
		<system_version>4.2.1</system_version>
		<token>i dunno, guess who has no idea what this returns./token>
		<udid>%s</udid>
		<old_udid>%s</old_udid>
		<environment>3</environment>
	<device>
	`, udid, old_udid))
}

// TODO
func GetSettings(c *fiber.Ctx) error {
	settings := bridge.Config{
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
	}
	return EncodeAndSend(c, settings)

}

func UpdateProfile(c *fiber.Ctx) error {
	// auth
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Lock the mutex for this user
	userMutex := getUserMutex(*my_did)
	userMutex.Lock()
	defer userMutex.Unlock()

	description := c.FormValue("description")
	name := c.FormValue("name")
	// These don't exist in bluesky.
	// location := c.FormValue("location")
	// url := c.FormValue("url")

	// some quality of life features
	description = strings.ReplaceAll(description, "\\n", "\n")

	oldProfile, err := blueskyapi.GetRecord(*pds, "app.bsky.actor.profile", *my_did, "self")
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get profile")
	}

	oldProfile.Value.DisplayName = name
	oldProfile.Value.Description = description

	if err := blueskyapi.UpdateRecord(*pds, *oauthToken, "app.bsky.actor.profile", *my_did, "self", oldProfile.CID, oldProfile.Value); err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update profile")
	}

	user, err := blueskyapi.GetUserInfo(*pds, *oauthToken, *my_did, true)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	user.Description = description
	user.Name = name

	return EncodeAndSend(c, user)
}

func UpdateProfilePicture(c *fiber.Ctx) error {
	// auth
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Lock the mutex for this user
	userMutex := getUserMutex(*my_did)
	userMutex.Lock()
	defer userMutex.Unlock()

	// get the old profile
	oldProfile, err := blueskyapi.GetRecord(*pds, "app.bsky.actor.profile", *my_did, "self")
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get profile")
	}

	// get our new image
	image, err := c.FormFile("image")
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusBadRequest).SendString("Please upload an image")
	}

	// read the image file content
	file, err := image.Open()
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to open image file")
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read image file")
	}

	// upload our new profile picture
	profilePictureBlob, err := blueskyapi.UploadBlob(*pds, *oauthToken, imageData, c.Get("Content-Type"))
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to upload profile picture")
	}

	// change our thing
	oldProfile.Value.Avatar = *profilePictureBlob

	if err := blueskyapi.UpdateRecord(*pds, *oauthToken, "app.bsky.actor.profile", *my_did, "self", oldProfile.CID, oldProfile.Value); err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update profile")
	}

	user, err := blueskyapi.GetUserInfo(*pds, *oauthToken, *my_did, true)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	return EncodeAndSend(c, user)
}
