package twitterv1

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/gofiber/fiber/v2"
)

// Searching, oh boy.
// This function contacts an internal API, which is:
// 1. Not documented
// 2. Too common of a function to find
// 3. Has a "non internal" version that is documented, but isn't this request.

func InternalSearch(c *fiber.Ctx) error {
	// Thank you so much @Savefade for what this should repsond.
	q := c.Query("q")
	fmt.Println("Search query:", q)

	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Pagination
	max_id := c.Query("max_id")
	var until *time.Time
	if max_id != "" {
		maxIDInt, err := strconv.ParseInt(max_id, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid max_id")
		}
		_, until, _, err = bridge.TwitterMsgIdToBluesky(&maxIDInt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid max_id")
		}
	}

	var since *time.Time
	since_id := c.Query("since_id")
	if since_id != "" {
		sinceIDInt, err := strconv.ParseInt(since_id, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid since_id")
		}
		_, until, _, err = bridge.TwitterMsgIdToBluesky(&sinceIDInt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid since_id")
		}
	}

	bskySearch, err := blueskyapi.PostSearch(*pds, *oauthToken, q, since, until)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to search")
	}

	// Optimization: Get all users at once so we don't have to do it in chunks
	var dids []string
	for _, search := range bskySearch {
		dids = append(dids, search.Author.DID)
	}
	blueskyapi.GetUsersInfo(*pds, *oauthToken, dids, false)

	replyUrls := []string{}

	for _, search := range bskySearch {
		if search.Record.Reply != nil {
			replyUrls = append(replyUrls, search.Record.Reply.Parent.URI)
		}
	}

	// Get all the replies
	replyToPostData, err := blueskyapi.GetPosts(*pds, *oauthToken, replyUrls)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get reply to data")
	}

	// Create a map for quick lookup of reply dates and user IDs
	replyDateMap := make(map[string]time.Time)
	replyUserIdMap := make(map[string]string)
	replyHandleMap := make(map[string]string)
	for _, post := range replyToPostData {
		replyDateMap[post.URI] = post.IndexedAt
		replyUserIdMap[post.URI] = post.Author.DID
		replyHandleMap[post.URI] = post.Author.Handle
	}

	// Translate to twitter
	tweets := []bridge.Tweet{}
	for _, search := range bskySearch {
		var replyDate *time.Time
		var replyUserId *string
		var replyUserHandle *string
		if search.Record.Reply != nil {
			if date, exists := replyDateMap[search.Record.Reply.Parent.URI]; exists {
				replyDate = &date
			}
			if userId, exists := replyUserIdMap[search.Record.Reply.Parent.URI]; exists {
				replyUserId = &userId
			}
			if handle, exists := replyHandleMap[search.Record.Reply.Parent.URI]; exists {
				replyUserHandle = &handle
			}
		}

		if replyDate == nil {
			tweets = append(tweets, TranslatePostToTweet(search, "", "", "", nil, nil, *oauthToken, *pds))
		} else {
			tweets = append(tweets, TranslatePostToTweet(search, search.Record.Reply.Parent.URI, *replyUserId, *replyUserHandle, replyDate, nil, *oauthToken, *pds))
		}

	}

	return EncodeAndSend(c, bridge.InternalSearchResult{
		Statuses: tweets,
	})
}

// https://web.archive.org/web/20120313235613/https://dev.twitter.com/docs/api/1/get/trends/%3Awoeid
// The bluesky feature to make this possible was released 17 hours ago, and is "beta", so this is likely to break
func trends_woeid(c *fiber.Ctx) error {
	// We don't have location specific trends soooooo
	// woeid := c.Params("woeid")

	//auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	// Get trends
	bsky_trends, err := blueskyapi.GetTrends(*pds, *oauthToken)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch trends")
	}

	trends := []bridge.Trend{}

	for _, trend := range bsky_trends.Topics {
		topic_query := url.QueryEscape(trend.Topic)
		topic_query = strings.ReplaceAll(topic_query, "%20", "+")
		trends = append(trends, bridge.Trend{
			Name:        trend.Topic,
			URL:         "https://twitter.com/search?q=" + topic_query,
			Promoted:    false,
			Query:       topic_query,
			TweetVolume: 1337, // We can't get this data without search every, single, topic. So we just make it up.
		})

	}

	return EncodeAndSend(c, bridge.Trends{
		Created: time.Now(),
		Trends:  trends,
		AsOf:    time.Now(), // no clue the differ
		Locations: []bridge.TrendLocation{
			{
				Name:  "Worldwide",
				Woeid: 1, // Where on earth ID. Since bluesky trends are global, this is always 1
			},
		},
	})
}

