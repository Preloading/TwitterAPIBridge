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

func DevicePushDestinations(c *fiber.Ctx) error {
	return EncodeAndSend(c, bridge.PushDestination{
		AvailableLevels: 0b1111111111, // I have no idea what any of this means.
		EnabledFor:      5,
	})
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
		return MissingAuth(c, err)
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
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.getRecord", UpdateProfile)
	}

	oldProfile.Value.DisplayName = name
	oldProfile.Value.Description = description

	if err := blueskyapi.UpdateRecord(*pds, *oauthToken, "app.bsky.actor.profile", *my_did, "self", oldProfile.CID, oldProfile.Value); err != nil {
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.putRecord", UpdateProfile)
	}

	user, err := blueskyapi.GetUserInfo(*pds, *oauthToken, *my_did, true)
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfile", UpdateProfile)
	}

	user.Description = description
	user.Name = name

	return EncodeAndSend(c, user)
}

func UpdateProfilePicture(c *fiber.Ctx) error {
	// auth
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c, err)
	}

	// Lock the mutex for this user
	userMutex := getUserMutex(*my_did)
	userMutex.Lock()
	defer userMutex.Unlock()

	// get the old profile
	oldProfile, err := blueskyapi.GetRecord(*pds, "app.bsky.actor.profile", *my_did, "self")
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.getRecord", UpdateProfilePicture)
	}

	// get our new image
	image, err := c.FormFile("image")
	if err != nil {
		fmt.Println("Error:", err)
		return ReturnError(c, "Please upload an image", 195, 403) // idk about this error code, since it's for url params instead of post data.
	}

	// read the image file content
	file, err := image.Open()
	if err != nil {
		fmt.Println("Error:", err)
		return ReturnError(c, "Uploaded image is invalid.", 195, 403)
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error:", err)
		return ReturnError(c, "Uploaded image is invalid.", 195, 403)
	}

	// upload our new profile picture
	profilePictureBlob, err := blueskyapi.UploadBlob(*pds, *oauthToken, imageData, c.Get("Content-Type"))
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.uploadBlob", UpdateProfilePicture)
	}

	// change our thing
	oldProfile.Value.Avatar = *profilePictureBlob

	if err := blueskyapi.UpdateRecord(*pds, *oauthToken, "app.bsky.actor.profile", *my_did, "self", oldProfile.CID, oldProfile.Value); err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.putRecord", UpdateProfilePicture)
	}

	user, err := blueskyapi.GetUserInfo(*pds, *oauthToken, *my_did, true)
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfile", UpdateProfile)
	}

	return EncodeAndSend(c, user)
}
