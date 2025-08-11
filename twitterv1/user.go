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
	actor := c.Query("screen_name")
	if ac := c.Locals("handle"); ac != nil {
		actor = ac.(string)
	}

	if actor == "" {
		userIDStr := c.Query("user_id")
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid ID format", 195, 403)
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&userID) // yup
		if err != nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		if actorPtr == nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		actor = *actorPtr
		if actor == "" {
			return ReturnError(c, "No user was specified", 195, 403)

		}

	}

	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	userinfo, err := blueskyapi.GetUserInfo(*pds, *oauthToken, actor, false)

	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfile", user_info)
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
				return ReturnError(c, "Invalid ID format ("+idStr+")", 195, 403)
			}
			actor, err := bridge.TwitterIDToBlueSky(&userID)
			if err != nil {
				return ReturnError(c, "ID not found. ("+strconv.FormatInt(userID, 10)+")", 144, fiber.StatusNotFound)
			}
			if *actor != "" {
				usersToLookUp = append(usersToLookUp, *actor)
			}
		}
	} else {
		return ReturnError(c, "No user was specified", 195, 403)
	}

	if len(usersToLookUp) > 100 {
		return ReturnError(c, "Max number of times to look up reached (100)", 195, 403)
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
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfiles", UsersLookup)
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
			return ReturnError(c, "No user was specified", 195, 403)
		}
	}

	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return MissingAuth(c, err)
	}

	actorsArray := strings.Split(actors, ",")
	if len(actorsArray) > 100 {
		return ReturnError(c, "Max number of times to look up reached (100)", 195, 403)
	}
	if isID {
		for i, actor := range actorsArray {
			actorID, err := strconv.ParseInt(actor, 10, 64)
			if err != nil {
				return ReturnError(c, "Invalid ID format ("+actor+")", 195, 403)
			}
			actorPtr, err := bridge.TwitterIDToBlueSky(&actorID)
			if err != nil {
				return ReturnError(c, "ID not found. ("+strconv.FormatInt(actorID, 10)+")", 144, fiber.StatusNotFound)
			}
			if actorPtr != nil {
				actorsArray[i] = *actorPtr
			}
		}
	}

	relationships := []bridge.UsersRelationship{}
	users, err := blueskyapi.GetUsersInfoRaw(*pds, *oauthToken, actorsArray, false)
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfiles", UserRelationships)
	}
	for _, user := range users {
		encodedUserId := bridge.BlueSkyToTwitterID(user.DID)

		connections := bridge.Connections{}
		connectionJSON := []string{}

		if user.Viewer.Following != nil {
			connections.Connection = append(connections.Connection, bridge.Connection{Value: "following"})
			connectionJSON = append(connectionJSON, "following")
		}
		if user.Viewer.FollowedBy != nil {
			connections.Connection = append(connections.Connection, bridge.Connection{Value: "followed_by"})
			connectionJSON = append(connectionJSON, "followed_by")
		}
		if user.Viewer.Blocking != nil {
			connections.Connection = append(connections.Connection, bridge.Connection{Value: "blocked"})
			connectionJSON = append(connectionJSON, "blocking")
		}
		if user.Viewer.BlockedBy {
			connections.Connection = append(connections.Connection, bridge.Connection{Value: "blocked_by"}) // Complete guess
			connectionJSON = append(connectionJSON, "blocked_by")
		}

		relationships = append(relationships, bridge.UsersRelationship{
			Name: func() string {
				if user.DisplayName == "" {
					return user.Handle
				}
				return user.DisplayName
			}(),
			ScreenName:     user.Handle,
			ID:             *encodedUserId,
			IDStr:          strconv.FormatInt(*encodedUserId, 10),
			ConnectionsXML: connections,
			Connections:    connectionJSON,
		})
	}

	if c.Params("filetype") == "xml" {
		root := bridge.UserRelationships{
			Relationships: relationships,
		}
		return EncodeAndSend(c, root)
	} else {
		return EncodeAndSend(c, relationships)
	}

}

