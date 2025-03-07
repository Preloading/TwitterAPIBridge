package twitterv1

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/Preloading/TwitterAPIBridge/db_controller"
	"github.com/gofiber/fiber/v2"
)

type TweetsRoot struct {
	XMLName  xml.Name `xml:"statuses"`
	Statuses []bridge.Tweet
}

func home_timeline(c *fiber.Ctx) error {
	return convert_timeline(c, "", blueskyapi.GetTimeline)
}

func user_timeline(c *fiber.Ctx) error {
	actor := c.Query("screen_name")
	if actor == "" {
		actor = c.Query("user_id")
		if actor == "" {
			return c.Status(fiber.StatusBadRequest).SendString("No user provided")
		}
		actorInt, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&actorInt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actor = *actorPtr
	}
	return convert_timeline(c, actor, blueskyapi.GetUserTimeline)
}

func media_timeline(c *fiber.Ctx) error {
	actor := c.Query("screen_name")
	if actor == "" {
		actor = c.Query("user_id")
		if actor == "" {
			return c.Status(fiber.StatusBadRequest).SendString("No user provided")
		}
		actorInt, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&actorInt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actor = *actorPtr
	}
	return convert_timeline(c, actor, blueskyapi.GetMediaTimeline)
}

func likes_timeline(c *fiber.Ctx) error {
	// We shall pretend that the only thing it can be is a user id. TODO: maybe rectify this later
	actor := c.Params("id")
	actorInt, err := strconv.ParseInt(actor, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
	}
	actorPtr, err := bridge.TwitterIDToBlueSky(&actorInt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
	}
	actor = *actorPtr

	return convert_timeline(c, actor, blueskyapi.GetActorLikes)
}

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/get/statuses/home_timeline
func convert_timeline(c *fiber.Ctx, param string, fetcher func(string, string, string, string, int) (error, *blueskyapi.Timeline)) error {
	// Get all of our keys, beeps, and bops
	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Limits
	limitStr := c.Query("count")
	if limitStr == "" {
		limitStr = "20"
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid count format")
	}
	if limit > 100 {
		limit = 100
	}

	// Check for context
	max_id := c.Query("max_id")
	context := ""

	// Handle getting things in the past
	if max_id != "" {
		// Get the timeline context from the DB
		maxIDInt, err := strconv.ParseInt(max_id, 10, 64)
		fmt.Println("Max ID:", maxIDInt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid max_id format")
		}
		uri, date, _, err := bridge.TwitterMsgIdToBluesky(&maxIDInt)
		fmt.Println("Max ID:", uri)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid max_id format")
		}
		context = date.Format(time.RFC3339)
	}

	fmt.Println("Context:", context)
	err, res := fetcher(*pds, *oauthToken, context, param, limit)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch timeline")
	}

	// Caching the user DIDs efficiently
	userDIDs := []string{}

	for _, item := range res.Feed {
		if !slices.Contains(userDIDs, item.Post.Author.DID) {
			userDIDs = append(userDIDs, item.Post.Author.DID)
		}
	}

	blueskyapi.GetUsersInfo(*pds, *oauthToken, userDIDs, false)

	// Translate the posts to tweets
	tweets := []bridge.Tweet{}

	for _, item := range res.Feed {
		tweets = append(tweets, TranslatePostToTweet(item.Post, item.Reply.Parent.URI, item.Reply.Parent.Author.DID, &item.Reply.Parent.Record.CreatedAt, item.Reason, *oauthToken, *pds))
	}

	if c.Params("filetype") == "xml" { // i wonder why twitter ditched xml
		tweetsRoot := TweetsRoot{
			Statuses: tweets,
		}
		return EncodeAndSend(c, tweetsRoot)
	}

	return EncodeAndSend(c, tweets)

}

