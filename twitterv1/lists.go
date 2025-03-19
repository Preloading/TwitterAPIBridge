package twitterv1

import (
	"strconv"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/gofiber/fiber/v2"
)

// https://web.archive.org/web/20120807220901/https://dev.twitter.com/docs/api/1/get/lists
func GetUsersLists(c *fiber.Ctx) error {

	screen_name := c.Params("user")
	if screen_name == "" {
		screen_name = c.Query("screen_name")
	}

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
			Slug:            listRKEY,
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
		Lists: twitterLists,
		Cursors: bridge.Cursors{
			NextCursor:        nextCursor,
			NextCursorStr:     strconv.FormatUint(nextCursor, 10),
			PreviousCursor:    -1,
			PreviousCursorStr: "0", // Previous can equal the top element in the list, provided that this isn't the beginning of the list, or smth like that.
		},
	})
}

// https://web.archive.org/web/20120807221920/https://dev.twitter.com/docs/api/1/get/lists/statuses
func list_timeline(c *fiber.Ctx) error {
	list := c.Params("slug")

	if list == "" {
		list = c.Query("slug")
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
			owner := c.Query("owner_screen_name")
			if owner == "" {
				owner = c.Query("owner_id")
				if owner == "" {
					return c.Status(fiber.StatusBadRequest).SendString("No Owner Provided")
				}
				ownerIdInt, err := strconv.ParseInt(list, 10, 64)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).SendString("Invalid owner id provided")
				}
				ownerPtr, err := bridge.TwitterIDToBlueSky(&ownerIdInt)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).SendString("Invalid owner id provided")
				}
				owner = *ownerPtr
			} else {
				ownerDID, err := blueskyapi.ResolveDIDFromHandle(owner)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).SendString("Invalid owner handle provided")
				}
				owner = *ownerDID
			}

			list = "at://" + owner + "/app.bsky.graph.list/" + list
		}
	} else {
		listIdInt, err := strconv.ParseInt(list, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid list id provided")
		}
		listPtr, err := bridge.TwitterIDToBlueSky(&listIdInt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid list id provided")
		}
		list = *listPtr
	}

	return convert_timeline(c, list, blueskyapi.GetListTimeline)
}

func GetListMembers(c *fiber.Ctx) error {
	list := c.Params("list")

	if list == "" {
		list = c.Query("slug")
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
			owner := c.Query("owner_screen_name")
			if owner == "" {
				owner = c.Query("owner_id")
				if owner == "" {
					return c.Status(fiber.StatusBadRequest).SendString("No Owner Provided")
				}
				ownerIdInt, err := strconv.ParseInt(list, 10, 64)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).SendString("Invalid owner id provided")
				}
				ownerPtr, err := bridge.TwitterIDToBlueSky(&ownerIdInt)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).SendString("Invalid owner id provided")
				}
				owner = *ownerPtr
			} else {
				ownerDID, err := blueskyapi.ResolveDIDFromHandle(owner)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).SendString("Invalid owner handle provided")
				}
				owner = *ownerDID
			}

			list = "at://" + owner + "/app.bsky.graph.list/" + list
		}
	} else {
		listIdInt, err := strconv.ParseInt(list, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid list id provided")
		}
		listPtr, err := bridge.TwitterIDToBlueSky(&listIdInt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid list id provided")
		}
		list = *listPtr
	}

	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	// Cursor
	cursor := c.Query("cursor")
	if cursor == "-1" {
		cursor = ""
	}

	// Get our list
	listInfo, err := blueskyapi.GetList(*pds, *oauthToken, list, 20, cursor) // No clue what the limit was on actual twitter.
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get list")
	}

	// Get the full user info on the members of the list.
	membersDID := []string{}

	for _, member := range listInfo.Items {
		membersDID = append(membersDID, member.Subject.DID)
	}

	members, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, membersDID, false)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get list member's info")
	}

	// Next Cursor
	nextCursor, err := bridge.TidToNum(listInfo.Cursor)
	if err != nil {
		nextCursor = 0
	}

	return EncodeAndSend(c, bridge.TwitterListMembers{
		Users: members,
		Cursors: bridge.Cursors{
			NextCursor:        nextCursor,
			NextCursorStr:     strconv.FormatUint(nextCursor, 10),
			PreviousCursor:    -1,
			PreviousCursorStr: "-1", // Previous can equal the top element in the list, provided that this isn't the beginning of the list, or smth like that.
		},
	})
}
