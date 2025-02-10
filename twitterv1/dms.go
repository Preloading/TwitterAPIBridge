package twitterv1

import (
	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/gofiber/fiber/v2"
)

// TODO: Figure out pagination & limits.
func GetAllDMs(c *fiber.Ctx) error {
	_, _, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	bsky_dms, err := blueskyapi.GetDMLogs(*oauthToken, 20, "2222222222222")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get DMs")
	}

	twitter_dms := []bridge.DirectMessage{}

	for i := range bsky_dms.Logs {
		if bsky_dms.Logs[i].Type == "chat.bsky.convo.defs#logCreateMessage" {
			messageId, err := bridge.TidToNum(bsky_dms.Logs[i].Message.Id)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to convert TID to number")
			}

			twitter_dms = append(twitter_dms, bridge.DirectMessage{
				Id:        int64(messageId),
				Text:      bsky_dms.Logs[i].Message.Text,
				CreatedAt: bridge.TwitterTimeConverter(bsky_dms.Logs[i].Message.SentAt),
			})
		}
	}

	return EncodeAndSend(c, twitter_dms)
}

func GetSentDMs(c *fiber.Ctx) error {
	return c.SendString("GetSentDMs")
}
