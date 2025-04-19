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

func SuggestedTopics(c *fiber.Ctx) error {
	// I think this is hard coded in?
	// It expects a size, but uhhh, it can be unlimited on bsky sooooo...
	// It might be worth it later to get the config for this
	// Also localization
	suggestions := []bridge.TopicSuggestion{
		{
			Name: "Animals",
			Slug: "animals",
			Size: 20,
		},
		{
			Name: "Art",
			Slug: "art",
			Size: 20,
		},
		{
			Name: "Books",
			Slug: "books",
			Size: 20,
		},
		{
			Name: "Comedy",
			Slug: "comedy",
			Size: 20,
		},
		{
			Name: "Comics",
			Slug: "comics",
			Size: 20,
		},
		{
			Name: "Culture",
			Slug: "culture",
			Size: 20,
		},
		{
			Name: "Software Dev",
			Slug: "dev",
			Size: 20,
		},
		{
			Name: "Education",
			Slug: "education",
			Size: 20,
		},
		{
			Name: "Food",
			Slug: "food",
			Size: 20,
		},
		{
			Name: "Video Games",
			Slug: "gaming",
			Size: 20,
		},
		{
			Name: "Journalism",
			Slug: "journalism",
			Size: 20,
		},
		{
			Name: "Movies",
			Slug: "movies",
			Size: 20,
		},
		{
			Name: "Music",
			Slug: "music",
			Size: 20,
		},
		{
			Name: "Nature",
			Slug: "nature",
			Size: 20,
		},
		{
			Name: "News",
			Slug: "news",
			Size: 20,
		},
		{
			Name: "Pets",
			Slug: "pets",
			Size: 20,
		},
		{
			Name: "Photography",
			Slug: "photography",
			Size: 20,
		},
		{
			Name: "Politics",
			Slug: "politics",
			Size: 20,
		},
		{
			Name: "Science",
			Slug: "science",
			Size: 20,
		},
		{
			Name: "Sports",
			Slug: "sports",
			Size: 20,
		},
		{
			Name: "Tech",
			Slug: "tech",
			Size: 20,
		},
		{
			Name: "TV",
			Slug: "tv",
			Size: 20,
		},
		{
			Name: "Writers",
			Slug: "writers",
			Size: 20,
		},
	}
	return EncodeAndSend(c, suggestions)
}
