package twitterv1

import (
	"fmt"

	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
)

// Thanks to bag.xml for helping me get what this request returns
func PushDestinations(c *fiber.Ctx) error {
	old_udid := c.Query("old_udid")
	udid := c.Query("udid")
	environment := c.Query("environment")

	return c.SendString(fmt.Sprintf(`
	<?xml version="1.0" encoding="UTF-8"?>
	<push_notifications>
		<udid>%s</udid>
		<old_udid>%s</old_udid>
		<environment>%s</environment>
	</push_notifications>
	`, udid, old_udid, environment))
}

// TODO
func GetSettings(c *fiber.Ctx) error {
	settings := bridge.Config{
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
	}
	xml, err := bridge.XMLEncoder(settings, "Config", "settings")
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode settings")
	}
	return c.SendString(*xml)

}