// Gets the relationship between two users
// https://web.archive.org/web/20120516154953/https://dev.twitter.com/docs/api/1/get/friendships/show
func GetUsersRelationship(c *fiber.Ctx) error {
	// Get the actors
	sourceActor := c.Query("source_id")
	if sourceActor == "" {
		sourceActor = c.Query("source_screen_name")
		if sourceActor == "" {
			return ReturnError(c, "No source user was specified", 195, 403)
		}
	} else {
		sourceIDInt, err := strconv.ParseInt(sourceActor, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid source ID format", 195, 403)
		}
		sourceActorPtr, err := bridge.TwitterIDToBlueSky(&sourceIDInt)
		if err != nil {
			return ReturnError(c, "Source ID not found.", 144, fiber.StatusNotFound)
		}
		if sourceActorPtr == nil {
			return ReturnError(c, "Source ID not found.", 144, fiber.StatusNotFound)
		}
		sourceActor = *sourceActorPtr
	}

	targetActor := c.Query("target_id")
	if targetActor == "" {
		targetActor = c.Query("target_screen_name")
		if targetActor == "" {
			return ReturnError(c, "No target user was specified", 195, 403)
		}
	} else {
		targetIDInt, err := strconv.ParseInt(targetActor, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid target ID format", 195, 403)
		}
		targetActorPtr, err := bridge.TwitterIDToBlueSky(&targetIDInt)
		if err != nil {
			return ReturnError(c, "Target ID not found.", 144, fiber.StatusNotFound)
		}
		if targetActorPtr == nil {
			return ReturnError(c, "Target ID not found.", 144, fiber.StatusNotFound)
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
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfile", GetUsersRelationship)
	}
	// Possible optimization: if the source user is us, we can skip the api call, and just use viewer info
	sourceUser, err := blueskyapi.GetUserInfo(*pds, *oauthToken, sourceActor, false)
	if err != nil {
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfile", GetUsersRelationship)
	}

	targetDID, err := bridge.TwitterIDToBlueSky(&targetUser.ID) // not the most efficient way to do this, but it works
	if err != nil {
		return ReturnError(c, "Some wonky stuff happened. (Failed to convert target user ID to BlueSky ID)", 131, 500)
	}
	sourceDID, err := bridge.TwitterIDToBlueSky(&sourceUser.ID)
	if err != nil {
		return ReturnError(c, "Some wonky stuff happened. (Failed to convert target user ID to BlueSky ID)", 131, 500)
	}

	relationship, err := blueskyapi.GetRelationships(*pds, *oauthToken, *sourceDID, []string{*targetDID})
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.graph.getRelationships", GetUsersRelationship)
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
		return MissingAuth(c, err)
	}

	// lets get the user params
	actor := c.FormValue("user_id")
	if actor == "" {
		actor = c.FormValue("screen_name")
		if actor == "" {
			return ReturnError(c, "No user was specified", 195, 403)
		}
	} else {
		id, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid ID format", 195, 403)
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&id)
		if err != nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		if actorPtr == nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		actor = *actorPtr
	}

	// follow
	user, err := blueskyapi.FollowUser(*pds, *oauthToken, actor, *my_did)

	if err != nil {
		return HandleBlueskyError(c, err.Error(), "app.bsky.graph.follow", FollowUser) // lexicon isnt tecnically right, but its fine idc
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
		return MissingAuth(c, err)
	}

	// follow
	user, err := blueskyapi.UnfollowUser(*pds, *oauthToken, actor, *my_did)

	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.graph.unfollow", func(c *fiber.Ctx) error {
			return UnfollowUser(c, actor)
		})
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
			return ReturnError(c, "No user was specified", 195, 403)
		}
	} else {
		id, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid ID format", 195, 403)
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&id)
		if err != nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		if actorPtr == nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
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
		return ReturnError(c, "Invalid ID format", 195, 403)
	}
	actorPtr, err := bridge.TwitterIDToBlueSky(&actorID)
	if err != nil {
		return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
	}
	if actorPtr == nil {
		return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
	}
	actor = *actorPtr

	return UnfollowUser(c, actor)
}

