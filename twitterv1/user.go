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

	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	userinfo, err := blueskyapi.GetUserInfo(*pds, *oauthToken, screen_name)

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

	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	// here's some fun problems!
	// twitter api's max is 100 users per call. bluesky's is 25. so we get to lookup in multiple requests

	users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, usersToLookUp, false)
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

	_, pds, _, oauthToken, err := GetAuthFromReq(c)

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
	users, err := blueskyapi.GetUsersInfoRaw(*pds, *oauthToken, actorsArray, false)
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
			ID:          &encodedUserId,
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
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		blankstring := "" // I. Hate. This.
		oauthToken = &blankstring
	}

	// It looks like there's a bug where I can't pass handles into GetRelationships, but we need to get the handle anyways, so this shouldn't impact that much

	targetUser, err := blueskyapi.GetUserInfo(*pds, *oauthToken, targetActor)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Failed to fetch target user")
	}
	// Possible optimization: if the source user is us, we can skip the api call, and just use viewer info
	sourceUser, err := blueskyapi.GetUserInfo(*pds, *oauthToken, sourceActor)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Failed to fetch source user")
	}

	targetDID := bridge.TwitterIDToBlueSky(*targetUser.ID) // not the most efficient way to do this, but it works
	sourceDID := bridge.TwitterIDToBlueSky(*sourceUser.ID)

	relationship, err := blueskyapi.GetRelationships(*pds, *oauthToken, sourceDID, []string{targetDID})
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
			Following:  relationship.Relationships[0].FollowedBy != "",
			FollowedBy: relationship.Relationships[0].Following != "",
		},
		Source: bridge.UserFriendship{
			ID:         sourceUser.ID,
			IDStr:      sourceUser.ID.String(),
			ScreenName: sourceUser.ScreenName,
			Following:  relationship.Relationships[0].Following != "",
			FollowedBy: relationship.Relationships[0].FollowedBy != "",
			CanDM:      &defaultTrue,
		},
	}

	xml, err := bridge.XMLEncoder(friendship, "SourceTargetFriendship", "relationship")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to encode XML")
	}

	return c.SendString(*xml)
}

// https://web.archive.org/web/20120407201029/https://dev.twitter.com/docs/api/1/post/friendships/create
func FollowUser(c *fiber.Ctx) error {
	// auth
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// lets get the user params
	actor := c.FormValue("user_id")
	if actor == "" {
		actor = c.FormValue("screen_name")
		if actor == "" {
			c.Status(fiber.StatusBadRequest).SendString("No user provided")
		}
	} else {
		id, ok := new(big.Int).SetString(actor, 10)
		if !ok {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actor = bridge.TwitterIDToBlueSky(*id)
	}

	// follow
	err, user := blueskyapi.FollowUser(*pds, *oauthToken, actor, *my_did)

	if err != nil {
		if err.Error() == "already following user" {
			return c.Status(403).SendString("already following user")
		}
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to follow user")
	}

	// convert user into twitter format
	twitterUser := blueskyapi.AuthorTTB(*user)

	// XML Encode
	xml, err := bridge.XMLEncoder(twitterUser, "TwitterUser", "user")
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode user info")
	}

	return c.SendString(*xml)
}

// https://web.archive.org/web/20120407201029/https://dev.twitter.com/docs/api/1/post/friendships/create
func UnfollowUser(c *fiber.Ctx, actor string) error {
	// auth
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// follow
	err, user := blueskyapi.UnfollowUser(*pds, *oauthToken, actor, *my_did)

	if err != nil {
		if err.Error() == "not following user" {
			return c.Status(403).SendString("not following user")
		}
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to unfollow user")
	}

	// convert user into twitter format
	twitterUser := blueskyapi.AuthorTTB(*user)

	// XML Encode
	xml, err := bridge.XMLEncoder(twitterUser, "TwitterUser", "user")
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode user info")
	}

	return c.SendString(*xml)
}

