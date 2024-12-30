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
	// Thank you so much @Safefade for what this returns for follows.
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
				ID:        bridge.BlueSkyToTwitterID(notification.URI),
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
				ID:        bridge.BlueSkyToTwitterID(notificationId),
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
				ID:            bridge.BlueSkyToTwitterID(notificationId),
				Sources:       sources,
				TargetObjects: []bridge.Tweet{retweetedTweet},
			})
		default:
			//fmt.Println("Unknown notification type:", notification.Reason)
		}

		position++ // Increment position
	}
	return c.JSON(twitterNotifications)

	return c.SendString(`
	[
  {
    "action": "retweet",
    "created_at": "2024-12-29T12:34:56+00:00",
    "id": "1234567890abcdef",
    "sources": [
      {
        "name": "JohnDoe",
        "profile_sidebar_border_color": "C0DEED",
        "profile_background_tile": false,
        "profile_sidebar_fill_color": "DDEEF6",
        "location": "New York, USA",
        "profile_image_url": "https://example.com/profiles/johndoe_avatar.png",
        "created_at": "2020-01-22T16:27:50+00:00",
        "profile_link_color": "0084B4",
        "favourites_count": 15,
        "url": null,
        "contributors_enabled": false,
        "utc_offset": -18000,
        "id": "did:plc:abcdef1234567890",
        "profile_use_background_image": false,
        "profile_text_color": "333333",
        "protected": false,
        "followers_count": 250,
        "lang": "en",
        "verified": true,
        "profile_background_color": "C0DEED",
        "geo_enabled": true,
        "notifications": false,
        "description": "Tech enthusiast and blogger",
        "time_zone": "EST",
        "friends_count": 180,
        "statuses_count": 500,
        "profile_background_image_url": "https://example.com/backgrounds/default.png",
        "screen_name": "JohnDoe123",
        "following": true
      }
    ],
	"targets": [
	{
	"coordinates": null,
	"favorited": false,
	"created_at": "Sun Dec 29 22:43:17 +0000 2024",
	"truncated": false,
	"entities": {
		"media": [
			{
				"id": 1,
				"id_str": "1",
				"media_url": "http://10.0.0.77:3000/cdn/img/?url=https%3A%2F%2Fcdn.bsky.app%2Fimg%2Ffeed_thumbnail%2Fplain%2Fdid%3Aplc%3Ayypmewyevkpcc2gqtb6mubb2%2Fbafkreigcglcl4szzqafiq6ay66q4s4kg3rkqmbhokku4ebiigo4wsogfyu%2F%40jpeg",
				"media_url_https": "",
				"url": "",
				"display_url": "",
				"expanded_url": "",
				"sizes": null,
				"type": "",
				"indices": null
			}
		],
		"urls": null,
		"user_mentions": [],
		"hashtags": null
	},
	"text": "got the best screenshot in #tf2 ever",
	"annotations": null,
	"contributors": null,
	"id": 1690229107132447236958063898331357098655707862726830507095380669682071829383676425265684087466036404071897638534331939139458028384499286701642897,
	"id_str": "1690229107132447236958063898331357098655707862726830507095380669682071829383676425265684087466036404071897638534331939139458028384499286701642897",
	"geo": null,
	"place": null,
	"user": {
		"name": "rodeo",
		"profile_sidebar_border_color": "87bc44",
		"profile_background_tile": false,
		"profile_sidebar_fill_color": "e0ff92",
		"created_at": "Tue Oct 29 16:43:33 +0000 2024",
		"profile_image_url": "http://10.0.0.77:3000/cdn/img/?url=https%3A%2F%2Fcdn.bsky.app%2Fimg%2Favatar%2Fplain%2Fdid%3Aplc%3Ayypmewyevkpcc2gqtb6mubb2%2Fbafkreicsv4dxffqewgd3yggwsf2brlapm4ftro567lcf3zgnhui4fr57tm%40jpeg:profile_bigger",
		"location": "",
		"profile_link_color": "0000ff",
		"follow_request_sent": false,
		"url": "",
		"favourites_count": 0,
		"contributors_enabled": false,
		"utc_offset": null,
		"id": 283395592579705328644393982843492908104031790027669,
		"profile_use_background_image": false,
		"profile_text_color": "000000",
		"protected": false,
		"followers_count": 7,
		"lang": "en",
		"notifications": null,
		"time_zone": null,
		"verified": false,
		"profile_background_color": "c0deed",
		"geo_enabled": false,
		"description": "Brody | 26 | he/him | Digital Artist | nsfw sometimes so beware",
		"friends_count": 11,
		"statuses_count": 5,
		"profile_background_image_url": "",
		"following": null,
		"screen_name": "nitroladybug.bsky.social",
		"show_all_inline_media": false,
		"is_translator": false,
		"listed_count": 0,
		"default_profile": false,
		"default_profile_image": false
	},
	"source": "Bluesky",
	"in_reply_to_user_id": null,
	"in_reply_to_user_id_str": null,
	"in_reply_to_status_id": null,
	"in_reply_to_status_id_str": null,
	"in_reply_to_screen_name": "rodeo",
	"possibly_sensitive": false,
	"retweet_count": 11,
	"retweeted": false
},
{
	"coordinates": null,
	"favorited": false,
	"created_at": "Sun Dec 29 22:43:17 +0000 2024",
	"truncated": false,
	"entities": {
		"media": [
			{
				"id": 1,
				"id_str": "1",
				"media_url": "http://10.0.0.77:3000/cdn/img/?url=https%3A%2F%2Fcdn.bsky.app%2Fimg%2Ffeed_thumbnail%2Fplain%2Fdid%3Aplc%3Ayypmewyevkpcc2gqtb6mubb2%2Fbafkreigcglcl4szzqafiq6ay66q4s4kg3rkqmbhokku4ebiigo4wsogfyu%2F%40jpeg",
				"media_url_https": "",
				"url": "",
				"display_url": "",
				"expanded_url": "",
				"sizes": null,
				"type": "",
				"indices": null
			}
		],
		"urls": null,
		"user_mentions": [],
		"hashtags": null
	},
	"text": "got the best screenshot in #tf2 ever",
	"annotations": null,
	"contributors": null,
	"id": 1690229107132447236958063898331357098655707862726830507095380669682071829383676425265684087466036404071897638534331939139458028384499286701642897,
	"id_str": "1690229107132447236958063898331357098655707862726830507095380669682071829383676425265684087466036404071897638534331939139458028384499286701642897",
	"geo": null,
	"place": null,
	"user": {
		"name": "rodeo",
		"profile_sidebar_border_color": "87bc44",
		"profile_background_tile": false,
		"profile_sidebar_fill_color": "e0ff92",
		"created_at": "Tue Oct 29 16:43:33 +0000 2024",
		"profile_image_url": "http://10.0.0.77:3000/cdn/img/?url=https%3A%2F%2Fcdn.bsky.app%2Fimg%2Favatar%2Fplain%2Fdid%3Aplc%3Ayypmewyevkpcc2gqtb6mubb2%2Fbafkreicsv4dxffqewgd3yggwsf2brlapm4ftro567lcf3zgnhui4fr57tm%40jpeg:profile_bigger",
		"location": "",
		"profile_link_color": "0000ff",
		"follow_request_sent": false,
		"url": "",
		"favourites_count": 0,
		"contributors_enabled": false,
		"utc_offset": null,
		"id": 283395592579705328644393982843492908104031790027669,
		"profile_use_background_image": false,
		"profile_text_color": "000000",
		"protected": false,
		"followers_count": 7,
		"lang": "en",
		"notifications": null,
		"time_zone": null,
		"verified": false,
		"profile_background_color": "c0deed",
		"geo_enabled": false,
		"description": "Brody | 26 | he/him | Digital Artist | nsfw sometimes so beware",
		"friends_count": 11,
		"statuses_count": 5,
		"profile_background_image_url": "",
		"following": null,
		"screen_name": "nitroladybug.bsky.social",
		"show_all_inline_media": false,
		"is_translator": false,
		"listed_count": 0,
		"default_profile": false,
		"default_profile_image": false
	},
	"source": "Bluesky",
	"in_reply_to_user_id": null,
	"in_reply_to_user_id_str": null,
	"in_reply_to_status_id": null,
	"in_reply_to_status_id_str": null,
	"in_reply_to_screen_name": "rodeo",
	"possibly_sensitive": false,
	"retweet_count": 11,
	"retweeted": false
}]
	
    
  }
	`)
}
