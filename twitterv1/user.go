package twitterv1

import (
	"fmt"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/gofiber/fiber/v2"
)

func user_info(c *fiber.Ctx) error {
	screen_name := c.Query("screen_name")
	_, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	userinfo, err := blueskyapi.GetUserInfo(*oauthToken, screen_name)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	// user := bridge.TweetUser{
	// 	Name:                      "Preloading",
	// 	ProfileSidebarBorderColor: "C0DEED",
	// 	ProfileBackgroundTile:     false,
	// 	ProfileSidebarFillColor:   "DDEEF6",
	// 	CreatedAt:                 "Wed Sep 01 00:00:00 +0000 2021",
	// 	ProfileImageURL:           "https://cdn.bsky.app/img/avatar_thumbnail/plain/did:plc:khcyntihpu7snjszuojjgjc4/bafkreignfoswre6f2ehujkifewpk2xdlrqhfhraloaoixjf5dommpzjxeq@png",
	// 	Location:                  "San Francisco",
	// 	ProfileLinkColor:          "0084B4",
	// 	FollowRequestSent:         false,
	// 	URL:                       "http://dev.twitter.com",
	// 	FavouritesCount:           8,
	// 	ContributorsEnabled:       false,
	// 	UtcOffset:                 -28800,
	// 	ID:                        2,
	// 	ProfileUseBackgroundImage: true,
	// 	ProfileTextColor:          "333333",
	// 	Protected:                 false,
	// 	FollowersCount:            200,
	// 	Lang:                      "en",
	// 	Notifications:             false,
	// 	TimeZone:                  "Pacific Time (US & Canada)",
	// 	Verified:                  false,
	// 	ProfileBackgroundColor:    "C0DEED",
	// 	GeoEnabled:                true,
	// 	Description:               "A developer just looking to make some cool stuff",
	// 	FriendsCount:              100,
	// 	StatusesCount:             333,
	// 	ProfileBackgroundImageURL: "http://a0.twimg.com/images/themes/theme1/bg.png",
	// 	Following:                 false,
	// 	ScreenName:                screen_name,
	// }
	return c.XML(userinfo)
}
