package twitterv1

import (
	"fmt"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
)

func UserSearch(c *fiber.Ctx) error {
	searchQuery := c.Query("q")
	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}
	// Search for users
	bskyUsers, err := blueskyapi.UserSearch(*pds, *oauthToken, searchQuery)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to search")
	}
	// Get complete user info.
	// We must do this as the search API only returns a subset of the user info, and twitter wants all of it.

	// Extract the dids into a string array
	var dids []string
	for _, user := range bskyUsers {
		dids = append(dids, user.DID)
	}
	if len(dids) == 0 {
		return c.JSON([]bridge.TwitterUser{})
	}
	users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, dids, false)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
	}

	return c.JSON(users)
}
