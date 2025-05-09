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
		return MissingAuth(c)
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
	// auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	err, thread := blueskyapi.GetPost(*pds, *oauthToken, "at://did:plc:khcyntihpu7snjszuojjgjc4/app.bsky.feed.post/3lfgrcq4di22c", 0, 1)

	if err != nil {
		return err
	}

	var displayTweet bridge.Tweet

	// TODO: Some things may be needed for reposts to show up correctly. thats a later problem :)
	if thread.Thread.Parent == nil {
		displayTweet = TranslatePostToTweet(thread.Thread.Post, "", "", "", nil, nil, *oauthToken, *pds)
	} else {
		displayTweet = TranslatePostToTweet(thread.Thread.Post, thread.Thread.Parent.Post.URI, thread.Thread.Parent.Post.Author.DID, thread.Thread.Parent.Post.Author.Handle, &thread.Thread.Parent.Post.Record.CreatedAt.Time, nil, *oauthToken, *pds)
	}

	return EncodeAndSend(c, bridge.Discovery{
		Statuses: []bridge.Tweet{
			displayTweet,
		},
		Stories: []bridge.Story{
			{
				Type:  "news",
				Score: 0.92,
				Data: bridge.StoryData{
					Title: "Thank you for using A Twitter Bridge!",
					Articles: []bridge.NewsArticle{
						{
							Title: "Thank you for using A Twitter Bridge!",
							Url: bridge.StoryURL{
								DisplayURL:  "twitterbridge.loganserver.net",
								ExpandedURL: "https://twitterbridge.loganserver.net",
							},
							TweetCount: 1500,
							Media: []bridge.StoryMediaInfo{
								{
									Type:     "image", // ?
									MediaURL: "https://raw.githubusercontent.com/Preloading/TwitterAPIBridge/refs/heads/main/resources/1.png",
									Width:    1920,
									Height:   1080,
								},
							},
						},
					},
				},
				SocialProof: bridge.SocialProof{
					Type: "social",
					ReferencedBy: bridge.SocialProofedReferencedBy{
						GlobalCount: 2500,
						Statuses:    []bridge.Tweet{displayTweet},
					},
				},
			},
		},

		RelatedQueries: []bridge.RelatedQuery{
			{
				Query: "Bluetweety",
			},
			{
				Query: "A Twitter Bridge",
			},
		},
		SpellingCorrections: []bridge.SpellingCorrection{},
	})
}

// Topics from bluesky
var topicLookup = map[string]string{
	"animals":     "Animals",
	"art":         "Art",
	"books":       "Books",
	"comedy":      "Comedy",
	"comics":      "Comics",
	"culture":     "Culture",
	"dev":         "Software Dev",
	"education":   "Education",
	"food":        "Food",
	"gaming":      "Video Games",
	"journalism":  "Journalism",
	"movies":      "Movies",
	"music":       "Music",
	"nature":      "Nature",
	"news":        "News",
	"pets":        "Pets",
	"photography": "Photography",
	"politics":    "Politics",
	"science":     "Science",
	"sports":      "Sports",
	"tech":        "Tech",
	"tv":          "TV",
	"writers":     "Writers",
}

// https://web.archive.org/web/20120516160451/https://dev.twitter.com/docs/api/1/get/users/suggestions
func SuggestedTopics(c *fiber.Ctx) error {
	// I think this is hard coded in?
	// It expects a size, but uhhh, it can be unlimited on bsky sooooo...
	// It might be worth it later to get the config for this
	// Also localization
	suggestions := []bridge.TopicSuggestion{}

	for slug, name := range topicLookup {
		suggestions = append(suggestions, bridge.TopicSuggestion{
			Name: name,
			Slug: slug,
			Size: 20,
		})
	}
	return EncodeAndSend(c, suggestions)
}

// https://web.archive.org/web/20120516160741/https://dev.twitter.com/docs/api/1/get/users/suggestions/%3Aslug
func GetTopicSuggestedUsers(c *fiber.Ctx) error {
	var err error
	// limits
	limit := 20
	if c.Query("limit") != "" {
		limit, err = strconv.Atoi(c.Query("limit"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid limit value")
		}
	}

	// auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c)
	}

	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusInternalServerError).SendString("Missing Slug")
	}

	name, exists := topicLookup[slug]
	if !exists {
		return c.Status(fiber.StatusNotFound).SendString("Invalid topic slug")
	}

	recommendedUsers, err := blueskyapi.GetTopicSuggestedUsers(*pds, *oauthToken, limit, slug)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch suggested users")
	}

	usersDID := []string{}
	for _, user := range recommendedUsers {
		usersDID = append(usersDID, user.DID)
	}

	usersInfo, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, usersDID, false)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	topic := bridge.TopicUserSuggestions{
		Name:  name,
		Slug:  slug,
		Size:  len(usersInfo),
		Users: usersInfo,
	}

	return EncodeAndSend(c, topic)
}