// Replies
// This is going to be painful to implement with lack of any docs
func RelatedResults(c *fiber.Ctx) error {
	encodedId := c.Params("id")
	idInt, err := strconv.ParseInt(encodedId, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	// Fetch ID
	uriPtr, _, _, err := bridge.TwitterMsgIdToBluesky(&idInt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	uri := *uriPtr

	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	err, thread := blueskyapi.GetPost(*pds, *oauthToken, uri, 1, 0)

	if err != nil {
		return err
	}

	if thread.Thread.Replies == nil {
		return EncodeAndSend(c, []bridge.RelatedResultsQuery{})
	}

	// Caching the user DIDs efficiently
	userDIDs := []string{}

	for _, item := range *thread.Thread.Replies {
		if !slices.Contains(userDIDs, item.Post.Author.DID) {
			userDIDs = append(userDIDs, item.Post.Author.DID)
		}
	}

	blueskyapi.GetUsersInfo(*pds, *oauthToken, userDIDs, false)

	postAuthor := bridge.BlueSkyToTwitterID(thread.Thread.Post.Author.DID)

	twitterReplies := bridge.RelatedResultsQuery{
		Annotations: []bridge.Annotations{},
		ResultType:  "Tweet",
		Score:       1.0,
		GroupName:   "TweetsWithConversation",
		Results:     []bridge.Results{},
	}
	for _, reply := range *thread.Thread.Replies {
		reply.Post.Record.CreatedAt = reply.Post.IndexedAt
		twitterReplies.Results = append(twitterReplies.Results, bridge.Results{
			Kind:  "Tweet",
			Score: 1.0,
			Value: TranslatePostToTweet(reply.Post, uri, strconv.FormatInt(*postAuthor, 10), &thread.Thread.Post.Record.CreatedAt, nil, *oauthToken, *pds),
			Annotations: []bridge.Annotations{
				{
					ConversationRole: "Descendant",
				},
			},
		})
	}

	return EncodeAndSend(c, []bridge.RelatedResultsQuery{twitterReplies})
}

// https://web.archive.org/web/20120708204036/https://dev.twitter.com/docs/api/1/get/statuses/show/%3Aid
func GetStatusFromId(c *fiber.Ctx) error {
	encodedId := c.Params("id")
	idInt, err := strconv.ParseInt(encodedId, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	// Fetch ID
	uriPtr, _, _, err := bridge.TwitterMsgIdToBluesky(&idInt)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	uri := *uriPtr

	fmt.Println("URI:", uri)
	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		emptyString := ""
		oauthToken = &emptyString
	}

	err, thread := blueskyapi.GetPost(*pds, *oauthToken, uri, 0, 1)

	if err != nil {
		return err
	}

	// TODO: Some things may be needed for reposts to show up correctly. thats a later problem :)
	if thread.Thread.Parent == nil {
		return EncodeAndSend(c, TranslatePostToTweet(thread.Thread.Post, "", "", nil, nil, *oauthToken, *pds))
	} else {
		return EncodeAndSend(c, TranslatePostToTweet(thread.Thread.Post, thread.Thread.Parent.Post.URI, thread.Thread.Parent.Post.Author.DID, &thread.Thread.Parent.Post.Record.CreatedAt, nil, *oauthToken, *pds))
	}
}

// This gigantic function is used to convert the bluesky post format, into a format that is compatible with the twitter API.
// https://web.archive.org/web/20120506182126/https://dev.twitter.com/docs/platform-objects/tweets
func TranslatePostToTweet(tweet blueskyapi.Post, replyMsgBskyURI string, replyUserBskyId string, replyTimeStamp *time.Time, postReason *blueskyapi.PostReason, token string, pds string) bridge.Tweet {
	var err error
	textOffset := 0

	isRetweet := false
	bsky_retweet_og_author := tweet.Author

	// Checking if this tweet is a retweet
	if postReason != nil {
		// This might contain other things in the future, idk
		if postReason.Type == "app.bsky.feed.defs#reasonRepost" {
			// We are a retweet.
			isRetweet = true

		}
	}

	if len(tweet.Record.Langs) > 0 {
		db_controller.StoreAnalyticData(db_controller.AnalyticData{
			DataType:  "postviewing",
			Timestamp: time.Now(),
			Language:  tweet.Record.Langs[0],
		})
	} else {
		db_controller.StoreAnalyticData(db_controller.AnalyticData{
			DataType:  "postviewing",
			Timestamp: time.Now(),
		})
	}

	processedText := func() string {
		// This fucks up all the entities :crying:

		if isRetweet {
			retweetedText := "RT @" + bsky_retweet_og_author.Handle + ": "
			textOffset += utf8.RuneCountInString(retweetedText)
			return retweetedText + tweet.Record.Text
		}
		return tweet.Record.Text
	}()

	if isRetweet {
		tweet.Author = postReason.By
	}

	tweetEntities := bridge.Entities{
		Hashtags:     nil,
		Urls:         nil,
		UserMentions: []bridge.UserMention{},
		Media:        []bridge.Media{},
	}

	id := 1
	for _, image := range tweet.Record.Embed.Images {
		// Add the image "url" to the text
		startLen, endLen := 0, 0
		formattedImageURL := configData.ImgURLText
		displayURL := configData.ImgDisplayText
		shortCode := ""
		if displayURL != "" {
			displayURL = strings.ReplaceAll(displayURL, "{shortblob}", image.Image.Ref.Link[len(image.Image.Ref.Link)-6:])
			displayURL = strings.ReplaceAll(displayURL, "{fullblob}", image.Image.Ref.Link)
			displayURL = strings.ReplaceAll(displayURL, "{user_did}", tweet.Author.DID)
			if strings.Contains(displayURL, "{shortcode}") {
				shortCode, err = CreateShortLink("/cdn/img/bsky/" + tweet.Author.DID + "/" + image.Image.Ref.Link + ".jpg")
				if err != nil {
					fmt.Println("Error creating short link:", err)
					displayURL = strings.ReplaceAll(displayURL, "{shortcode}", "")
				} else {
					displayURL = strings.ReplaceAll(displayURL, "{shortcode}", shortCode)
				}
			}

			if len(processedText) == 0 {
				endLen = utf8.RuneCountInString(displayURL)

				processedText = displayURL
			} else {
				startLen = utf8.RuneCountInString(processedText) + 1
				endLen = (utf8.RuneCountInString(processedText) + 1) + utf8.RuneCountInString(displayURL)

				processedText = processedText + "\n" + displayURL
			}
		}
		if formattedImageURL != "" {
			formattedImageURL = strings.ReplaceAll(formattedImageURL, "{shortblob}", image.Image.Ref.Link[len(image.Image.Ref.Link)-6:])
			formattedImageURL = strings.ReplaceAll(formattedImageURL, "{fullblob}", image.Image.Ref.Link)
			formattedImageURL = strings.ReplaceAll(formattedImageURL, "{user_did}", tweet.Author.DID)
			if strings.Contains(formattedImageURL, "{shortcode}") {
				if shortCode == "" {
					shortCode, err = CreateShortLink("/cdn/img/bsky/" + tweet.Author.DID + "/" + image.Image.Ref.Link + ".jpg")
					if err != nil {
						fmt.Println("Error creating short link:", err)
						formattedImageURL = strings.ReplaceAll(formattedImageURL, "{shortcode}", "")
					} else {
						formattedImageURL = strings.ReplaceAll(formattedImageURL, "{shortcode}", shortCode)
					}
				} else {
					formattedImageURL = strings.ReplaceAll(formattedImageURL, "{shortcode}", shortCode)
				}

			}
		}

		mediaWebURL := configData.CdnURL + "/cdn/img/bsky/" + tweet.Author.DID + "/" + image.Image.Ref.Link + ".jpg"

		// Process each image
		tweetEntities.Media = append(tweetEntities.Media, bridge.Media{
			Type:          "photo",
			ID:            int64(id),
			IDStr:         strconv.Itoa(id),
			MediaURL:      mediaWebURL,
			MediaURLHttps: mediaWebURL,

			DisplayURL:  displayURL,
			ExpandedURL: mediaWebURL,
			URL:         formattedImageURL,

			Sizes: bridge.MediaSize{
				Thumb: func() bridge.Size {
					w, h := image.AspectRatio.Width, image.AspectRatio.Height
					if w > h {
						return bridge.Size{
							W:      150,
							H:      int(150 * float64(h) / float64(w)),
							Resize: "crop",
						}
					}
					return bridge.Size{
						W:      int(150 * float64(w) / float64(h)),
						H:      150,
						Resize: "crop",
					}
				}(),
				Small: func() bridge.Size {
					w, h := image.AspectRatio.Width, image.AspectRatio.Height
					if w > h {
						return bridge.Size{
							W:      340,
							H:      int(340 * float64(h) / float64(w)),
							Resize: "fit",
						}
					}
					return bridge.Size{
						W:      int(340 * float64(w) / float64(h)),
						H:      340,
						Resize: "fit",
					}
				}(),
				Medium: func() bridge.Size {
					w, h := image.AspectRatio.Width, image.AspectRatio.Height
					if w > h {
						return bridge.Size{
							W:      600,
							H:      int(600 * float64(h) / float64(w)),
							Resize: "fit",
						}
					}
					return bridge.Size{
						W:      int(600 * float64(w) / float64(h)),
						H:      600,
						Resize: "fit",
					}
				}(),
				Large: bridge.Size{
					W:      image.AspectRatio.Width,
					H:      image.AspectRatio.Height,
					Resize: "fit",
				},
			},

			Indices: []int{
				startLen,
				endLen,
			},
			Start:     startLen,
			End:       endLen,
			StartAttr: startLen,
			EndAttr:   endLen,
		})
		id++
	}

	// Faucets, essentially links, mentions, and hashtags
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
			if faucet.Index.ByteEnd > len(tweet.Record.Text) { // yup! this is in fact necessary.
				break
			}
			if faucet.Index.ByteStart < 0 {
				break
			}
			startIndex := utf8.RuneCountInString(tweet.Record.Text[:faucet.Index.ByteStart]) + textOffset
			endIndex := utf8.RuneCountInString(tweet.Record.Text[:faucet.Index.ByteEnd]) + textOffset
			tweetEntities.UserMentions = append(tweetEntities.UserMentions, bridge.UserMention{
				Name:       tweet.Record.Text[faucet.Index.ByteStart+1 : faucet.Index.ByteEnd],
				ScreenName: tweet.Record.Text[faucet.Index.ByteStart+1 : faucet.Index.ByteEnd],
				ID:         bridge.BlueSkyToTwitterID(faucet.Features[0].Did),
				IDStr:      strconv.FormatInt(*bridge.BlueSkyToTwitterID(faucet.Features[0].Did), 10),
				Indices: []int{
					startIndex,
					endIndex,
				},
				Start: startIndex,
				End:   endIndex,
			})
		case "app.bsky.richtext.facet#link":
			if faucet.Index.ByteEnd > len(tweet.Record.Text) { // yup! this is in fact necessary.
				break
			}
			if faucet.Index.ByteStart < 0 {
				break
			}
			startIndex := utf8.RuneCountInString(tweet.Record.Text[:faucet.Index.ByteStart]) + textOffset
			endIndex := utf8.RuneCountInString(tweet.Record.Text[:faucet.Index.ByteEnd]) + textOffset
			tweetEntities.Urls = append(tweetEntities.Urls, bridge.URL{
				ExpandedURL: faucet.Features[0].Uri,
				URL:         faucet.Features[0].Uri,
				DisplayURL:  tweet.Record.Text[faucet.Index.ByteStart:faucet.Index.ByteEnd],
				Start:       startIndex,
				End:         endIndex,
				Indices: []int{
					startIndex,
					endIndex,
				},
				XMLName: xml.Name{Local: "url"},
				XMLFormat: bridge.URLXMLFormat{
					Start:       startIndex,
					End:         endIndex,
					DisplayURL:  tweet.Record.Text[faucet.Index.ByteStart:faucet.Index.ByteEnd],
					URL:         faucet.Features[0].Uri,
					ExpandedURL: faucet.Features[0].Uri,
				},
			})
		case "app.bsky.richtext.facet#tag":
			if faucet.Index.ByteEnd > len(tweet.Record.Text) { // yup! this is in fact necessary.
				break
			}
			if faucet.Index.ByteStart < 0 {
				break
			}
			startIndex := utf8.RuneCountInString(tweet.Record.Text[:faucet.Index.ByteStart]) + textOffset
			endIndex := utf8.RuneCountInString(tweet.Record.Text[:faucet.Index.ByteEnd]) + textOffset
			tweetEntities.Hashtags = append(tweetEntities.Hashtags, bridge.Hashtag{
				Text: faucet.Features[0].Tag, // Shortcut url
				Indices: []int{
					startIndex,
					endIndex,
				},
				Start: startIndex,
				End:   endIndex,
			})
		}
	}

	// Videos.
	// I am 99% sure twitter API 1.0 did not have proper video uploads, so we embed it as a link.

	if tweet.Record.Embed.Video.Video != nil {
		video := tweet.Record.Embed.Video // i don't want to refrence it forever

		// Adding the URL into the text of the tweet
		startLen, endLen := 0, 0
		formattedVideoURL := configData.VidURLText
		displayURL := configData.VidDisplayText
		shortCode := ""
		if displayURL != "" {
			displayURL = strings.ReplaceAll(displayURL, "{shortblob}", video.Video.Ref.Link[len(video.Video.Ref.Link)-6:])
			displayURL = strings.ReplaceAll(displayURL, "{fullblob}", video.Video.Ref.Link)
			displayURL = strings.ReplaceAll(displayURL, "{user_did}", tweet.Author.DID)
			if strings.Contains(displayURL, "{shortcode}") {
				shortCode, err = CreateShortLink("/cdn/vid/bsky/" + tweet.Author.DID + "/" + video.Video.Ref.Link + "/")
				if err != nil {
					fmt.Println("Error creating short link:", err)
					displayURL = strings.ReplaceAll(displayURL, "{shortcode}", "")
				} else {
					displayURL = strings.ReplaceAll(displayURL, "{shortcode}", shortCode)
				}
			}

			if len(processedText) == 0 {
				endLen = utf8.RuneCountInString(displayURL)

				processedText = displayURL
			} else {
				startLen = utf8.RuneCountInString(processedText) + 1
				endLen = (utf8.RuneCountInString(processedText) + 1) + utf8.RuneCountInString(displayURL)

				processedText = processedText + "\n" + displayURL
			}
		}
		if formattedVideoURL != "" {
			formattedVideoURL = strings.ReplaceAll(formattedVideoURL, "{shortblob}", video.Video.Ref.Link[len(video.Video.Ref.Link)-6:])
			formattedVideoURL = strings.ReplaceAll(formattedVideoURL, "{fullblob}", video.Video.Ref.Link)
			formattedVideoURL = strings.ReplaceAll(formattedVideoURL, "{user_did}", tweet.Author.DID)
			if strings.Contains(formattedVideoURL, "{shortcode}") {
				if shortCode == "" {
					shortCode, err = CreateShortLink("/cdn/vid/bsky/" + tweet.Author.DID + "/" + video.Video.Ref.Link + "/")
					if err != nil {
						fmt.Println("Error creating short link:", err)
						formattedVideoURL = strings.ReplaceAll(formattedVideoURL, "{shortcode}", "")
					} else {
						formattedVideoURL = strings.ReplaceAll(formattedVideoURL, "{shortcode}", shortCode)
					}
				} else {
					formattedVideoURL = strings.ReplaceAll(formattedVideoURL, "{shortcode}", shortCode)
				}

			}
		}

		// Add the URL in the entities.
		tweetEntities.Urls = append(tweetEntities.Urls, bridge.URL{
			ExpandedURL: "https://video.bsky.app/watch/" + tweet.Author.DID + "/" + video.Video.Ref.Link + "/720p/video.m3u8",
			URL:         formattedVideoURL,
			DisplayURL:  displayURL,
			Start:       startLen,
			End:         endLen,
			Indices: []int{
				startLen,
				endLen,
			},
			XMLName: xml.Name{Local: "url"},
			XMLFormat: bridge.URLXMLFormat{
				Start:       startLen,
				End:         endLen,
				DisplayURL:  displayURL,
				ExpandedURL: "https://video.bsky.app/watch/" + tweet.Author.DID + "/" + video.Video.Ref.Link + "/720p/video.m3u8",
				URL:         formattedVideoURL,
			},
		})
	}

	// if isRetweet {
	// 	tweet.Author = postReason.By
	// }

	// Get the user info
	var author *bridge.TwitterUser
	author, err = blueskyapi.GetUserInfo(pds, token, tweet.Author.DID, false)
	if err != nil {
		fmt.Println("Error:", err)
		// fallback
		authorPtr := GetUserInfoFromTweetData(tweet)
		author = &authorPtr
	}

	// final object conversion.
	convertedTweet := bridge.Tweet{
		Coordinates: nil,
		Favourited:  tweet.Viewer.Like != nil,
		CreatedAt: func() string {
			if isRetweet {
				return bridge.TwitterTimeConverter(postReason.IndexedAt)
			}
			return bridge.TwitterTimeConverter(tweet.Record.CreatedAt)
		}(),
		Truncated:    false,
		Text:         processedText,
		Entities:     tweetEntities,
		Annotations:  nil, // I am curious what annotations are
		Contributors: nil,
		ID: func() int64 {
			// we have to use psudo ids because of https://github.com/bluesky-social/atproto/issues/1811
			if isRetweet {
				return *bridge.BskyMsgToTwitterID(tweet.URI, &postReason.IndexedAt, &postReason.By.DID)
			}
			return *bridge.BskyMsgToTwitterID(tweet.URI, &tweet.Record.CreatedAt, nil)
		}(),
		IDStr: func() string {
			if isRetweet {
				id := bridge.BskyMsgToTwitterID(tweet.URI, &postReason.IndexedAt, &postReason.By.DID)
				return strconv.FormatInt(*id, 10)
			}
			id := bridge.BskyMsgToTwitterID(tweet.URI, &tweet.Record.CreatedAt, nil)
			return strconv.FormatInt(*id, 10)
		}(),
		Geo:               nil,
		Place:             nil,
		PossiblySensitive: false,
		InReplyToUserID: func() *int64 {
			if replyMsgBskyURI == "" || replyUserBskyId == "" {
				return nil
			}

			id := bridge.BlueSkyToTwitterID(replyUserBskyId)
			return id
		}(),
		InReplyToUserIDStr: func() *string {
			if replyMsgBskyURI == "" || replyUserBskyId == "" {
				return nil
			}
			id := bridge.BlueSkyToTwitterID(replyUserBskyId)
			idStr := strconv.FormatInt(*id, 10)
			return &idStr
		}(),
		// InReplyToScreenName: &tweet.Author.DisplayName,
		User:   *author,
		Source: "Bluesky",
		InReplyToStatusID: func() *int64 {
			if replyMsgBskyURI == "" || replyUserBskyId == "" {
				return nil
			}
			id := bridge.BskyMsgToTwitterID(replyMsgBskyURI, replyTimeStamp, nil)
			return id
		}(),
		InReplyToStatusIDStr: func() *string {
			if replyMsgBskyURI == "" || replyUserBskyId == "" {
				return nil
			}
			id := bridge.BskyMsgToTwitterID(replyMsgBskyURI, replyTimeStamp, nil)
			idStr := strconv.FormatInt(*id, 10)
			return &idStr
		}(),
		Retweeted:    tweet.Viewer.Repost != nil && !isRetweet,
		RetweetCount: tweet.RepostCount,
		RetweetedStatus: func() *bridge.RetweetedTweet {
			if isRetweet {
				retweet_bsky := tweet
				retweet_bsky.Author = bsky_retweet_og_author
				//retweet_bsky.Viewer.Repost = nil
				translatedTweet := TranslatePostToTweet(retweet_bsky, replyMsgBskyURI, replyUserBskyId, replyTimeStamp, nil, token, pds)
				translatedTweet.CurrentUserRetweet = nil
				return &bridge.RetweetedTweet{
					Tweet: translatedTweet, // Oh how i love XML
				}
			}
			return nil
		}(),
		//TODO: If the user has retweeted, and this tweet itself is a retweet, we can't have current_user_retweet at the same time as retweeted_status
		CurrentUserRetweet: func() *bridge.CurrentUserRetweet {
			if tweet.Viewer.Repost != nil && !isRetweet {
				RepostRecord, err := blueskyapi.GetRecordWithUri(pds, *tweet.Viewer.Repost)
				if err != nil {
					fmt.Println("Error:", err)
					return nil
				}

				_, my_did, _ := blueskyapi.GetURIComponents(*tweet.Viewer.Repost)
				retweetId := bridge.BskyMsgToTwitterID(tweet.URI, &RepostRecord.Value.CreatedAt, &my_did)
				return &bridge.CurrentUserRetweet{
					ID:    *retweetId,
					IDStr: strconv.FormatInt(*retweetId, 10),
				}
			}
			return nil
		}(),
	}
	return convertedTweet
}

