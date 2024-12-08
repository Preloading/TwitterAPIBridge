package twitterv1

import (
	"fmt"
	"math/big"
	"net/url"
	"strconv"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/Preloading/MastodonTwitterAPI/db_controller"
	"github.com/gofiber/fiber/v2"
)

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/get/statuses/home_timeline
func home_timeline(c *fiber.Ctx) error {
	// Get all of our keys, beeps, and bops
	user_did, session_uuid, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	encryptionKey, err := GetEncryptionKeyFromRequest(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Check for context
	max_id := c.Query("max_id")
	context := ""

	// Handle getting things in the past
	if max_id != "" {
		// Get the timeline context from the DB
		maxIDBigInt, ok := new(big.Int).SetString(max_id, 10)
		if !ok {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid max_id format")
		}
		fmt.Println("Max ID:", bridge.TwitterIDToBlueSky(maxIDBigInt))
		contextPtr, err := db_controller.GetTimelineContext(*user_did, *session_uuid, *maxIDBigInt, *encryptionKey)
		if err == nil {
			context = *contextPtr

			if err != nil {
				fmt.Println("Error:", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch timeline context")
			}
		}
	}

	err, res := blueskyapi.GetTimeline(*oauthToken, context)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch timeline")
	}

	// Translate the posts to tweets
	tweets := []bridge.Tweet{}

	for _, item := range res.Feed {
		tweets = append(tweets, TranslatePostToTweet(item.Post, item.Reply.Parent.URI, item.Reply.Parent.Author.DID, item.Reason))
	}

	// Store the last message id, along with our context in the DB
	err = db_controller.SetTimelineContext(*user_did, *session_uuid, tweets[len(tweets)-1].ID, res.Cursor, *encryptionKey)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to save timeline context")
	}

	return c.JSON(tweets)

}

