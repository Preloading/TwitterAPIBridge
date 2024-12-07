package twitterv1

import (
	"fmt"
	"math/big"
	"net/url"
	"strconv"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
)

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/get/statuses/home_timeline
func home_timeline(c *fiber.Ctx) error {
	_, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	err, res := blueskyapi.GetTimeline(*oauthToken)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch timeline")
	}

	tweets := []bridge.Tweet{}

	for _, item := range res.Feed {
		tweets = append(tweets, TranslatePostToTweet(item.Post, item.Reply.Parent.URI, item.Reply.Parent.Author.DID))
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
	uri := bridge.TwitterIDToBlueSky(idBigInt)

	_, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	err, thread := blueskyapi.GetPost(*oauthToken, uri, 0, 1)

	if err != nil {
		return err
	}

	return c.JSON(TranslatePostToTweet(thread.Thread.Post, "", ""))
}

func TranslatePostToTweet(tweet blueskyapi.Post, replyMsgBskyURI string, replyUserBskyId string) bridge.Tweet {
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
			fmt.Println("we found a mention")
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
	convertedTweet := bridge.Tweet{
		Coordinates:       nil,
		Favourited:        tweet.Viewer.Like,
		CreatedAt:         bridge.TwitterTimeConverter(tweet.Record.CreatedAt),
		Truncated:         false,
		Text:              tweet.Record.Text,
		Entities:          tweetEntities,
		Annotations:       nil, // I am curious what annotations are
		Contributors:      nil,
		ID:                *bridge.BlueSkyToTwitterID(tweet.URI),
		IDStr:             bridge.BlueSkyToTwitterID(tweet.URI).String(),
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
			DefaultProfileImage: true,
			ShowAllInlineMedia:  false,

			// User Stats
			ListedCount:     0,
			FavouritesCount: 0,
			FollowersCount:  200,
			FriendsCount:    100,
			StatusesCount:   333,
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
	}
	return convertedTweet
}
