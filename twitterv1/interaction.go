package twitterv1

import (
	"fmt"
	"math/big"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
)

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/post/statuses/update
func status_update(c *fiber.Ctx) error {
	_, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	status := c.FormValue("status")
	trim_user := c.FormValue("trim_user")
	in_reply_to_status_id := c.FormValue("in_reply_to_status_ids")

	fmt.Println("Status:", status)
	fmt.Println("TrimUser:", trim_user)
	fmt.Println("InReplyToStatusID:", in_reply_to_status_id)

	if err := blueskyapi.UpdateStatus(*oauthToken, status); err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update status")
	}

	// TODO: Implement this

	return c.SendString("Not implemented")
}

// https://web.archive.org/web/20120407091252/https://dev.twitter.com/docs/api/1/post/statuses/retweet/%3Aid
func retweet(c *fiber.Ctx) error {
	postId := c.Params("id")
	user_did, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	idBigInt, ok := new(big.Int).SetString(postId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postId, _ = bridge.TwitterMsgIdToBluesky(idBigInt)

	err, originalPost, retweetPostURI := blueskyapi.ReTweet(*oauthToken, postId, *user_did)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update status")
	}

	retweet := TranslatePostToTweet(originalPost.Thread.Post, originalPost.Thread.Post.URI, originalPost.Thread.Parent.Author.DID, nil)
	retweet.Retweeted = true
	retweet.ID = *bridge.BlueSkyToTwitterID(*retweetPostURI)
	retweet.IDStr = bridge.BlueSkyToTwitterID(*retweetPostURI).String()

	return c.JSON(bridge.Retweet{
		Tweet:           retweet,
		RetweetedStatus: TranslatePostToTweet(originalPost.Thread.Post, originalPost.Thread.Post.URI, originalPost.Thread.Parent.Author.DID, nil), // TODO: make this respond with proper retweet data
	})
}
