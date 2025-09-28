package notifications

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	sgn "github.com/Preloading/SkyglowNotificationLibraries"
	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/db_controller"
	"github.com/Preloading/TwitterAPIBridge/twitterv1"
	"golang.org/x/net/websocket"
)

func getBit(n int, pos uint) bool {
	// Left shift 1 to the 'pos' position to create a bitmask.
	// Then, perform a bitwise AND with the number 'n'.
	// If the result is non-zero, the bit at 'pos' is set (1); otherwise, it's unset (0).
	if (n & (1 << pos)) != 0 {
		return true
	}
	return false
}

type JetstreamCommit struct {
	Operation  string
	RKey       string `json:"rkey"`
	Record     json.RawMessage
	Collection string
}

// Tailored for post because lazy
type JetstreamPostOutput struct {
	DID string `json:"did"`
	// Not all this data is needed, and it speeds up json decodes :)
	TimeUS int64 `json:"time_us"`
	// Kind   string `json:"kind"`
	Commit JetstreamCommit `json:"commit"`
}

var (
	mentionedDIDs               []string
	mentionedFollowingOnlyDIDs  []string
	retweetDIDs                 []string
	retweetFollowingOnlyDIDs    []string
	favouritesDIDs              []string
	favouritesFollowingOnlyDIDs []string
	newFollowers                []string
	// posterDIDs                  []string
	lastUpdatedNotificationTime time.Time
)