// https://web.archive.org/web/20101115102530/http://apiwiki.twitter.com/w/page/22554748/Twitter-REST-API-Method%3a-statuses%C2%A0followers
// At the moment we are not doing pagination, so this will only return the first ~50 followers.
func GetStatusesFollowers(c *fiber.Ctx) error {
	// auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c, err)
	}

	// lets go get our user data

	actor := c.FormValue("user_id")
	if actor == "" {
		actor = c.FormValue("screen_name")
		if actor == "" {
			return ReturnError(c, "No user was specified", 195, 403)
		}
	} else {
		id, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid ID format", 195, 403)
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&id)
		if err != nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		if actorPtr == nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		actor = *actorPtr
	}

	// fetch followers
	followers, err := blueskyapi.GetFollowers(*pds, *oauthToken, "", actor)
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.graph.getFollowers", GetStatusesFollowers)
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
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfiles", GetStatusesFollowers)
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

func GetFollowers(c *fiber.Ctx) error {
	// auth
	userDID, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c, err)
	}

	// lets go get our user data
	actor := c.FormValue("user_id")
	if actor == "" {
		actor = c.FormValue("screen_name")
		if actor == "" {
			if userDID != nil {
				actor = *userDID
			} else {
				return ReturnError(c, "No user was specified", 195, 403)
			}
		}
	} else {
		id, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid ID format", 195, 403)
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&id)
		if err != nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		if actorPtr == nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		actor = *actorPtr
	}

	cursor := ""
	var cursorInt int64

	cursorStr := c.FormValue("cursor")
	if cursorStr != "" {
		cursorInt, err = strconv.ParseInt(cursorStr, 10, 64)
		if err != nil || cursorInt > 1 {
			cursor, err = bridge.NumToTid(uint64(cursorInt))
			if err != nil {
				fmt.Println("Error when converting Followers Cursor:", err)
				cursor = ""
			}
		} else {
			cursor = ""
		}
	} else {
		cursor = ""
	}

	if cursorInt == 0 {
		return EncodeAndSend(c, struct {
			Users             []bridge.TwitterUser `json:"users" xml:"users"`
			NextCursor        uint64               `json:"next_cursor" xml:"next_cursor"`
			PreviousCursor    uint64               `json:"previous_cursor" xml:"previous_cursor"`
			NextCursorStr     string               `json:"next_cursor_str" xml:"-"`
			PreviousCursorStr string               `json:"previous_cursor_str" xml:"-"`
		}{
			Users:             []bridge.TwitterUser{},
			NextCursor:        0,
			PreviousCursor:    0, // Unimplemented. This could probably be figured out if i could figure out what the TID corrisponds to, if it corrisponds to anything at all.
			NextCursorStr:     "0",
			PreviousCursorStr: "0",
		})
	}

	// fetch followers
	followers, err := blueskyapi.GetFollowers(*pds, *oauthToken, cursor, actor)
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.graph.getFollowers", GetFollowers)
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
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfiles", GetFollowers)
	}

	// Convert []*bridge.TwitterUser to []bridge.TwitterUser
	var twitterUsersConverted []bridge.TwitterUser
	for _, user := range twitterUsers {
		twitterUsersConverted = append(twitterUsersConverted, *user)
	}

	next_cursor, err := bridge.TidToNum(followers.Cursor)
	if err != nil {
		next_cursor = 0
	}

	return EncodeAndSend(c, struct {
		Users             []bridge.TwitterUser `json:"users" xml:"users"`
		NextCursor        uint64               `json:"next_cursor" xml:"next_cursor"`
		PreviousCursor    uint64               `json:"previous_cursor" xml:"previous_cursor"`
		NextCursorStr     string               `json:"next_cursor_str" xml:"-"`
		PreviousCursorStr string               `json:"previous_cursor_str" xml:"-"`
	}{
		Users:             twitterUsersConverted,
		NextCursor:        next_cursor,
		PreviousCursor:    0, // Unimplemented. This could probably be figured out if i could figure out what the TID corrisponds to, if it corrisponds to anything at all.
		NextCursorStr:     strconv.FormatUint(next_cursor, 10),
		PreviousCursorStr: "0",
	})
}

