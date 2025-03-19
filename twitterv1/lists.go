package twitterv1

import (
	"strconv"
	"strings"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/gofiber/fiber/v2"
)

// https://web.archive.org/web/20120807220901/https://dev.twitter.com/docs/api/1/get/lists
func GetUsersLists(c *fiber.Ctx) error {
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
	}

	userDID, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	if screen_name == "" {
		if userDID == nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid user_id provided")
		}
		screen_name = *userDID
	}

	cursor := c.Query("cursor")

	lists, err := blueskyapi.GetUsersLists(*pds, *oauthToken, screen_name, 20, cursor)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get lists")
	}

	listsOwner, err := blueskyapi.GetUserInfo(*pds, *oauthToken, screen_name, false)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
	}

	twitterLists := []bridge.TwitterList{}

	for _, list := range lists.Lists {
		listDID, _, listRKEY := blueskyapi.GetURIComponents(list.URI)
		id := bridge.BlueSkyToTwitterID(list.URI)

		twitterLists = append(twitterLists, bridge.TwitterList{
			Slug:            listDID + "/" + listRKEY,
			Name:            list.Name,
			URI:             listDID + "/" + listRKEY,
			FullName:        list.Name,
			Description:     list.Description,
			ID:              *id,
			IDStr:           strconv.FormatInt(*id, 10),
			Following:       false, // You cannot subscibe to lists, and following is... fucky
			MemberCount:     list.ListItemCount,
			SubscriberCount: 0, // You can't subscribe to lists
			Mode:            "public",
			User:            *listsOwner,
		})
	}

	// Next Cursor
	nextCursor, err := bridge.TidToNum(lists.Cursor)
	if err != nil {
		nextCursor = 0
	}

	return EncodeAndSend(c, bridge.TwitterLists{
		Lists:             twitterLists,
		NextCursor:        nextCursor,
		NextCursorStr:     strconv.FormatUint(nextCursor, 10),
		PreviousCursor:    -1,
		PreviousCursorStr: "0", // Previous can equal the top element in the list, provided that this isn't the beginning of the list, or smth like that.
	})
}

// https://web.archive.org/web/20120807221920/https://dev.twitter.com/docs/api/1/get/lists/statuses
func list_timeline(c *fiber.Ctx) error {
	list := c.Query("slug")
	if list == "" {
		list = c.Query("list_id")
		if list == "" {
			return c.Status(fiber.StatusBadRequest).SendString("No List Provided")
		}
		listIdInt, err := strconv.ParseInt(list, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid list id provided")
		}
		listPtr, err := bridge.TwitterIDToBlueSky(&listIdInt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid list id provided")
		}
		list = *listPtr
	} else {
		listParts := strings.Split(list, "/")
		if len(listParts) != 2 {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid list slug provided")
		}
		list = "at://" + listParts[0] + "/app.bsky.graph.list/" + listParts[1]
	}
	return convert_timeline(c, list, blueskyapi.GetListTimeline)
}
