package twitterv1

import (
	"fmt"
	"math/big"
	"strings"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
)

// groupUsers splits a slice of users into chunks of the specified size
func groupUsers(users []string, size int) [][]string {
	var groups [][]string
	for size < len(users) {
		users, groups = users[size:], append(groups, users[0:size:size])
	}
	groups = append(groups, users)
	return groups
}

// https://web.archive.org/web/20120508075505/https://dev.twitter.com/docs/api/1/get/users/show
func user_info(c *fiber.Ctx) error {
	screen_name := c.Query("screen_name")

	if screen_name == "" {
		userIDStr := c.Query("user_id")
		userID, ok := new(big.Int).SetString(userIDStr, 10)
		if !ok {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		screen_name = bridge.TwitterIDToBlueSky(userID) // yup
		if screen_name == "" {
			return c.Status(fiber.StatusBadRequest).SendString("No screen_name or user_id provided")

		}

	}

	_, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	userinfo, err := blueskyapi.GetUserInfo(*oauthToken, screen_name)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	xml, err := bridge.XMLEncoder(userinfo, "TwitterUser", "user")
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode user info")
	}

	return c.SendString(*xml)
}

// https://web.archive.org/web/20120508165240/https://dev.twitter.com/docs/api/1/get/users/lookup
func UsersLookup(c *fiber.Ctx) error {
	screen_name := c.Query("screen_name")
	user_id := c.Query("user_id")
	var usersToLookUp []string

	if screen_name != "" {
		usersToLookUp = strings.Split(screen_name, ",")
	} else if user_id != "" {
		userIDs := strings.Split(user_id, ",")
		for _, idStr := range userIDs {
			userID, ok := new(big.Int).SetString(idStr, 10)
			if !ok {
				return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
			}
			handle := bridge.TwitterIDToBlueSky(userID)
			if handle != "" {
				usersToLookUp = append(usersToLookUp, handle)
			}
		}
	} else {
		return c.Status(fiber.StatusBadRequest).SendString("No screen_name or user_id provided")
	}

	if len(usersToLookUp) > 100 {
		return c.Status(fiber.StatusBadRequest).SendString("Too many users to look up")
	}

	_, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	// here's some fun problems!
	// twitter api's max is 100 users per call. bluesky's is 25. so we get to lookup in multiple requests

	users, err := LookupUsers(usersToLookUp, oauthToken)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}
	return c.JSON(users)
}

func LookupUsers(usersToLookUp []string, oauthToken *string) ([]bridge.TwitterUser, error) {
	userLookupGroups := groupUsers(usersToLookUp, 25)
	var users []bridge.TwitterUser

	for _, group := range userLookupGroups {
		usersGroup, err := blueskyapi.GetUsersInfo(*oauthToken, group)
		if err != nil {
			return nil, err
		}
		for _, user := range usersGroup {
			users = append(users, *user)
		}
	}
	return users, nil
}
