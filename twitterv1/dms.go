package twitterv1

import (
	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/gofiber/fiber/v2"
)

// TODO: Figure out pagination & limits.
// This also needs a fuckton of optimization.
// https://web.archive.org/web/20120807211214/https://dev.twitter.com/docs/api/1/get/direct_messages
func GetRecievedDMs(c *fiber.Ctx) error {
	userDID, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	myUser, err := blueskyapi.GetUserInfo(*pds, *oauthToken, *userDID, false)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
	}

	bsky_dms, err := blueskyapi.GetDMLogs(*oauthToken, 20, "2222222222222")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get DMs")
	}

	twitter_dms := []bridge.DirectMessage{}

	for i := range bsky_dms.Logs {
		if bsky_dms.Logs[i].Type == "chat.bsky.convo.defs#logCreateMessage" && bsky_dms.Logs[i].Message.Sender.Did != *userDID {
			messageId, err := bridge.TidToNum(bsky_dms.Logs[i].Message.Id)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to convert TID to number")
			}

			senderUser, err := blueskyapi.GetUserInfo(*pds, *oauthToken, bsky_dms.Logs[i].Message.Sender.Did, false)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to get sender user info")
			}

			twitter_dms = append(twitter_dms, bridge.DirectMessage{
				Id:                  int64(messageId),
				Text:                bsky_dms.Logs[i].Message.Text,
				CreatedAt:           bridge.TwitterTimeConverter(bsky_dms.Logs[i].Message.SentAt),
				Recipient:           *myUser,
				RecipientScreenName: myUser.ScreenName,
				RecipientId:         myUser.ID,
				Sender:              *senderUser,
				SenderScreenName:    senderUser.ScreenName,
				SenderId:            senderUser.ID,
			})
		}
	}

	return EncodeAndSend(c, twitter_dms)
}

func GetSentDMs(c *fiber.Ctx) error {
	userDID, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	bsky_dms, err := blueskyapi.GetDMLogs(*oauthToken, 20, "2222222222222")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get DMs")
	}

	twitter_dms := []bridge.DirectMessage{}

	convoInfos := make(map[string]blueskyapi.ConvoInfo)

	for i := range bsky_dms.Logs {
		if bsky_dms.Logs[i].Type == "chat.bsky.convo.defs#logCreateMessage" && bsky_dms.Logs[i].Message.Sender.Did == *userDID {
			messageId, err := bridge.TidToNum(bsky_dms.Logs[i].Message.Id)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to convert TID to number")
			}

			var recipientUser *bridge.TwitterUser

			// We get to go figure out who the recipient is.
			var convoInfo *blueskyapi.ConvoInfo
			if fetchedInfo, ok := convoInfos[bsky_dms.Logs[i].ConvoId]; ok {
				convoInfo = &fetchedInfo
			} else {
				convoInfo, err = blueskyapi.GetConvoInfo(*oauthToken, bsky_dms.Logs[i].ConvoId)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).SendString("Failed to get convo info")
				}
				convoInfos[bsky_dms.Logs[i].ConvoId] = *convoInfo
			}

			// Get the member that isn't ours.
			for _, member := range convoInfo.Members {
				if member.DID != *userDID {
					recipientUser, err = blueskyapi.GetUserInfo(*pds, *oauthToken, member.DID, false)
					if err != nil {
						return c.Status(fiber.StatusInternalServerError).SendString("Failed to get recipient user info")
					}
					break
				}
			}

			senderUser, err := blueskyapi.GetUserInfo(*pds, *oauthToken, bsky_dms.Logs[i].Message.Sender.Did, false)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to get sender user info")
			}

			twitter_dms = append(twitter_dms, bridge.DirectMessage{
				Id:                  int64(messageId),
				Text:                bsky_dms.Logs[i].Message.Text,
				CreatedAt:           bridge.TwitterTimeConverter(bsky_dms.Logs[i].Message.SentAt),
				Recipient:           *recipientUser,
				RecipientScreenName: recipientUser.ScreenName,
				RecipientId:         recipientUser.ID,
				Sender:              *senderUser,
				SenderScreenName:    senderUser.ScreenName,
				SenderId:            senderUser.ID,
			})
		}
	}

	return EncodeAndSend(c, twitter_dms)
}
