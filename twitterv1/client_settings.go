package twitterv1

import (
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
)

// TODO: Find anything about this request
func PushDestinations(c *fiber.Ctx) error {
	// This request is /1/account/push_destinations/device.xml, and i cannot find any info on this.
	return c.SendStatus(fiber.StatusNotImplemented)
}

// TODO
// 99% sure this relies on PushDestinations, which i have no data on.
func GetSettings(c *fiber.Ctx) error {
	return c.XML(bridge.Config{
		SleepTime: bridge.SleepTime{
			EndTime:   nil,
			Enabled:   true,
			StartTime: nil,
		},
		TrendLocation: []bridge.TrendLocation{
			{
				Name:  "Worldwide",
				Woeid: 1,
				PlaceType: bridge.PlaceType{
					Name: "Supername",
					Code: 19,
				},
				Country:     "",
				URL:         "http://where.yahooapis.com/v1/place/1",
				CountryCode: nil,
			},
		},
		Language:            "en",
		AlwaysUseHttps:      false,
		DiscoverableByEmail: true,
		TimeZone: bridge.TimeZone{
			Name:       "Pacific Time (US & Canada)",
			TzinfoName: "America/Los_Angeles",
			UtcOffset:  -28800,
		},
		GeoEnabled: true,
	})

}