// https://web.archive.org/web/20120708204036/https://dev.twitter.com/docs/api/1/get/statuses/show/%3Aid
func GetStatusFromId(c *fiber.Ctx) error {
	encodedId := c.Params("id")
	idBigInt, ok := new(big.Int).SetString(encodedId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	uri, _ := bridge.TwitterMsgIdToBluesky(idBigInt) // TODO: maybe look up with the retweet? idk

	_, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	err, thread := blueskyapi.GetPost(*oauthToken, uri, 0, 1)

	if err != nil {
		return err
	}

	return c.JSON(TranslatePostToTweet(thread.Thread.Post, "", "", nil))
}

func TranslatePostToTweet(tweet blueskyapi.Post, replyMsgBskyURI string, replyUserBskyId string, postReason *blueskyapi.PostReason) bridge.Tweet {
	tweetEntities := bridge.Entities{
		Hashtags:     nil,
		Urls:         nil,
		UserMentions: []bridge.UserMention{},
		Media:        []bridge.Media{},
	}

	id := 1
	for _, image := range tweet.Record.Embed.Images {
		// Process each image
		// fmt.Println("Image:", "http://10.0.0.77:3000/cdn/img/?url="+url.QueryEscape("https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:"+item.Post.Author.DID+"/"+image.Image.Ref.Link+"/@jpeg"))
		tweetEntities.Media = append(tweetEntities.Media, bridge.Media{
			ID:       *big.NewInt(int64(id)),
			IDStr:    strconv.Itoa(id),
			MediaURL: "http://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape("https://cdn.bsky.app/img/feed_thumbnail/plain/"+tweet.Author.DID+"/"+image.Image.Ref.Link+"/@jpeg"),
			// MediaURLHttps: "https://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape("https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:"+image.Image.Ref.Link+"@jpeg"),
		})
		id++
	}
	for _, faucet := range tweet.Record.Facets {
		// I haven't seen this exceed 1 element yet
		// if len(faucet.Features) > 1 {
		// fmt.Println("Faucet with more than 1 feature found!")
		// faucetJSON, err := json.Marshal(faucet)
		// if err != nil {
		// 	fmt.Println("Error encoding faucet to JSON:", err)
		// } else {
		// 	fmt.Println("Faucet JSON:", string(faucetJSON))
		// }
		// // }
		// fmt.Println(faucet.Features[0].Type)
		switch faucet.Features[0].Type {
		case "app.bsky.richtext.facet#mention":
			tweetEntities.UserMentions = append(tweetEntities.UserMentions, bridge.UserMention{
				Name:       "test",
				ScreenName: "test",
				//ScreenName: item.Post.Record.Text[faucet.Index.ByteStart+1 : faucet.Index.ByteEnd],
				ID: *bridge.BlueSkyToTwitterID(faucet.Features[0].Did),
				Indices: []int{
					faucet.Index.ByteStart,
					faucet.Index.ByteEnd,
				},
			})
		}

	}

	isRetweet := false
	// Checking if this tweet is a retweet
	if postReason != nil {
		// This might contain other things in the future, idk
		if postReason.Type == "app.bsky.feed.defs#reasonRepost" {
			// We are a retweet.
			isRetweet = true
		}
	}

	convertedTweet := bridge.Tweet{
		Coordinates: nil,
		Favourited:  tweet.Viewer.Like,
		CreatedAt: func() string {
			if isRetweet {
				return bridge.TwitterTimeConverter(postReason.CreatedAt)
			}
			return bridge.TwitterTimeConverter(tweet.Record.CreatedAt)
		}(),
		Truncated:    false,
		Text:         tweet.Record.Text,
		Entities:     tweetEntities,
		Annotations:  nil, // I am curious what annotations are
		Contributors: nil,
		ID: func() big.Int {
			// we have to use psudo ids because of https://github.com/bluesky-social/atproto/issues/1811
			if isRetweet {
				return *bridge.BlueSkyToTwitterID(fmt.Sprintf("%s:/:%s", tweet.URI, postReason.By.DID))
			}
			return *bridge.BlueSkyToTwitterID(tweet.URI)
		}(),
		IDStr: func() string {
			if isRetweet {
				return bridge.BlueSkyToTwitterID(fmt.Sprintf("%s:/:%s", tweet.URI, postReason.By.DID)).String()
			}
			return bridge.BlueSkyToTwitterID(tweet.URI).String()
		}(),
		Retweeted:         tweet.Viewer.Repost != nil,
		RetweetCount:      tweet.RepostCount,
		Geo:               nil,
		Place:             nil,
		PossiblySensitive: false,
		InReplyToUserID: func() *big.Int {
			id := bridge.BlueSkyToTwitterID(replyUserBskyId)
			if id.Cmp(big.NewInt(0)) == 0 {
				return nil
			}
			return id
		}(),
		InReplyToUserIDStr: func() *string {
			id := bridge.BlueSkyToTwitterID(replyUserBskyId)
			if id.Cmp(big.NewInt(0)) == 0 {
				return nil
			}
			idStr := id.String()
			return &idStr
		}(),
		InReplyToScreenName: &tweet.Author.DisplayName,
		User: bridge.TwitterUser{
			Name: func() string {
				if tweet.Author.DisplayName == "" {
					return tweet.Author.Handle
				}
				return tweet.Author.DisplayName
			}(),
			ProfileSidebarBorderColor: "eeeeee",
			ProfileBackgroundTile:     false,
			ProfileSidebarFillColor:   "efefef",
			CreatedAt:                 bridge.TwitterTimeConverter(tweet.Author.Associated.CreatedAt),
			ProfileImageURL:           "http://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape(tweet.Author.Avatar) + "&width=128&height=128",
			// ProfileImageURLHttps:           "https://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape(tweet.Author.Avatar) + "&width=128&height=128",
			Location:            "Twitter",
			ProfileLinkColor:    "009999",
			FollowRequestSent:   false,
			URL:                 "",
			ScreenName:          tweet.Author.Handle,
			ContributorsEnabled: false,
			UtcOffset:           nil,
			IsTranslator:        false,
			ID:                  *bridge.BlueSkyToTwitterID(tweet.URI),
			// IDStr:                          bridge.BlueSkyToTwitterID(tweet.URI).String(),
			ProfileUseBackgroundImage: false,
			ProfileTextColor:          "333333",
			Protected:                 false,
			Lang:                      "en",
			Notifications:             nil,
			TimeZone:                  nil,
			Verified:                  false,
			ProfileBackgroundColor:    "C0DEED",
			GeoEnabled:                true,
			Description:               "",
			ProfileBackgroundImageURL: "http://a0.twimg.com/images/themes/theme1/bg.png",
			// ProfileBackgroundImageURLHttps: "http://a0.twimg.com/images/themes/theme1/bg.png",
			Following: nil,

			// huh
			DefaultProfile:      false,
			DefaultProfileImage: false,
			ShowAllInlineMedia:  false,

			// User Stats
			// ListedCount:     0,
			// FavouritesCount: 0,
			// FollowersCount:  200,
			// FriendsCount:    100,
			// StatusesCount:   333,
		},
		Source: "Bluesky",
		InReplyToStatusID: func() *big.Int {
			id := bridge.BlueSkyToTwitterID(replyMsgBskyURI) // hack, later probably do this more efficently
			if id.Cmp(big.NewInt(0)) == 0 {
				return nil
			}
			return bridge.BlueSkyToTwitterID(replyMsgBskyURI)
		}(),
		InReplyToStatusIDStr: func() *string {
			id := bridge.BlueSkyToTwitterID(replyMsgBskyURI) // hack, later probably do this more efficently
			if id.Cmp(big.NewInt(0)) == 0 {
				return nil
			}
			idStr := id.String()
			return &idStr
		}(),
		RetweetedStatus: func() *bridge.Tweet {
			if isRetweet {
				retweet_bsky := tweet
				retweet_bsky.Author = postReason.By
				translatedTweet := TranslatePostToTweet(retweet_bsky, replyMsgBskyURI, replyUserBskyId, nil)
				return &translatedTweet
			}
			return nil
		}(),
	}
	return convertedTweet
}