// This is "depercated"/a togglable option in the config (eventually)
// Primarly used as a fallback if we cannot lookup user info
func GetUserInfoFromTweetData(tweet blueskyapi.Post) bridge.TwitterUser {
	return bridge.TwitterUser{
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
		ProfileImageURL:           configData.CdnURL + "/cdn/img/?url=" + url.QueryEscape(tweet.Author.Avatar) + ":profile_bigger",
		ProfileImageURLHttps:      configData.CdnURL + "/cdn/img/?url=" + url.QueryEscape(tweet.Author.Avatar) + ":profile_bigger",
		Location:                  "Twitter",
		ProfileLinkColor:          "009999",
		FollowRequestSent:         false,
		URL:                       "",
		ScreenName:                tweet.Author.Handle,
		ContributorsEnabled:       false,
		UtcOffset:                 nil,
		IsTranslator:              false,
		ID:                        *bridge.BlueSkyToTwitterID(tweet.URI),
		IDStr:                     strconv.FormatInt(*bridge.BlueSkyToTwitterID(tweet.URI), 10),
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
	}
}

// This request is an "internal" request, and thus, these are very little to no docs. this is a problem.
// The most docs I could find: https://blog.fgribreau.com/2012/01/twitter-unofficial-api-getting-tweets.html
func TweetInfo(c *fiber.Ctx) error {
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header (yes i know this isn't complient with the twitter api)")
	}

	encodedId := c.Params("id")
	idBigInt, err := strconv.ParseInt(encodedId, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	// Fetch ID
	idPtr, _, _, err := bridge.TwitterMsgIdToBluesky(&idBigInt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	id := *idPtr

	err, thread := blueskyapi.GetPost(*pds, *oauthToken, id, 1, 0)

	if err != nil {
		return err
	}

	likes, err := blueskyapi.GetPostLikes(*pds, *oauthToken, id, 100)

	if err != nil {
		return err
	}

	reposters, err := blueskyapi.GetRetweetAuthors(*pds, *oauthToken, id, 100)

	if err != nil {
		return err
	}

	repliers := []int64{}
	favourites := []int64{}
	retweeters := []int64{}

	for _, reply := range *thread.Thread.Replies {
		repliers = append(repliers, *bridge.BlueSkyToTwitterID(reply.Post.Author.DID))
	}
	for _, like := range likes.Likes {
		favourites = append(favourites, *bridge.BlueSkyToTwitterID(like.Actor.DID))
	}
	for _, reposter := range reposters.RepostedBy {
		retweeters = append(retweeters, *bridge.BlueSkyToTwitterID(reposter.DID))
	}

	return EncodeAndSend(c, bridge.TwitterActivitiySummary{
		FavouritesCount: strconv.Itoa(thread.Thread.Post.LikeCount),
		RetweetsCount:   strconv.Itoa(thread.Thread.Post.RepostCount),
		RepliersCount:   strconv.Itoa(thread.Thread.Post.ReplyCount),
		Favourites:      favourites,
		Retweets:        retweeters,
		Repliers:        repliers,
	})
}
