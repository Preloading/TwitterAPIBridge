package twitterv1

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/Preloading/TwitterAPIBridge/db_controller"
	"github.com/gofiber/fiber/v2"
)

var mastodonRegex, _ = regexp.Compile("\n\\[bridged from .* on the fediverse by fed\\.brid\\.gy \\]$")

type TweetsRoot struct {
	XMLName  xml.Name `xml:"statuses"`
	Statuses []bridge.Tweet
}

func home_timeline(c *fiber.Ctx) error {
	return convert_timeline(c, "", true, blueskyapi.GetTimeline)
}

func hot_post_timeline(c *fiber.Ctx) error {
	return convert_timeline(c, "", true, blueskyapi.GetHotPosts)
}

func user_timeline(c *fiber.Ctx) error {
	actor := c.Query("screen_name")
	if actor == "" {
		actor = c.Query("user_id")
		if actor == "" {
			return ReturnError(c, "No user provided", 195, fiber.StatusForbidden)
		}
		actorInt, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid user_id provided", 195, fiber.StatusForbidden)
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&actorInt)
		if err != nil {
			return ReturnError(c, "Invalid user_id provided", 195, fiber.StatusForbidden)
		}
		actor = *actorPtr
	}
	return convert_timeline(c, actor, false, blueskyapi.GetUserTimeline)
}

func media_timeline(c *fiber.Ctx) error {
	actor := c.Query("screen_name")
	if actor == "" {
		actor = c.Query("user_id")
		if actor == "" {
			return ReturnError(c, "No user provided", 195, fiber.StatusForbidden)
		}
		actorInt, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid user_id provided", 195, fiber.StatusForbidden)
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&actorInt)
		if err != nil {
			return ReturnError(c, "Invalid user_id provided", 195, fiber.StatusForbidden)
		}
		actor = *actorPtr
	}
	return convert_timeline(c, actor, false, blueskyapi.GetMediaTimeline)
}