func discovery(c *fiber.Ctx) error {

	return c.SendString(`
 

{
  "statuses": [
{
		"coordinates": null,
		"favorited": false,
		"created_at": "Tue Apr 01 05:34:17 +0000 2025",
		"truncated": false,
		"entities": {
			"media": [],
			"urls": null,
			"user_mentions": [
				{
					"name": "theonion.com",
					"id": 5479733389352290846,
					"id_str": "5479733389352290846",
					"indices": [
						59,
						72
					],
					"screen_name": "theonion.com"
				}
			],
			"hashtags": null
		},
		"text": "In a legendary move, the world's finest news organization, @theonion.com has purchased \"A Twitter Bridge\". From now on, we will:\n\n- Be including ads into our platform, hosted by the onion\n- All news stories will be from the onion\n- and more!\n\nEnjoy this enhanced version of A Twitter Bridge.",
		"annotations": null,
		"contributors": null,
		"id": 6396966313848207287,
		"id_str": "6396966313848207287",
		"geo": null,
		"place": null,
		"user": {
			"name": "Preloading",
			"profile_sidebar_border_color": "87bc44",
			"profile_background_tile": false,
			"profile_sidebar_fill_color": "e0ff92",
			"created_at": "Sat Nov 16 02:29:14 +0000 2024",
			"profile_image_url": "` + configData.CdnURL + `/cdn/img/?url=https%3A%2F%2Fcdn.bsky.app%2Fimg%2Favatar%2Fplain%2Fdid%3Aplc%3Akhcyntihpu7snjszuojjgjc4%2Fbafkreifjrbt5v4h7ufdxuwuivlkagawbwsxaqattjmjlxcfrhiymjnozvy%40jpeg:profile_bigger",
			"profile_image_url_https": "` + configData.CdnURL + `/cdn/img/?url=https%3A%2F%2Fcdn.bsky.app%2Fimg%2Favatar%2Fplain%2Fdid%3Aplc%3Akhcyntihpu7snjszuojjgjc4%2Fbafkreifjrbt5v4h7ufdxuwuivlkagawbwsxaqattjmjlxcfrhiymjnozvy%40jpeg:profile_bigger",
			"location": "",
			"profile_link_color": "0000ff",
			"follow_request_sent": false,
			"url": "",
			"favourites_count": 0,
			"contributors_enabled": false,
			"utc_offset": null,
			"id": 5123166115319017703,
			"id_str": "5123166115319017703",
			"profile_use_background_image": false,
			"profile_text_color": "000000",
			"protected": false,
			"followers_count": 92,
			"lang": "en",
			"notifications": null,
			"time_zone": null,
			"verified": false,
			"profile_background_color": "c0deed",
			"geo_enabled": false,
			"description": "I make stuff that doesn't work.\n\nhe/him\nDiscord: @Preloading\nMastodon: @preloading@mastodon.social\nWebsite: loganserver.net",
			"friends_count": 48,
			"statuses_count": 160,
			"profile_banner_url": "",
			"profile_banner_url_https": "",
			"profile_background_image_url": "",
			"following": null,
			"screen_name": "preloading.dev",
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
		"in_reply_to_screen_name": null,
		"possibly_sensitive": false,
		"retweet_count": 0,
		"retweeted": false
	}
  ],
  "stories": [
    {
      "type": "news",
      "score": 0.92,
      "data": {
        "title": "The onion has now bought A Twitter Bridge!",
        "articles": [
          {
            "title": "The onion has now bought A Twitter Bridge!",
            "url": {
              "display_url": "twitterbridge.loganserver.net",
              "expanded_url": "https://twitterbridge.loganserver.net"
            },
            "tweet_count": 9999999999,
            "media": []
          }
        ]
      },
      "social_proof": {
        "social_proof_type": "social",
        "referenced_by": {
          "global_count": 99999999999,
          "statuses": [
            {
		"coordinates": null,
		"favorited": false,
		"created_at": "Tue Apr 01 05:34:17 +0000 2025",
		"truncated": false,
		"entities": {
			"media": [],
			"urls": null,
			"user_mentions": [
				{
					"name": "theonion.com",
					"id": 5479733389352290846,
					"id_str": "5479733389352290846",
					"indices": [
						59,
						72
					],
					"screen_name": "theonion.com"
				}
			],
			"hashtags": null
		},
		"text": "In a legendary move, the world's finest news organization, @theonion.com has purchased \"A Twitter Bridge\". From now on, we will:\n\n- Be including ads into our platform, hosted by the onion\n- All news stories will be from the onion\n- and more!\n\nEnjoy this enhanced version of A Twitter Bridge.",
		"annotations": null,
		"contributors": null,
		"id": 6396966313848207287,
		"id_str": "6396966313848207287",
		"geo": null,
		"place": null,
		"user": {
			"name": "Preloading",
			"profile_sidebar_border_color": "87bc44",
			"profile_background_tile": false,
			"profile_sidebar_fill_color": "e0ff92",
			"created_at": "Sat Nov 16 02:29:14 +0000 2024",
			"profile_image_url": "` + configData.CdnURL + `/cdn/img/?url=https%3A%2F%2Fcdn.bsky.app%2Fimg%2Favatar%2Fplain%2Fdid%3Aplc%3Akhcyntihpu7snjszuojjgjc4%2Fbafkreifjrbt5v4h7ufdxuwuivlkagawbwsxaqattjmjlxcfrhiymjnozvy%40jpeg:profile_bigger",
			"profile_image_url_https": "` + configData.CdnURL + `/cdn/img/?url=https%3A%2F%2Fcdn.bsky.app%2Fimg%2Favatar%2Fplain%2Fdid%3Aplc%3Akhcyntihpu7snjszuojjgjc4%2Fbafkreifjrbt5v4h7ufdxuwuivlkagawbwsxaqattjmjlxcfrhiymjnozvy%40jpeg:profile_bigger",
			"location": "",
			"profile_link_color": "0000ff",
			"follow_request_sent": false,
			"url": "",
			"favourites_count": 0,
			"contributors_enabled": false,
			"utc_offset": null,
			"id": 5123166115319017703,
			"id_str": "5123166115319017703",
			"profile_use_background_image": false,
			"profile_text_color": "000000",
			"protected": false,
			"followers_count": 92,
			"lang": "en",
			"notifications": null,
			"time_zone": null,
			"verified": false,
			"profile_background_color": "c0deed",
			"geo_enabled": false,
			"description": "I make stuff that doesn't work.\n\nhe/him\nDiscord: @Preloading\nMastodon: @preloading@mastodon.social\nWebsite: loganserver.net",
			"friends_count": 48,
			"statuses_count": 160,
			"profile_banner_url": "",
			"profile_banner_url_https": "",
			"profile_background_image_url": "",
			"following": null,
			"screen_name": "preloading.dev",
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
		"in_reply_to_screen_name": null,
		"possibly_sensitive": false,
		"retweet_count": 0,
		"retweeted": false
	}
          ]
        }
      }
    }
  ],
  "related_queries": [
    {
      "query": "The Onion"
    },
    {
      "query": "A Twitter Bridge"
    }
  ],
  "spelling_corrections": []
}


	`)

}
