package twitterv1

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
)

// TODO: Implement this
func Search(c *fiber.Ctx) error {
	q := c.Query("q")
	fmt.Println("Search query:", q)
	return c.SendStatus(fiber.StatusNotImplemented)
}

// https://web.archive.org/web/20120313235613/https://dev.twitter.com/docs/api/1/get/trends/%3Awoeid
// The bluesky feature to make this possible was released 17 hours ago, and is "beta", so this is likely to break
func trends_woeid(c *fiber.Ctx) error {
	// We don't have location specific trends soooooo
	// woeid := c.Params("woeid")

	//auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
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

	return c.JSON(bridge.Trends{
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