// https://web.archive.org/web/20120407214017/https://dev.twitter.com/docs/api/1/get/statuses/friends
func GetStatusesFollows(c *fiber.Ctx) error {
	// auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c, err)
	}

	// lets go get our user data

	actor := c.FormValue("user_id")
	if actor == "" {
		actor = c.FormValue("screen_name")
		if actor == "" {
			return ReturnError(c, "No user was specified", 195, 403)
		}
	} else {
		id, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid ID format", 195, 403)
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&id)
		if err != nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		if actorPtr == nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		actor = *actorPtr
	}

	// fetch follows
	followers, err := blueskyapi.GetFollows(*pds, *oauthToken, "", actor)
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.graph.getFollows", GetStatusesFollows)
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
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfiles", GetStatusesFollows)
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

func GetFollows(c *fiber.Ctx) error {
	// auth
	userDID, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c, err)
	}

	// lets go get our user data
	actor := c.FormValue("user_id")
	if actor == "" {
		actor = c.FormValue("screen_name")
		if actor == "" {
			if userDID != nil {
				actor = *userDID
			} else {
				return ReturnError(c, "No user was specified", 195, 403)
			}
		}
	} else {
		id, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid ID format", 195, 403)
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&id)
		if err != nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		if actorPtr == nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		actor = *actorPtr
	}

	cursor := ""
	var cursorInt int64

	cursorStr := c.FormValue("cursor")
	if cursorStr != "" {
		cursorInt, err = strconv.ParseInt(cursorStr, 10, 64)
		if err != nil || cursorInt > 1 {
			cursor, err = bridge.NumToTid(uint64(cursorInt))
			if err != nil {
				fmt.Println("Error when converting Followers Cursor:", err)
				cursor = ""
			}
		} else {
			cursor = ""
		}
	} else {
		cursor = ""
	}

	if cursorInt == 0 {
		return EncodeAndSend(c, struct {
			Users []bridge.TwitterUser `json:"users" xml:"users"`
			bridge.Cursors
		}{
			Users: []bridge.TwitterUser{},
			Cursors: bridge.Cursors{
				NextCursor:        0,
				PreviousCursor:    0, // Unimplemented. This could probably be figured out if i could figure out what the TID corrisponds to, if it corrisponds to anything at all.
				NextCursorStr:     "0",
				PreviousCursorStr: "0",
			},
		})
	}

	// fetch follows
	followers, err := blueskyapi.GetFollows(*pds, *oauthToken, cursor, actor)
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.graph.getFollows", GetFollows)
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
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfiles", GetFollows)
	}

	// Convert []*bridge.TwitterUser to []bridge.TwitterUser
	var twitterUsersConverted []bridge.TwitterUser
	for _, user := range twitterUsers {
		twitterUsersConverted = append(twitterUsersConverted, *user)
	}

	next_cursor, err := bridge.TidToNum(followers.Cursor)
	if err != nil {
		next_cursor = 0
	}

	return EncodeAndSend(c, struct {
		Users []bridge.TwitterUser `json:"users" xml:"users"`
		bridge.Cursors
	}{
		Users: twitterUsersConverted,
		Cursors: bridge.Cursors{
			NextCursor:        next_cursor,
			PreviousCursor:    0, // Unimplemented. This could probably be figured out if i could figure out what the TID corrisponds to, if it corrisponds to anything at all.
			NextCursorStr:     strconv.FormatUint(next_cursor, 10),
			PreviousCursorStr: "0",
		},
	})
}