func UnfollowUserForm(c *fiber.Ctx) error {
	// lets get the user params
	actor := c.FormValue("user_id")
	if actor == "" {
		actor = c.FormValue("screen_name")
		if actor == "" {
			c.Status(fiber.StatusBadRequest).SendString("No user provided")
		}
	} else {
		id, ok := new(big.Int).SetString(actor, 10)
		if !ok {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actor = bridge.TwitterIDToBlueSky(*id)
	}
	return UnfollowUser(c, actor)
}

func UnfollowUserParams(c *fiber.Ctx) error {
	// This should allow lookup with a handle, but tbh, i'm too lazy to implement that right now as i do not see it being used.
	actor := c.Params("id")
	actorID, ok := new(big.Int).SetString(actor, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
	}
	actor = bridge.TwitterIDToBlueSky(*actorID)

	return UnfollowUser(c, actor)
}

// https://web.archive.org/web/20101115102530/http://apiwiki.twitter.com/w/page/22554748/Twitter-REST-API-Method%3a-statuses%C2%A0followers
// At the moment we are not doing pagination, so this will only return the first ~50 followers.
func GetFollowers(c *fiber.Ctx) error {
	// auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// lets go get our user data

	actor := c.FormValue("user_id")
	if actor == "" {
		actor = c.FormValue("screen_name")
		if actor == "" {
			c.Status(fiber.StatusBadRequest).SendString("No user provided")
		}
	} else {
		id, ok := new(big.Int).SetString(actor, 10)
		if !ok {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actor = bridge.TwitterIDToBlueSky(*id)
	}

	// fetch followers
	followers, err := blueskyapi.GetFollowers(*pds, *oauthToken, "", actor)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch followers")
	}

	// convert users into twitter format
	// This right now doesn't act on pagination, i'll figure that out later
	var actorsToLookUp []string
	for _, user := range followers.Followers {
		actorsToLookUp = append(actorsToLookUp, user.DID)
	}

	twitterUsers, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, actorsToLookUp, false)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	// Convert []*bridge.TwitterUser to []bridge.TwitterUser
	var twitterUsersConverted []bridge.TwitterUser
	for _, user := range twitterUsers {
		twitterUsersConverted = append(twitterUsersConverted, *user)
	}

	// XML Encode
	xml, err := bridge.XMLEncoder(
		bridge.TwitterUsers{
			Users: twitterUsersConverted,
		}, "TwitterUsers", "users")
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode user info")
	}

	return c.SendString(*xml)
}

// https://web.archive.org/web/20120407214017/https://dev.twitter.com/docs/api/1/get/statuses/friends
func GetFollows(c *fiber.Ctx) error {
	// auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// lets go get our user data

	actor := c.FormValue("user_id")
	if actor == "" {
		actor = c.FormValue("screen_name")
		if actor == "" {
			c.Status(fiber.StatusBadRequest).SendString("No user provided")
		}
	} else {
		id, ok := new(big.Int).SetString(actor, 10)
		if !ok {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actor = bridge.TwitterIDToBlueSky(*id)
	}

	// fetch followers
	followers, err := blueskyapi.GetFollows(*pds, *oauthToken, "", actor)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch followers")
	}

	// convert users into twitter format
	// This right now doesn't act on pagination, i'll figure that out later
	var actorsToLookUp []string
	for _, user := range followers.Followers {
		actorsToLookUp = append(actorsToLookUp, user.DID)
	}

	twitterUsers, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, actorsToLookUp, false)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	// Convert []*bridge.TwitterUser to []bridge.TwitterUser
	var twitterUsersConverted []bridge.TwitterUser
	for _, user := range twitterUsers {
		twitterUsersConverted = append(twitterUsersConverted, *user)
	}

	// XML Encode
	xml, err := bridge.XMLEncoder(
		bridge.TwitterUsers{
			Users: twitterUsersConverted,
		}, "TwitterUsers", "users")
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode user info")
	}

	return c.SendString(*xml)
}
