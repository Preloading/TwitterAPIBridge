package twitterv1

import (
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	skyglownotificationlib "github.com/Preloading/SkyglowNotificationLibraries"
	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/Preloading/TwitterAPIBridge/db_controller"
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
	my_did, _, _, _, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c, err)
	}

	if configData.NotificationTrustedServer == "" {
		return ReturnError(c, "push notifications are disabled on this server", 1000, 404)
	}

	notificationTokens, err := db_controller.GetPushTokensForDID(*my_did)
	if err != nil {
		return EncodeAndSend(c, bridge.PushDestination{
			AvailableLevels: 1021,
			EnabledFor:      340,
		})
	}

	if !(len(notificationTokens) > 0) {
		return EncodeAndSend(c, bridge.PushDestination{
			AvailableLevels: 1021,
			EnabledFor:      340,
		})
	}

	return EncodeAndSend(c, bridge.PushDestination{
		AvailableLevels: 1021, // Idk what this means

		// EnabledFor is a "binary" format
		// 9: tweets
		// 8: retweets: from anyone
		// 7: retweets: from people you follow
		// 6: favourites: from anyone
		// 5: favourites: from people you follow
		// 4: new followers
		// 3: mentions: from anyone
		// 2: mentions: from people you follow
		// 1: ?
		// 0: direct messages
		EnabledFor: notificationTokens[0].EnabledFor,
	})
}

// https://gist.github.com/ZweiSteinSoft/4733612
func UpdatePushNotifications(c *fiber.Ctx) error {
	// auth
	my_did, _, _, _, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c, err)
	}

	if configData.NotificationTrustedServer == "" {
		return ReturnError(c, "push notifications are disabled on this server", 1000, 404)
	}

	enabledFor, err := strconv.Atoi(c.FormValue("enabled_for"))
	if err != nil {
		return ReturnError(c, "enabled_for is missing", 0, 400)
	}

	fmt.Println(c.FormValue("token"))

	// device token
	notificationToken := make([]byte, 32)
	_, err = base64.StdEncoding.Decode(notificationToken, []byte(c.FormValue("token")))
	if err != nil {
		fmt.Println(err.Error())
		return ReturnError(c, "device token is invalid", 0, 400)
	}

	routing_key, routing_server_address, err := skyglownotificationlib.RoutingInfoFromDeviceToken(notificationToken)

	if err != nil {
		fmt.Println(err.Error())
		// user is probably not using SGN
		return ReturnError(c, "Skyglow Notifications is required for notifications", 1000, 404)
	}

	db_controller.CreateModifyRegisteredPushNotifications(db_controller.NotificationTokens{
		UserDID:       *my_did,
		DeviceToken:   notificationToken,
		RoutingKey:    routing_key,
		ServerAddress: *routing_server_address,
		EnabledFor:    enabledFor,
		LastUpdated:   time.Now(),
	})

	return EncodeAndSend(c, bridge.PushDestination{
		AvailableLevels: 1021, // Idk what this means
		// EnabledFor is a "binary" format
		// 9: tweets
		// 8: retweets: from anyone
		// 7: retweets: from people you follow
		// 6: favourites: from anyone
		// 5: favourites: from people you follow
		// 4: new followers
		// 3: mentions: from anyone
		// 2: mentions: from people you follow
		// 1: ?
		// 0: direct messages
		EnabledFor: enabledFor,
	})
}

// this should probably use the udid
func RemovePush(c *fiber.Ctx) error {
	// auth
	my_did, _, _, _, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c, err)
	}

	if configData.NotificationTrustedServer == "" {
		return ReturnError(c, "push notifications are disabled on this server", 1000, 404)
	}

	err = db_controller.DeleteeeeeeeeeeeeRegistrationForPushNotificationsWithDid(*my_did)

	if err != nil {
		fmt.Println(err.Error())
		return ReturnError(c, "something went wrong when deregistering you or smth idk and i dont care", 131, 500)
	}

	return EncodeAndSend(c, bridge.PushDestination{
		AvailableLevels: 0,
		EnabledFor:      0,
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
