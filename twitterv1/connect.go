package twitterv1

import (
	"fmt"
	"strconv"
	"time"

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
		return EncodeAndSend(c, []bridge.TwitterUser{})
	}
	users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, dids, false)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
	}

	return EncodeAndSend(c, users)
}

// /i/activity/about_me.json?contributor_details=1&include_entities=true&include_my_retweet=true&send_error_codes=true
func GetMyActivity(c *fiber.Ctx) error {
	// Thank you so much @Savefade for what this returns for follows.
	// This function could probably optimized to use less GetUsers calls, but whatever.
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Context is key, or so i've heard.
	context := ""
	maxID := c.Query("max_id")
	if maxID != "" {
		maxIDInt, err := strconv.ParseInt(maxID, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid max_id")
		}
		// I've had some problems with it giving the same result twice, so I'm going to subtract 2ms to the max_id
		maxIDInt -= 2
		max_time := time.UnixMilli(maxIDInt)
		context = max_time.Format(time.RFC3339)
	}

	// count
	countStr := c.Query("count")
	count := 50
	if countStr != "" {
		countInt, err := strconv.Atoi(countStr)
		if err == nil {
			count = countInt
		}
	}

	bskyNotifcations, err := blueskyapi.GetNotifications(*pds, *oauthToken, count, context)

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
			if position+1 < len(bskyNotifcations.Notifications) {
				position--
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
				Action:      "follow",
				CreatedAt:   bridge.TwitterTimeConverter(notification.IndexedAt),
				MinPosition: notification.IndexedAt.UnixMilli(),
				MaxPosition: bskyNotifcations.Notifications[position].IndexedAt.UnixMilli(), // I don't believe that these IDs are used for anything besides pagination & positioning
				Sources:     sources,
			})
			position++
		case "like":
			usersInBlock := []string{notification.Author.DID}
			for position+1 < len(bskyNotifcations.Notifications) {
				position++
				if bskyNotifcations.Notifications[position].Reason == "like" && bskyNotifcations.Notifications[position].ReasonSubject == notification.ReasonSubject {
					usersInBlock = append(usersInBlock, bskyNotifcations.Notifications[position].Author.DID)
				} else {
					break
				}
			}
			if position+1 < len(bskyNotifcations.Notifications) {
				position--
			}

			// slight optimization in network traffic, saves us from having to call seperately for the poster
			_, poster_did, _ := blueskyapi.GetURIComponents(notification.ReasonSubject)
			usersInBlock = append(usersInBlock, poster_did)
			users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, usersInBlock, false)
			if err != nil {
				fmt.Println("Error:", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
			}

			// Remove the current user from the list if present (only once)
			for i, user := range users {
				if user.ID == *bridge.BlueSkyToTwitterID(*my_did) {
					users = append(users[:i], users[i+1:]...)
					break
				}
			}

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
				Action:      "favorite",
				CreatedAt:   bridge.TwitterTimeConverter(notification.IndexedAt),
				MinPosition: notification.IndexedAt.UnixMilli(),
				MaxPosition: bskyNotifcations.Notifications[position].IndexedAt.UnixMilli(), // I don't believe that these IDs are used for anything besides pagination & positioning
				Sources:     sources,
				Targets:     []bridge.Tweet{likedTweet},
			})
			position++
		// case "repost":
		// 	fmt.Println("Repost")
		// 	usersInBlock := []string{notification.Author.DID}
		// 	for position+1 < len(bskyNotifcations.Notifications)+1 {
		// 		position++
		// 		if bskyNotifcations.Notifications[position].Reason == "repost" && bskyNotifcations.Notifications[position].ReasonSubject == notification.ReasonSubject {
		// 			usersInBlock = append(usersInBlock, bskyNotifcations.Notifications[position].Author.DID)
		// 		} else {
		// 			break
		// 		}
		// 	}
		// 	if position+1 < len(bskyNotifcations.Notifications) {
		// 		position--
		// 	}

		// 	// slight optimization in network traffic, saves us from having to call seperately for the poster
		// 	_, poster_did, _ := blueskyapi.GetURIComponents(notification.ReasonSubject)
		// 	usersInBlock = append(usersInBlock, poster_did)
		// 	users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, usersInBlock, false)
		// 	if err != nil {
		// 		fmt.Println("Error:", err)
		// 		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
		// 	}

		// 	// Remove the current user from the list if present (only once)
		// 	for i, user := range users {
		// 		if user.ID == *bridge.BlueSkyToTwitterID(*my_did) {
		// 			users = append(users[:i], users[i+1:]...)
		// 			break
		// 		}
		// 	}

		// 	err, repostedPost := blueskyapi.GetPost(*pds, *oauthToken, notification.ReasonSubject, 0, 1)
		// 	if err != nil {
		// 		fmt.Println("Error:", err)
		// 		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get post")
		// 	}

		// 	retweetedTweet := TranslatePostToTweet(repostedPost.Thread.Post, "", "", nil, nil, *oauthToken, *pds)
		// 	if repostedPost.Thread.Parent != nil {
		// 		retweetedTweet = TranslatePostToTweet(repostedPost.Thread.Post, repostedPost.Thread.Parent.Post.URI, repostedPost.Thread.Parent.Post.Author.DID, &repostedPost.Thread.Parent.Post.IndexedAt, nil, *oauthToken, *pds)
		// 	}

		// 	var sources []bridge.TwitterUser
		// 	for _, user := range users {
		// 		sources = append(sources, *user) // pain#
		// 	}

		// 	twitterNotifications = append(twitterNotifications, bridge.MyActivity{
		// 		Action:        "retweet",
		// 		CreatedAt:     bridge.TwitterTimeConverter(notification.IndexedAt),
		// 		MinPosition:   notification.IndexedAt.UnixMilli(),
		// 		MaxPosition:   bskyNotifcations.Notifications[position].IndexedAt.UnixMilli(), // I don't believe that these IDs are used for anything besides pagination & positioning
		// 		Sources:       sources,
		// 		TargetObjects: []bridge.Tweet{retweetedTweet},
		// 	})
		// 	position++
		default:
			fmt.Println("Unknown notification type:", notification.Reason)
		}

		position++ // Increment position
	}

	return EncodeAndSend(c, twitterNotifications)
}
