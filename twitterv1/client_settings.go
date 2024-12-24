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
	// environment := c.Query("environment")

	// 	app_version=4.1.3&
	// device_model=iPhone&
	// device_name=Logan%E2%80%99s%20iPhone&
	// enabled_for=23&
	// environment=3&
	// language=en&
	// old_udid=d89b164326e0c50494438d5bd360988c53e672f0&
	// send_error_codes=true&
	// system_name=iPhone%20OS&
	// system_version=4.2.1&
	// token=u1um37SQzAoE3yF%2F0DVZLQFPk4Ssie%2FGTkS1rMIZk4c%3D&
	// udid=291C3725-6221-4B96-A897-3436AE9D48DF

	return c.SendString(fmt.Sprintf(`
	<?xml version="1.0" encoding="UTF-8"?>
	<device>
		<device_model>iPhone</device_model>
		<device_name>Logan’s iPhone</device_name>
		<enabled_for>23</enabled_for>
		<language>en</language>
		<send_error_codes>true</send_error_codes>
		<system_name>iPhone OS</system_name>
		<system_version>4.2.1</system_version>
		<token>u1um37SQzAoE3yF/0DVZLQFPk4Ssie/GTkS1rMIZk4c=</token>
		<udid>%s</udid>
		<old_udid>%s</old_udid>
		<environment>3</environment>
	<device>
	`, udid, old_udid))
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