func likes_timeline(c *fiber.Ctx) error {
	// We shall pretend that the only thing it can be is a user id. TODO: maybe rectify this later
	actor := c.Params("id")
	actorInt, err := strconv.ParseInt(actor, 10, 64)
	if err != nil {
		return ReturnError(c, "Invalid user_id provided", 195, fiber.StatusForbidden)
	}
	actorPtr, err := bridge.TwitterIDToBlueSky(&actorInt)
	if err != nil {
		return ReturnError(c, "Invalid user_id provided", 195, fiber.StatusForbidden)
	}
	actor = *actorPtr

	return convert_timeline(c, actor, false, blueskyapi.GetActorLikes)
}

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/get/statuses/home_timeline
func convert_timeline(c *fiber.Ctx, param string, requireAuth bool, fetcher func(string, string, string, string, int) (*blueskyapi.Timeline, error)) error {
	// Get all of our keys, beeps, and bops
	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil && requireAuth {
		return MissingAuth(c, err)
	}

	// Limits
	limitStr := c.Query("count")
	if limitStr == "" {
		limitStr = "20"
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return ReturnError(c, "Invalid count provided", 195, fiber.StatusForbidden)
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
			return ReturnError(c, "Invalid max_id format", 195, fiber.StatusForbidden)
		}
		_, date, _, err := bridge.TwitterMsgIdToBluesky(&maxIDInt)
		if err != nil {
			return ReturnError(c, "max_id was not found", 144, fiber.StatusForbidden)
		}
		context = date.Format(time.RFC3339)
	}

	since_id := c.Query("since_id")
	since_date := time.Time{}
	hasSinceDate := false

	// Handle getting things in the past
	if since_id != "" {
		// Get the timeline context from the DB
		sinceIdInt, err := strconv.ParseInt(since_id, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid since_id format", 195, fiber.StatusForbidden)
		}
		_, tempDate, _, err := bridge.TwitterMsgIdToBluesky(&sinceIdInt)
		if err != nil {
			return ReturnError(c, "since_id was not found", 144, fiber.StatusForbidden)
		}
		since_date = *tempDate
		hasSinceDate = true
	}

	fmt.Println("Context:", context)
	res, err := fetcher(*pds, *oauthToken, context, param, limit)
	if err != nil {
		return HandleBlueskyError(c, err.Error(), "app.bsky.feed.defs#timeline", func(c *fiber.Ctx) error { // dislike the "lexicon", but its fine.
			return convert_timeline(c, param, requireAuth, fetcher)
		})
	}

	if hasSinceDate {
		filteredFeed := []blueskyapi.Feed{}
		for _, item := range res.Feed {
			postTime := item.Post.IndexedAt
			if postTime.After(since_date) {
				filteredFeed = append(filteredFeed, item)
			}
		}
		res.Feed = filteredFeed
	}

	fmt.Println(len(res.Feed))

	// Caching the user DIDs efficiently
	userDIDs := []string{}

	for _, item := range res.Feed {
		if !slices.Contains(userDIDs, item.Post.Author.DID) {
			userDIDs = append(userDIDs, item.Post.Author.DID)
		}
	}

	blueskyapi.GetUsersInfo(*pds, *oauthToken, userDIDs, false) // fill cache

	// Translate the posts to tweets
	tweets := []bridge.Tweet{}

	for _, item := range res.Feed {
		tweets = append(tweets, TranslatePostToTweet(item.Post, item.Reply.Parent.URI, item.Reply.Parent.Author.DID, item.Reply.Parent.Author.Handle, &item.Reply.Parent.Record.CreatedAt.Time, item.Reason, *oauthToken, *pds))
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
		return ReturnError(c, "Invalid ID format", 195, 403)
	}
	// Fetch ID
	uriPtr, _, _, err := bridge.TwitterMsgIdToBluesky(&idInt)
	if err != nil {
		return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
	}
	uri := *uriPtr

	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	thread, err := blueskyapi.GetPost(*pds, *oauthToken, uri, 1, 0)

	if err != nil {
		return HandleBlueskyError(c, err.Error(), "app.bsky.feed.getPostThread", RelatedResults)
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
		reply.Post.Record.CreatedAt = blueskyapi.FTime{Time: reply.Post.IndexedAt}
		twitterReplies.Results = append(twitterReplies.Results, bridge.Results{
			Kind:  "Tweet",
			Score: 1.0,
			Value: TranslatePostToTweet(reply.Post, uri, strconv.FormatInt(*postAuthor, 10), reply.Post.Author.Handle, &thread.Thread.Post.Record.CreatedAt.Time, nil, *oauthToken, *pds),
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
		return ReturnError(c, "Invalid ID format", 195, 403)
	}
	// Fetch ID
	uriPtr, _, _, err := bridge.TwitterMsgIdToBluesky(&idInt)
	if err != nil {
		fmt.Println("Error:", err)
		return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
	}
	uri := *uriPtr

	fmt.Println("URI:", uri)
	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		emptyString := ""
		oauthToken = &emptyString
	}

	thread, err := blueskyapi.GetPost(*pds, *oauthToken, uri, 0, 1)

	if err != nil {
		return HandleBlueskyError(c, err.Error(), "app.bsky.feed.getPostThread", GetStatusFromId)
	}

	// TODO: Some things may be needed for reposts to show up correctly. thats a later problem :)
	if thread.Thread.Parent == nil {
		return EncodeAndSend(c, TranslatePostToTweet(thread.Thread.Post, "", "", "", nil, nil, *oauthToken, *pds))
	} else {
		return EncodeAndSend(c, TranslatePostToTweet(thread.Thread.Post, thread.Thread.Parent.Post.URI, thread.Thread.Parent.Post.Author.DID, thread.Thread.Parent.Post.Author.Handle, &thread.Thread.Parent.Post.Record.CreatedAt.Time, nil, *oauthToken, *pds))
	}
}