func GetFollowingIds(c *fiber.Ctx) error {
	// auth
	userDID, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c, err)
	}

	// lets go get our user data
	actor := c.FormValue("user_id")
	if actor == "" {
		actor = c.FormValue("screen_name")
		if actor == "" {
			if userDID != nil {
				actor = *userDID
			} else {
				return ReturnError(c, "No user was specified", 195, 403)
			}
		}
	} else {
		id, err := strconv.ParseInt(actor, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid ID format", 195, 403)
		}
		actorPtr, err := bridge.TwitterIDToBlueSky(&id)
		if err != nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		if actorPtr == nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		actor = *actorPtr
	}

	cursor := ""
	var cursorInt int64

	cursorStr := c.FormValue("cursor")
	if cursorStr != "" {
		cursorInt, err = strconv.ParseInt(cursorStr, 10, 64)
		if err != nil || cursorInt > 1 {
			cursor, err = bridge.NumToTid(uint64(cursorInt))
			if err != nil {
				fmt.Println("Error when converting Followers Cursor:", err)
				cursor = ""
			}
		} else {
			cursor = ""
		}
	} else {
		cursor = ""
	}

	if cursorInt == 0 {
		return EncodeAndSend(c, struct {
			Users []bridge.TwitterUser `json:"users" xml:"users"`
			bridge.Cursors
		}{
			Users: []bridge.TwitterUser{},
			Cursors: bridge.Cursors{
				NextCursor:        0,
				PreviousCursor:    0, // Unimplemented. This could probably be figured out if i could figure out what the TID corrisponds to, if it corrisponds to anything at all.
				NextCursorStr:     "0",
				PreviousCursorStr: "0",
			},
		})
	}

	// fetch follows
	followers, err := blueskyapi.GetFollows(*pds, *oauthToken, cursor, actor)
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.graph.getFollows", GetFollows)
	}

	var userIDs []int64
	for _, user := range followers.Followers {
		userIDs = append(userIDs, *bridge.BlueSkyToTwitterID(user.DID))
	}

	next_cursor, err := bridge.TidToNum(followers.Cursor)
	if err != nil {
		next_cursor = 0
	}

	return EncodeAndSend(c, bridge.IdsWithCursor{
		Ids: userIDs,
		Cursors: bridge.Cursors{
			NextCursor:        next_cursor,
			PreviousCursor:    0, // Unimplemented. This could probably be figured out if i could figure out what the TID corrisponds to, if it corrisponds to anything at all.
			NextCursorStr:     strconv.FormatUint(next_cursor, 10),
			PreviousCursorStr: "0",
		},
	})
}

func GetSuggestedUsers(c *fiber.Ctx) error {
	var err error
	// limits
	limit := 30
	if c.Query("limit") != "" {
		limit, err = strconv.Atoi(c.Query("limit"))
		if err != nil {
			return ReturnError(c, "Invalid limit", 195, 403)
		}
	}

	// auth
	_, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c, err)
	}

	var recommendedUsers []blueskyapi.User

	// see if they provided a user id
	userID := c.Query("user_id")
	if userID != "" {
		userIDInt, err := strconv.ParseInt(userID, 10, 64)
		if err != nil {
			return ReturnError(c, "Invalid ID format", 195, 403)
		}
		userIDPtr, err := bridge.TwitterIDToBlueSky(&userIDInt)
		if err != nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		if userIDPtr == nil {
			return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
		}
		userID = *userIDPtr

		recommendedUsers, err = blueskyapi.GetOthersSuggestedUsers(*pds, *oauthToken, limit, userID)
		if err != nil {
			return HandleBlueskyError(c, err.Error(), "app.bsky.graph.getSuggestedFollowsByActor", GetSuggestedUsers)
		}
	} else {
		recommendedUsers, err = blueskyapi.GetMySuggestedUsers(*pds, *oauthToken, limit)
		if err != nil {
			fmt.Println("Error:", err)
			return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getSuggestions", GetSuggestedUsers)
		}
	}

	usersDID := []string{}
	for _, user := range recommendedUsers {
		usersDID = append(usersDID, user.DID)
	}

	usersInfo, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, usersDID, false)
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "app.bsky.actor.getProfiles", GetSuggestedUsers)
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
