package twitterv1

import (
	"fmt"
	"math/big"
	"strings"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
)

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

	users, err := blueskyapi.GetUsersInfo(*oauthToken, usersToLookUp, false)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}
	return c.JSON(users)
}

// another xml endpoint
// https://github.com/RuaanV/MyTwit/blob/3466157350ad8ce2ca4e3503ae3cc5bbbe3d3de4/MyTwit/LinqToTwitterAg/Friendship/FriendshipRequestProcessor.cs#L118
// and
// https://web.archive.org/web/20120516155714/https://dev.twitter.com/docs/api/1/get/friendships/lookup
func UserRelationships(c *fiber.Ctx) error {
	actors := c.Query("user_id")
	if actors == "" {
		actors = c.Query("screen_name")
		if actors == "" {
			return c.Status(fiber.StatusBadRequest).SendString("No user_id provided")
		}
	}

	_, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	actorsArray := strings.Split(actors, ",")
	if len(actorsArray) > 100 {
		return c.Status(fiber.StatusBadRequest).SendString("Too many users to look up")
	}

	relationships := []bridge.UserRelationship{}
	users, err := blueskyapi.GetUsersInfoRaw(*oauthToken, actorsArray, false)
	for _, user := range users {
		encodedUserId := *bridge.BlueSkyToTwitterID(user.DID)
		if err != nil {
			fmt.Println("Error:", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode user id")
		}

		connections := []string{}
		if user.Viewer.Following != nil {
			connections = append(connections, "following")
		}
		if user.Viewer.FollowedBy != nil {
			connections = append(connections, "followed_by")
		}
		if user.Viewer.Blocking != nil {
			connections = append(connections, "blocking") // Complete guess
		}
		if user.Viewer.BlockedBy {
			connections = append(connections, "blocked_by") // Complete guess
		}

		relationships = append(relationships, bridge.UserRelationship{
			Name: func() string {
				if user.DisplayName == "" {
					return user.Handle
				}
				return user.DisplayName
			}(),
			ScreenName:  user.Handle,
			ID:          encodedUserId,
			IDStr:       encodedUserId.String(),
			Connections: connections,
		})
	}

	// i hate xml

	root := bridge.UserRelationships{
		Relationships: relationships,
	}

	xml, err := bridge.XMLEncoder(root, "UserRelationships", "relationships")
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode user relationships")
	}

	newXml := strings.ReplaceAll(*xml, "<UserRelationship>", "<relationship>")
	newXml = strings.ReplaceAll(newXml, "</UserRelationship>", "</relationship>")

	return c.SendString(newXml)
}
