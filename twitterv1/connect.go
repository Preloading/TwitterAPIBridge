package twitterv1

import (
	"fmt"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
)

func UserSearch(c *fiber.Ctx) error {
	searchQuery := c.Query("q")
	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}
	// Search for users
	bskyUsers, err := blueskyapi.UserSearch(*pds, *oauthToken, searchQuery)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to search")
	}
	// Get complete user info.
	// We must do this as the search API only returns a subset of the user info, and twitter wants all of it.

	// Extract the dids into a string array
	var dids []string
	for _, user := range bskyUsers {
		dids = append(dids, user.DID)
	}
	if len(dids) == 0 {
		return c.JSON([]bridge.TwitterUser{})
	}
	users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, dids, false)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
	}

	return c.JSON(users)
}

// /i/activity/about_me.json?contributor_details=1&include_entities=true&include_my_retweet=true&send_error_codes=true
func GetMyActivity(c *fiber.Ctx) error {
	// Thank you so much @Savefade for what this returns for follows.
	// This function could probably optimized to use less GetUsers calls, but whatever.
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	bskyNotifcations, err := blueskyapi.GetNotifications(*pds, *oauthToken, 50, "")

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get notifications")
	}

	twitterNotifications := []bridge.MyActivity{}

	position := 0

	for position < len(bskyNotifcations.Notifications) {
		notification := bskyNotifcations.Notifications[position]
		switch notification.Reason {
		case "follow":
			usersInBlock := []string{notification.Author.DID}
			for position+1 < len(bskyNotifcations.Notifications) {
				position++
				if bskyNotifcations.Notifications[position].Reason == "follow" {
					usersInBlock = append(usersInBlock, bskyNotifcations.Notifications[position].Author.DID)
				} else {
					break
				}
			}
			users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, usersInBlock, false)
			if err != nil {
				fmt.Println("Error:", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
			}

			var sources []bridge.TwitterUser
			for _, user := range users {
				sources = append(sources, *user) // pain
			}

			twitterNotifications = append(twitterNotifications, bridge.MyActivity{
				Action:    "follow",
				CreatedAt: bridge.TwitterTimeConverter(notification.IndexedAt),
				ID:        *bridge.BlueSkyToTwitterID(notification.URI),
				Sources:   sources,
			})
		case "like":
			usersInBlock := []string{notification.Author.DID}
			notificationId := notification.URI
			for position+1 < len(bskyNotifcations.Notifications) {
				position++
				if bskyNotifcations.Notifications[position].Reason == "like" && bskyNotifcations.Notifications[position].ReasonSubject == notification.ReasonSubject {
					usersInBlock = append(usersInBlock, bskyNotifcations.Notifications[position].Author.DID)
					notificationId = bskyNotifcations.Notifications[position].URI
				} else {
					break
				}
			}
			// slight optimization in network traffic, saves us from having to call seperately for the poster
			_, poster_did, _ := blueskyapi.GetURIComponents(notification.ReasonSubject)
			usersInBlock = append(usersInBlock, poster_did)
			users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, usersInBlock, false)
			if err != nil {
				fmt.Println("Error:", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
			}
			users = users[:len(users)-1] // remove the last user, as it's the poster, and we don't need it here

			err, likedPost := blueskyapi.GetPost(*pds, *oauthToken, notification.ReasonSubject, 0, 1)
			if err != nil {
				fmt.Println("Error:", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to get post")
			}

			likedTweet := TranslatePostToTweet(likedPost.Thread.Post, "", "", nil, nil, *oauthToken, *pds)
			if likedPost.Thread.Parent != nil {
				likedTweet = TranslatePostToTweet(likedPost.Thread.Post, likedPost.Thread.Parent.Post.URI, likedPost.Thread.Parent.Post.Author.DID, &likedPost.Thread.Parent.Post.IndexedAt, nil, *oauthToken, *pds)
			}

			var sources []bridge.TwitterUser
			for _, user := range users {
				sources = append(sources, *user) // pain#
			}

			twitterNotifications = append(twitterNotifications, bridge.MyActivity{
				Action:    "favorite",
				CreatedAt: bridge.TwitterTimeConverter(notification.IndexedAt),
				ID:        *bridge.BlueSkyToTwitterID(notificationId),
				Sources:   sources,
				Targets:   []bridge.Tweet{likedTweet},
			})
		case "repost":
			usersInBlock := []string{notification.Author.DID}
			notificationId := notification.URI // We get the last URI so if we get a new repost ontop of this, it adds to to the prev, not a new one
			for position+1 < len(bskyNotifcations.Notifications)+1 {
				position++
				if bskyNotifcations.Notifications[position].Reason == "repost" && bskyNotifcations.Notifications[position].ReasonSubject == notification.ReasonSubject {
					usersInBlock = append(usersInBlock, bskyNotifcations.Notifications[position].Author.DID)
					notificationId = bskyNotifcations.Notifications[position].URI
				} else {
					break
				}
			}
			// slight optimization in network traffic, saves us from having to call seperately for the poster
			_, poster_did, _ := blueskyapi.GetURIComponents(notification.ReasonSubject)
			usersInBlock = append(usersInBlock, poster_did)
			users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, usersInBlock, false)
			if err != nil {
				fmt.Println("Error:", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
			}
			users = users[:len(users)-1] // remove the last user, as it's the poster, and we don't need it here

			err, repostedPost := blueskyapi.GetPost(*pds, *oauthToken, notification.ReasonSubject, 0, 1)
			if err != nil {
				fmt.Println("Error:", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to get post")
			}

			retweetedTweet := TranslatePostToTweet(repostedPost.Thread.Post, "", "", nil, nil, *oauthToken, *pds)
			if repostedPost.Thread.Parent != nil {
				retweetedTweet = TranslatePostToTweet(repostedPost.Thread.Post, repostedPost.Thread.Parent.Post.URI, repostedPost.Thread.Parent.Post.Author.DID, &repostedPost.Thread.Parent.Post.IndexedAt, nil, *oauthToken, *pds)
			}

			var sources []bridge.TwitterUser
			for _, user := range users {
				sources = append(sources, *user) // pain#
			}

			twitterNotifications = append(twitterNotifications, bridge.MyActivity{
				Action:        "retweet",
				CreatedAt:     bridge.TwitterTimeConverter(notification.IndexedAt),
				ID:            *bridge.BlueSkyToTwitterID(notificationId),
				Sources:       sources,
				TargetObjects: []bridge.Tweet{retweetedTweet},
			})
		default:
			//fmt.Println("Unknown notification type:", notification.Reason)
		}

		position++ // Increment position
	}

	return c.JSON(twitterNotifications)
}
