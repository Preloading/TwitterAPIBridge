package twitterv1

import (
	"fmt"
	"strconv"
	"strings"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/gofiber/fiber/v2"
)

// https://web.archive.org/web/20120508075505/https://dev.twitter.com/docs/api/1/get/users/show
func user_info(c *fiber.Ctx) error {
	screen_name := c.Query("screen_name")

	if screen_name == "" {
		userIDStr := c.Query("user_id")
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		screen_namePtr, err := bridge.TwitterIDToBlueSky(&userID) // yup
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		if screen_namePtr == nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		screen_name = *screen_namePtr
		if screen_name == "" {
			return c.Status(fiber.StatusBadRequest).SendString("No screen_name or user_id provided")

		}

	}

	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	userinfo, err := blueskyapi.GetUserInfo(*pds, *oauthToken, screen_name, false)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	return EncodeAndSend(c, userinfo)
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
			userID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
			}
			handle, err := bridge.TwitterIDToBlueSky(&userID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
			}
			if *handle != "" {
				usersToLookUp = append(usersToLookUp, *handle)
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
	return EncodeAndSend(c, users)
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
			actorID, err := strconv.ParseInt(actor, 10, 64)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
			}
			handlePtr, err := bridge.TwitterIDToBlueSky(&actorID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
			}
			if handlePtr != nil {
				actorsArray[i] = *handlePtr
			}
		}
	}

	relationships := []bridge.UsersRelationship{}
	users, err := blueskyapi.GetUsersInfoRaw(*pds, *oauthToken, actorsArray, false)
	for _, user := range users {
		encodedUserId := bridge.BlueSkyToTwitterID(user.DID)
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
			ID:          *encodedUserId,
			IDStr:       strconv.FormatInt(*encodedUserId, 10),
			Connections: connections,
		})
	}

	root := bridge.UserRelationships{
		Relationships: relationships,
	}

	return EncodeAndSend(c, root)
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
		sourceIDInt, err := strconv.ParseInt(sourceActor, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid source_id provided")
		}
		sourceActorPtr, err := bridge.TwitterIDToBlueSky(&sourceIDInt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert source_id to screen_name")
		}
		if sourceActorPtr == nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert source_id to screen_name")
		}
		sourceActor = *sourceActorPtr
	}

	targetActor := c.Query("target_id")
	if targetActor == "" {
		targetActor = c.Query("target_screen_name")
		if targetActor == "" {
			return c.Status(fiber.StatusBadRequest).SendString("No source_id provided")
		}
	} else {
		targetIDInt, err := strconv.ParseInt(targetActor, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid source_id provided")
		}
		targetActorPtr, err := bridge.TwitterIDToBlueSky(&targetIDInt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert source_id to screen_name")
		}
		if targetActorPtr == nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert source_id to screen_name")
		}
		targetActor = *targetActorPtr
	}

	// auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		blankstring := "" // I. Hate. This.
		oauthToken = &blankstring
	}

	// It looks like there's a bug where I can't pass handles into GetRelationships, but we need to get the handle anyways, so this shouldn't impact that much

	targetUser, err := blueskyapi.GetUserInfo(*pds, *oauthToken, targetActor, false)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Failed to fetch target user")
	}
	// Possible optimization: if the source user is us, we can skip the api call, and just use viewer info
	sourceUser, err := blueskyapi.GetUserInfo(*pds, *oauthToken, sourceActor, false)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Failed to fetch source user")
	}

	targetDID, err := bridge.TwitterIDToBlueSky(&targetUser.ID) // not the most efficient way to do this, but it works
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to convert target user ID to BlueSky ID")
	}
	sourceDID, err := bridge.TwitterIDToBlueSky(&sourceUser.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to convert source user ID to BlueSky ID")
	}

	relationship, err := blueskyapi.GetRelationships(*pds, *oauthToken, *sourceDID, []string{*targetDID})
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch relationship")
	}
	defaultTrue := true // holy fuck i hate this

	friendship := bridge.SourceTargetFriendship{
		Target: bridge.UserFriendship{
			ID:         targetUser.ID,
			IDStr:      strconv.FormatInt(targetUser.ID, 10),
			ScreenName: targetUser.ScreenName,
			Following:  relationship.Relationships[0].FollowedBy != "",
			FollowedBy: relationship.Relationships[0].Following != "",
		},
		Source: bridge.UserFriendship{
			ID:         sourceUser.ID,
			IDStr:      strconv.FormatInt(sourceUser.ID, 10),
			ScreenName: sourceUser.ScreenName,
			Following:  relationship.Relationships[0].Following != "",
			FollowedBy: relationship.Relationships[0].FollowedBy != "",
			CanDM:      &defaultTrue,
		},
	}

	root := bridge.SourceTargetFriendshipRoot{
		Relation: friendship,
	}

	return EncodeAndSend(c, root)
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
		id, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		if actorPtr == nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		actor = *actorPtr
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

	return EncodeAndSend(c, twitterUser)
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

	return EncodeAndSend(c, twitterUser)
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
		id, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		if actorPtr == nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		actor = *actorPtr
	}
	return UnfollowUser(c, actor)
}

func UnfollowUserParams(c *fiber.Ctx) error {
	// This should allow lookup with a handle, but tbh, i'm too lazy to implement that right now as i do not see it being used.
	actor := c.Params("id")
	actorID, err := strconv.ParseInt(actor, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
	}
	actorPtr, err := bridge.TwitterIDToBlueSky(&actorID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
	}
	if actorPtr == nil {
		return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
	}
	actor = *actorPtr

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
		id, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		if actorPtr == nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		actor = *actorPtr
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

	return EncodeAndSend(c, bridge.TwitterUsers{
		Users: twitterUsersConverted,
	})
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
		id, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		if actorPtr == nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		actor = *actorPtr
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

	return EncodeAndSend(c, bridge.TwitterUsers{
		Users: twitterUsersConverted,
	})
}

func GetSuggestedUsers(c *fiber.Ctx) error {
	var err error
	// limits
	limit := 30
	if c.Query("limit") != "" {
		limit, err = strconv.Atoi(c.Query("limit"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid limit value")
		}
	}

	// auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	var recommendedUsers []blueskyapi.User

	// see if they provided a user id
	userID := c.Query("user_id")
	if userID != "" {
		userIDInt, err := strconv.ParseInt(userID, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		userIDPtr, err := bridge.TwitterIDToBlueSky(&userIDInt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		if userIDPtr == nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to convert user_id to screen_name")
		}
		userID = *userIDPtr

		recommendedUsers, err = blueskyapi.GetOthersSuggestedUsers(*pds, *oauthToken, limit, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch suggested users")
		}
	} else {
		recommendedUsers, err = blueskyapi.GetMySuggestedUsers(*pds, *oauthToken, limit)
	}

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch suggested users")
	}

	usersDID := []string{}
	for _, user := range recommendedUsers {
		usersDID = append(usersDID, user.DID)
	}

	usersInfo, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, usersDID, false)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	recommended := []bridge.TwitterRecommendation{}
	for _, user := range usersInfo {
		recommended = append(recommended, bridge.TwitterRecommendation{
			UserID: user.ID,
			User:   *user,
		})
	}

	return EncodeAndSend(c, recommended)
}
