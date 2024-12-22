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
		screen_name = bridge.TwitterIDToBlueSky(*userID) // yup
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
			handle := bridge.TwitterIDToBlueSky(*userID)
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

// Gets the relationship between the authenticated user and the users specified
// another xml endpoint
// https://github.com/RuaanV/MyTwit/blob/3466157350ad8ce2ca4e3503ae3cc5bbbe3d3de4/MyTwit/LinqToTwitterAg/Friendship/FriendshipRequestProcessor.cs#L118
// and
// https://web.archive.org/web/20120516155714/https://dev.twitter.com/docs/api/1/get/friendships/lookup
func UserRelationships(c *fiber.Ctx) error {
	actors := c.Query("user_id")
	isID := true
	if actors == "" {
		actors = c.Query("screen_name")
		isID = false
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
	if isID {
		for i, actor := range actorsArray {
			actorID, ok := new(big.Int).SetString(actor, 10)
			if !ok {
				return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
			}
			actorsArray[i] = bridge.TwitterIDToBlueSky(*actorID)
		}
	}

	relationships := []bridge.UsersRelationship{}
	users, err := blueskyapi.GetUsersInfoRaw(*oauthToken, actorsArray, false)
	for _, user := range users {
		encodedUserId := *bridge.BlueSkyToTwitterID(user.DID)
		if err != nil {
			fmt.Println("Error:", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode user id")
		}

		connections := bridge.Connections{}

		if user.Viewer.Following != nil {
			connections.Connection = append(connections.Connection, bridge.Connection{Value: "following"})
		}
		if user.Viewer.FollowedBy != nil {
			connections.Connection = append(connections.Connection, bridge.Connection{Value: "followed_by"})
		}
		if user.Viewer.Blocking != nil {
			connections.Connection = append(connections.Connection, bridge.Connection{Value: "blocked"})
		}
		if user.Viewer.BlockedBy {
			connections.Connection = append(connections.Connection, bridge.Connection{Value: "blocked_by"}) // Complete guess
		}

		relationships = append(relationships, bridge.UsersRelationship{
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

// Gets the relationship between two users
// https://web.archive.org/web/20120516154953/https://dev.twitter.com/docs/api/1/get/friendships/show
func GetUsersRelationship(c *fiber.Ctx) error {
	// Get the actors
	sourceActor := c.Query("source_id")
	if sourceActor == "" {
		sourceActor = c.Query("source_screen_name")
		if sourceActor == "" {
			return c.Status(fiber.StatusBadRequest).SendString("No source_id provided")
		}
	} else {
		sourceIDInt, ok := new(big.Int).SetString(sourceActor, 10)
		if !ok {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid source_id provided")
		}
		sourceActor = bridge.TwitterIDToBlueSky(*sourceIDInt)
	}

	targetActor := c.Query("target_id")
	if targetActor == "" {
		targetActor = c.Query("target_screen_name")
		if targetActor == "" {
			return c.Status(fiber.StatusBadRequest).SendString("No source_id provided")
		}
	} else {
		targetIDInt, ok := new(big.Int).SetString(targetActor, 10)
		if !ok {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid source_id provided")
		}
		targetActor = bridge.TwitterIDToBlueSky(*targetIDInt)
	}

	// auth
	_, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// It looks like there's a bug where I can't pass handles into GetRelationships, but we need to get the handle anyways, so this shouldn't impact that much

	targetUser, err := blueskyapi.GetUserInfo(*oauthToken, targetActor)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Failed to fetch target user")
	}
	// Possible optimization: if the source user is us, we can skip the api call, and just use viewer info
	sourceUser, err := blueskyapi.GetUserInfo(*oauthToken, sourceActor)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Failed to fetch source user")
	}

	fmt.Println("ID " + sourceUser.ID.String())
	targetDID := bridge.TwitterIDToBlueSky(targetUser.ID) // not the most efficient way to do this, but it works
	fmt.Println("ID " + sourceUser.ID.String())
	sourceDID := bridge.TwitterIDToBlueSky(sourceUser.ID)
	fmt.Println("ID " + sourceUser.ID.String())

	relationship, err := blueskyapi.GetRelationships(*oauthToken, sourceDID, []string{targetDID})
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch relationship")
	}
	defaultTrue := true // holy fuck i hate this

	friendship := bridge.SourceTargetFriendship{
		Target: bridge.UserFriendship{
			ID:         targetUser.ID,
			IDStr:      targetUser.ID.String(),
			ScreenName: targetUser.ScreenName,
			Following:  relationship.Relationships[0].Following != "",
			FollowedBy: relationship.Relationships[0].FollowedBy != "",
		},
		Source: bridge.UserFriendship{
			ID:         sourceUser.ID,
			IDStr:      sourceUser.ID.String(),
			ScreenName: sourceUser.ScreenName,
			Following:  relationship.Relationships[0].FollowedBy != "",
			FollowedBy: relationship.Relationships[0].Following != "",
			CanDM:      &defaultTrue,
		},
	}

	xml, err := bridge.XMLEncoder(friendship, "SourceTargetFriendship", "relationship")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to encode XML")
	}

	return c.SendString(*xml)
}