func RunNotifications() {
	sgn.ConfigureSession("d.preloading.dev") // todo make this configureable

	incomingMessages := make(chan JetstreamPostOutput)

	go func() {
		for {
			// this updates which users have notififcations registered and what they have registered with.
			latestPushNotificationRound, err := db_controller.GetAllActivePushNotifications()
			if err != nil {
				panic(err)
			}

			var (
				tMentionedDIDs               []string
				tMentionedFollowingOnlyDIDs  []string
				tRetweetDIDs                 []string
				tRetweetFollowingOnlyDIDs    []string
				tFavouritesDIDs              []string
				tFavouritesFollowingOnlyDIDs []string
				tNewFollowers                []string
				// tPosterDIDs                  []string
			)

			for _, notificationToSplit := range latestPushNotificationRound {
				e := notificationToSplit.EnabledFor
				if getBit(e, 2) {
					tMentionedFollowingOnlyDIDs = append(tMentionedFollowingOnlyDIDs, notificationToSplit.UserDID)
				}
				if getBit(e, 3) {
					tMentionedDIDs = append(tMentionedDIDs, notificationToSplit.UserDID)
				}
				if getBit(e, 4) {
					tNewFollowers = append(tNewFollowers, notificationToSplit.UserDID)
				}
				if getBit(e, 5) {
					tFavouritesFollowingOnlyDIDs = append(tFavouritesFollowingOnlyDIDs, notificationToSplit.UserDID)
				}
				if getBit(e, 6) {
					tFavouritesDIDs = append(tFavouritesDIDs, notificationToSplit.UserDID)
				}
				if getBit(e, 7) {
					tRetweetFollowingOnlyDIDs = append(tRetweetFollowingOnlyDIDs, notificationToSplit.UserDID)
				}
				if getBit(e, 8) {
					tRetweetDIDs = append(tRetweetDIDs, notificationToSplit.UserDID)
				}
				// if getBit(e, 9) { // this one is a weird case, it's for notitifcations when other people post
				// 	tFavouritesFollowingOnlyDIDs = append(tFavouritesFollowingOnlyDIDs, notificationToSplit.UserDID)
				// }
			}
			mentionedDIDs = tMentionedDIDs
			mentionedFollowingOnlyDIDs = tMentionedFollowingOnlyDIDs
			retweetDIDs = tRetweetDIDs
			retweetFollowingOnlyDIDs = tRetweetFollowingOnlyDIDs
			favouritesDIDs = tFavouritesDIDs
			favouritesFollowingOnlyDIDs = tFavouritesFollowingOnlyDIDs
			newFollowers = tNewFollowers
			// posterDIDs =
			lastUpdatedNotificationTime = time.Now()
			for time.Since(lastUpdatedNotificationTime) < 5*time.Minute {
				time.Sleep(10 * time.Second)
			}
		}
	}()

	go func() {
		for {
			ws, err := websocket.Dial("wss://jetstream1.us-east.bsky.network/subscribe?wantedCollections=app.bsky.feed.post&wantedCollections=app.bsky.feed.like&wantedCollections=app.bsky.feed.repost&wantedCollections=app.bsky.graph.follow", "", "https://jetstream1.us-east.bsky.network/subscribe?wantedCollections=app.bsky.feed.post")
			if err != nil {
				fmt.Printf("Jetstream dial failed: %v\n", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// read until error / disconnect; when readJetstreamMessages returns, close and retry
			readJetstreamMessages(ws, incomingMessages)
			if err := ws.Close(); err != nil {
				fmt.Printf("Error closing Jetstream ws: %v\n", err)
			}
			fmt.Println("Jetstream disconnected; reconnecting in 3s")
			time.Sleep(3 * time.Second)
		}
	}()

	for message := range incomingMessages {
		if message.Commit.Operation != "create" {
			continue
		}
		switch message.Commit.Collection {
		case "app.bsky.feed.post":
			{
				// if slices.Contains(posterDIDs, message.DID) {
				// 	fmt.Printf("New Jetstream Message (user): %+v\n", message)
				// }

				var record blueskyapi.PostRecord
				if err := json.Unmarshal(message.Commit.Record, &record); err != nil {
					continue
				}

				for _, facet := range record.Facets {
					for _, feature := range facet.Features {
						if feature.Type == "app.bsky.richtext.facet#mention" {
							// its got a mention

							// check if the mention is in our list of people subscribed to mention notifications
							if slices.Contains(mentionedDIDs, feature.Did) {
								fmt.Printf("New Jetstream Message (mention): %+v\n", record)
								go sendPushNotificationForPost(feature.Did, "mention", message.DID, message.Commit.RKey, nil)
							}

							// now lets do the follower check
							if slices.Contains(mentionedFollowingOnlyDIDs, feature.Did) {
								fmt.Printf("New Jetstream Message (mentioned following): %+v\n", record)
								go sendPushNotificationForPost(feature.Did, "mention_following", message.DID, message.Commit.RKey, nil)
							}
						}
					}
				}
			}
		case "app.bsky.feed.like":
			{
				var record blueskyapi.ProperSubjectInteractionRecord
				if err := json.Unmarshal(message.Commit.Record, &record); err != nil {
					continue
				}

				// at://did:plc:ce3lui3j4c3l7bya6xwahcrn/app.bsky.feed.post/3lzoxzsh4p22e
				splitURI := strings.Split(record.Subject.URI, "/")

				didOfPoster := splitURI[2]

				if slices.Contains(favouritesDIDs, didOfPoster) {
					fmt.Printf("New Jetstream Message (liked): %+v\n", record)
					go sendPushNotificationForPost(didOfPoster, "liked", message.DID, splitURI[4], nil)
				}

				if slices.Contains(favouritesFollowingOnlyDIDs, didOfPoster) {
					fmt.Printf("New Jetstream Message (liked_following): %+v\n", record)
					go sendPushNotificationForPost(didOfPoster, "liked_following", message.DID, splitURI[4], nil)
				}

			}
		case "app.bsky.feed.repost":
			{
				var record blueskyapi.ProperSubjectInteractionRecord
				if err := json.Unmarshal(message.Commit.Record, &record); err != nil {
					continue
				}

				splitURI := strings.Split(record.Subject.URI, "/")

				didOfPoster := splitURI[2]

				if slices.Contains(retweetDIDs, didOfPoster) {
					fmt.Printf("New Jetstream Message (retweet): %+v\n", record)
					go sendPushNotificationForPost(didOfPoster, "retweet", message.DID, splitURI[4], nil)
				}

				if slices.Contains(retweetFollowingOnlyDIDs, didOfPoster) {
					fmt.Printf("New Jetstream Message (retweet_following): %+v\n", record)
					go sendPushNotificationForPost(didOfPoster, "retweet_following", message.DID, splitURI[4], nil)
				}

			}
		case "app.bsky.graph.follow":
			{
				var record blueskyapi.PostInteractionRecord
				if err := json.Unmarshal(message.Commit.Record, &record); err != nil {
					continue
				}

				subject, ok := record.Subject.(string)
				if !ok {
					continue
				}

				if slices.Contains(newFollowers, subject) {
					fmt.Printf("New Jetstream Message (retweet): %+v\n", record)
					go sendPushNotificationForPost(subject, "follow", message.DID, "", nil)
				}

			}
		}

	}

}

func readJetstreamMessages(ws *websocket.Conn, incomingMessages chan JetstreamPostOutput) {
	for {
		var message JetstreamPostOutput
		err := websocket.JSON.Receive(ws, &message)
		// err := websocket.Message.Receive(ws, &message)
		if err != nil {
			fmt.Printf("Error with notification stream: %s\n", err.Error())
			return
		}
		incomingMessages <- message
	}
}

// This function is quite a bit slower than our inital check, and it does the following:
// 1. ~~If it's a mention, verify that the user hasn't blocked~~ I dont think this is possible.
// 2. Get the device tokens of the devices that would like these specific push notifications.
// 3. Converting the text into a twitter post
// 4. Send the twitter post's content as a push notification via SGN.
func sendPushNotificationForPost(did string, typeOfNotification string, didOfPoster string, rkey string, indexed_at *int64) {
	notificationBody := map[string]interface{}{}
	// GetPost

	switch typeOfNotification {
	case "mention", "mention_following":
		{
			if typeOfNotification == "mention_following" {
				relationship, err := blueskyapi.GetRelationships("https://public.api.bsky.app", "", did, []string{didOfPoster})
				if err != nil {
					return
				}
				if !(len(relationship.Relationships) >= 1) {
					return
				}
				if relationship.Relationships[0].Following == "" {
					return
				}

			}
			bskyPost, err := blueskyapi.GetPost("https://public.api.bsky.app", "", fmt.Sprintf("at://%s/app.bsky.feed.post/%s", didOfPoster, rkey), 0, 0)
			if err != nil {
				return
			}

			tweet := twitterv1.TranslatePostToTweet(bskyPost.Thread.Post, "", "", "", nil, nil, "", "https://public.api.bsky.app")
			// our body
			notificationBody = map[string]interface{}{
				"aps": map[string]interface{}{
					"alert": fmt.Sprintf("Mentioned by @%s: %s", tweet.User.ScreenName, tweet.Text),
					"sound": "default",
				},
			}
		}
	case "liked", "liked_following":
		{
			if typeOfNotification == "liked_following" {
				relationship, err := blueskyapi.GetRelationships("https://public.api.bsky.app", "", did, []string{didOfPoster})
				if err != nil {
					return
				}
				if !(len(relationship.Relationships) >= 1) {
					return
				}
				if relationship.Relationships[0].Following == "" {
					return
				}

			}
			bskyPost, err := blueskyapi.GetPost("https://public.api.bsky.app", "", fmt.Sprintf("at://%s/app.bsky.feed.post/%s", did, rkey), 0, 0)
			if err != nil {
				return
			}

			bskyUser, err := blueskyapi.GetUserInfo("https://public.api.bsky.app", "", didOfPoster, false)
			if err != nil {
				return
			}

			tweet := twitterv1.TranslatePostToTweet(bskyPost.Thread.Post, "", "", "", nil, nil, "", "https://public.api.bsky.app")
			// our body
			notificationBody = map[string]interface{}{
				"aps": map[string]interface{}{
					"alert": fmt.Sprintf("@%s favourited your tweet: %s", bskyUser.ScreenName, tweet.Text), // idk what this is
					"sound": "default",
				},
			}
		}
	case "retweet", "retweet_following":
		{
			if typeOfNotification == "retweet_following" {
				relationship, err := blueskyapi.GetRelationships("https://public.api.bsky.app", "", did, []string{didOfPoster})
				if err != nil {
					return
				}
				if !(len(relationship.Relationships) >= 1) {
					return
				}
				if relationship.Relationships[0].Following == "" {
					return
				}

			}
			bskyPost, err := blueskyapi.GetPost("https://public.api.bsky.app", "", fmt.Sprintf("at://%s/app.bsky.feed.post/%s", did, rkey), 0, 0)
			if err != nil {
				return
			}

			bskyUser, err := blueskyapi.GetUserInfoRaw("https://public.api.bsky.app", "", didOfPoster)
			if err != nil {
				return
			}

			postReason := blueskyapi.PostReason{
				Type: "app.bsky.feed.defs#reasonRepost",
				By:   *bskyUser,
			}

			tweet := twitterv1.TranslatePostToTweet(bskyPost.Thread.Post, "", "", "", nil, &postReason, "", "https://public.api.bsky.app")

			// our body
			notificationBody = map[string]interface{}{
				"aps": map[string]interface{}{
					"alert": fmt.Sprintf("@%s: %s", tweet.User.ScreenName, tweet.Text),
					"sound": "default",
				},
			}
		}
	case "follow":
		{
			bskyUser, err := blueskyapi.GetUserInfo("https://public.api.bsky.app", "", didOfPoster, false)
			if err != nil {
				return
			}

			// our body
			notificationBody = map[string]interface{}{
				"aps": map[string]interface{}{
					"alert": fmt.Sprintf("@%s is now following you!", bskyUser.ScreenName),
					"sound": "default",
				},
			}
		}
	}

	pushTokens, err := db_controller.GetPushTokensForDID(did)
	if err != nil {
		return
	}
	for _, token := range pushTokens {
		if err := sgn.SendNotification(token.DeviceToken, notificationBody); err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("i just send a notification")
	}
}
