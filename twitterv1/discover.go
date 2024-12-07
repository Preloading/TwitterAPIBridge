package twitterv1

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// TODO: Implement this
func Search(c *fiber.Ctx) error {
	q := c.Query("q")
	fmt.Println("Search query:", q)
	return c.SendStatus(fiber.StatusNotImplemented)
}

// https://web.archive.org/web/20120313235613/https://dev.twitter.com/docs/api/1/get/trends/%3Awoeid
// For now, we will be pretending WOEID doesn't exist
// TODO: Implement this with data from bsky
func trends_woeid(c *fiber.Ctx) error {
	type Trends struct {
		Created string `json:"created_at"`
		Trends  []struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			Promoted    bool   `json:"promoted"`
			Query       string `json:"query"`
			TweetVolume int    `json:"tweet_volume"`
		} `json:"trends"`
		AsOf      string `json:"as_of"`
		Locations []struct {
			Name  string `json:"name"`
			WOEID int    `json:"woeid"`
		} `json:"locations"`
	}

	return c.JSON(Trends{
		Created: "2021-09-01T00:00:00Z",
		Trends: []struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			Promoted    bool   `json:"promoted"`
			Query       string `json:"query"`
			TweetVolume int    `json:"tweet_volume"`
		}{
			{
				Name:        "Trending Topic",
				URL:         "https://twitter.com/search?q=%22Trending%20Topic%22",
				Promoted:    false,
				Query:       "%22Trending%20Topic%22",
				TweetVolume: 10000,
			},
		},
		AsOf: "2021-09-01T00:00:00Z",
		Locations: []struct {
			Name  string `json:"name"`
			WOEID int    `json:"woeid"`
		}{
			{
				Name:  "Worldwide",
				WOEID: 1,
			},
		},
	})
}