// This gigantic function is used to convert the bluesky post format, into a format that is compatible with the twitter API.
// https://web.archive.org/web/20120506182126/https://dev.twitter.com/docs/platform-objects/tweets
func TranslatePostToTweet(tweet blueskyapi.Post, replyMsgBskyURI string, replyUserBskyId string, replyUserHandle string, replyTimeStamp *time.Time, postReason *blueskyapi.PostReason, token string, pds string) bridge.Tweet {
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

	// Quote replies are different for some reason, app.bsky.embed.recordWithMedia

	// Images

	bskyimages := tweet.Record.Embed.Images
	if tweet.Record.Embed.Media != nil {
		bskyimages = append(bskyimages, tweet.Record.Embed.Media.Images...)
	}
	for _, image := range bskyimages {
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
				shortCode, err = CreateShortLink("/cdn/img/bsky/"+tweet.Author.DID+"/"+image.Image.Ref.Link+".jpg", "i")
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
					shortCode, err = CreateShortLink("/cdn/img/bsky/"+tweet.Author.DID+"/"+image.Image.Ref.Link+".jpg", "i")
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
			XMLFormat: bridge.MediaXML{
				Start:         startLen,
				End:           endLen,
				Type:          "photo",
				ID:            int64(id),
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
			},
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
		})
		id++
	}

	// GIFs
	if tweet.Record.Embed.External != nil && tweet.Record.Embed.Type == "app.bsky.embed.external" && strings.HasPrefix(tweet.Record.Embed.External.Uri, "https://media.tenor.com/") {
		startLen, endLen := 0, 0
		formattedImageURL := configData.GifURLText
		displayURL := configData.GifDisplayText
		shortCode := ""
		if displayURL != "" {
			if strings.Contains(displayURL, "{shortcode}") {
				shortCode, err = CreateShortLink(tweet.Record.Embed.External.Uri, "g")
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
			if strings.Contains(formattedImageURL, "{shortcode}") {
				if shortCode == "" {
					shortCode, err = CreateShortLink(tweet.Record.Embed.External.Uri, "g")
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

		// url parsing jank to get the width and height
		w := 200
		h := 200

		u, err := url.Parse(tweet.Record.Embed.External.Uri)
		if err == nil {
			q := u.Query()
			widthStr := q.Get("ww")
			heightStr := q.Get("hh")
			fmt.Sscanf(widthStr, "%d", &w)
			fmt.Sscanf(heightStr, "%d", &h)
		}

		mediaWebURL := configData.CdnURL + "/cdn/img/bsky/" + tweet.Author.DID + "/" + tweet.Record.Embed.External.Thumb.Ref.Link + ".jpg"

		// Process each image
		tweetEntities.Media = append(tweetEntities.Media, bridge.Media{
			XMLFormat: bridge.MediaXML{
				Start:         startLen,
				End:           endLen,
				Type:          "photo",
				ID:            int64(id),
				MediaURL:      mediaWebURL,
				MediaURLHttps: mediaWebURL,

				DisplayURL:  displayURL,
				ExpandedURL: tweet.Record.Embed.External.Uri,
				URL:         formattedImageURL,

				Sizes: bridge.MediaSize{
					Thumb: func() bridge.Size {
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
						W:      w,
						H:      h,
						Resize: "fit",
					},
				},
			},
			Type:          "photo",
			ID:            int64(id),
			IDStr:         strconv.Itoa(id),
			MediaURL:      mediaWebURL,
			MediaURLHttps: mediaWebURL,

			DisplayURL:  displayURL,
			ExpandedURL: tweet.Record.Embed.External.Uri,
			URL:         formattedImageURL,

			Sizes: bridge.MediaSize{
				Thumb: func() bridge.Size {
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
					W:      w,
					H:      h,
					Resize: "fit",
				},
			},

			Indices: []int{
				startLen,
				endLen,
			},
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

	if tweet.Record.Embed.Video != nil && tweet.Record.Embed.Video.Video != nil {
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
				shortCode, err = CreateShortLink("/cdn/vid/bsky/"+tweet.Author.DID+"/"+video.Video.Ref.Link+"/", "v")
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
					shortCode, err = CreateShortLink("/cdn/vid/bsky/"+tweet.Author.DID+"/"+video.Video.Ref.Link+"/", "v")
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
			return bridge.TwitterTimeConverter(tweet.Record.CreatedAt.Time)
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
			return *bridge.BskyMsgToTwitterID(tweet.URI, &tweet.Record.CreatedAt.Time, nil)
		}(),
		IDStr: func() string {
			if isRetweet {
				id := bridge.BskyMsgToTwitterID(tweet.URI, &postReason.IndexedAt, &postReason.By.DID)
				return strconv.FormatInt(*id, 10)
			}
			id := bridge.BskyMsgToTwitterID(tweet.URI, &tweet.Record.CreatedAt.Time, nil)
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
		User: *author,
		Source: func() string {
			// small lil easter egg, if the account is bridged thru bridgy fed, we change the source to mastodon.
			if mastodonRegex.MatchString(tweet.Author.Description) {
				return "Mastodon"
			} else {
				return "Bluesky"
			}
		}(),
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
				translatedTweet := TranslatePostToTweet(retweet_bsky, replyMsgBskyURI, replyUserBskyId, replyUserHandle, replyTimeStamp, nil, token, pds)
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
				retweetId := bridge.BskyMsgToTwitterID(tweet.URI, &RepostRecord.Value.CreatedAt.Time, &my_did)
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
		CreatedAt:                 bridge.TwitterTimeConverter(tweet.Author.Associated.CreatedAt.Time),
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
		return MissingAuth(c, err)
	}

	encodedId := c.Params("id")
	idBigInt, err := strconv.ParseInt(encodedId, 10, 64)
	if err != nil {
		return ReturnError(c, "Invalid ID format", 195, 403)
	}
	// Fetch ID
	idPtr, _, _, err := bridge.TwitterMsgIdToBluesky(&idBigInt)
	if err != nil {
		return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
	}
	id := *idPtr

	thread, err := blueskyapi.GetPost(*pds, *oauthToken, id, 1, 0)

	if err != nil {
		return HandleBlueskyError(c, err.Error(), "app.bsky.feed.getPostThread", TweetInfo)
	}

	likes, err := blueskyapi.GetPostLikes(*pds, *oauthToken, id, 100)

	if err != nil {
		return HandleBlueskyError(c, err.Error(), "app.bsky.feed.getLikes", TweetInfo)
	}

	reposters, err := blueskyapi.GetRetweetAuthors(*pds, *oauthToken, id, 100)

	if err != nil {
		return HandleBlueskyError(c, err.Error(), "app.bsky.feed.getRepostedBy", TweetInfo)
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

// Mentions timeline, using notifications to make my life hell
func mentions_timeline(c *fiber.Ctx) error {
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c, err)
	}

	// Handle pagination
	context := ""
	max_id := c.Query("max_id")
	// Handle getting things in the past
	if max_id != "" {
		// Get the timeline context from the DB
		maxIDInt, err := strconv.ParseInt(max_id, 10, 64)
		fmt.Println("Max ID:", maxIDInt)
		if err != nil {
			return ReturnError(c, "Invalid max_id format", 195, fiber.StatusForbidden)
		}
		_, date, _, err := bridge.TwitterMsgIdToBluesky(&maxIDInt)
		if err != nil {
			return ReturnError(c, "max_id was not found", 144, fiber.StatusForbidden)
		}
		context = date.Format(time.RFC3339)
	}

	// Handle count
	count := 20
	if countStr := c.Query("count"); countStr != "" {
		if countInt, err := strconv.Atoi(countStr); err == nil {
			count = countInt
		}
	}
	if count > 100 {
		count = 100
	}

	// Get notifications
	bskyNotifications, err := blueskyapi.GetMentions(*pds, *oauthToken, count, context)
	if err != nil {
		return HandleBlueskyError(c, err.Error(), "app.bsky.notification.listNotifications", mentions_timeline)
	}

	// Track unique users and posts
	uniqueUsers := make(map[string]bool)
	uniquePosts := make(map[string]bool)

	// First pass: collect unique users and posts
	for _, notification := range bskyNotifications.Notifications {
		uniqueUsers[notification.Author.DID] = true
		uniquePosts[notification.URI] = true
	}

	// Convert maps to slices
	usersToLookUp := make([]string, 0, len(uniqueUsers))
	postsToLookUp := make([]string, 0, len(uniquePosts))
	for user := range uniqueUsers {
		usersToLookUp = append(usersToLookUp, user)
	}
	for post := range uniquePosts {
		postsToLookUp = append(postsToLookUp, post)
	}

	// Create thread-safe maps for results
	var userCache sync.Map
	var postCache sync.Map

	// Process in parallel
	var wg sync.WaitGroup

	// Fetch users in chunks
	wg.Add(1)
	go func() {
		defer wg.Done()
		users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, usersToLookUp, false)
		if err == nil {
			for _, user := range users {
				userCache.Store(user.ScreenName[strings.LastIndex(user.ScreenName, "/")+1:], user)
			}
		}
	}()

	// Fetch posts in parallel chunks
	postChunks := chunkSlice(postsToLookUp, 10)
	for _, chunk := range postChunks {
		wg.Add(1)
		go func(posts []string) {
			defer wg.Done()
			for _, postID := range posts {
				if post, err := blueskyapi.GetPost(*pds, *oauthToken, postID, 0, 1); err == nil {
					tweet := TranslatePostToTweet(
						post.Thread.Post,
						func() string {
							if post.Thread.Parent != nil {
								return post.Thread.Parent.Post.URI
							}
							return ""
						}(),
						func() string {
							if post.Thread.Parent != nil {
								return post.Thread.Parent.Post.Author.DID
							}
							return ""
						}(),
						func() string {
							if post.Thread.Parent != nil {
								return post.Thread.Parent.Post.Author.Handle
							}
							return ""
						}(),
						func() *time.Time {
							if post.Thread.Parent != nil {
								return &post.Thread.Parent.Post.IndexedAt
							}
							return nil
						}(),
						nil,
						*oauthToken,
						*pds,
					)
					postCache.Store(postID, &tweet)
				}
			}
		}(chunk)
	}

	wg.Wait()

	// Convert notifications to tweets timeline
	tweets := []bridge.Tweet{}
	for _, notification := range bskyNotifications.Notifications {
		if post, ok := postCache.Load(notification.URI); ok {
			tweet := post.(*bridge.Tweet)
			tweets = append(tweets, *tweet)
		}
	}

	if c.Params("filetype") == "xml" {
		tweetsRoot := TweetsRoot{
			Statuses: tweets,
		}
		return EncodeAndSend(c, tweetsRoot)
	}

	return EncodeAndSend(c, tweets)
}
