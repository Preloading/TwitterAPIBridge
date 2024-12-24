package twitterv1

import (
	"fmt"
	"math/big"
	"time"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
)

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/post/statuses/update
func status_update(c *fiber.Ctx) error {
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	status := c.FormValue("status")
	//	trim_user := c.FormValue("trim_user") // Unused
	encoded_in_reply_to_status_id_str := c.FormValue("in_reply_to_status_id")
	encoded_in_reply_to_status_id_int := new(big.Int)
	encoded_in_reply_to_status_id_int, ok := encoded_in_reply_to_status_id_int.SetString(encoded_in_reply_to_status_id_str, 10)
	var in_reply_to_status_id *string
	if ok {
		in_reply_to_status_id, _, _, err = bridge.TwitterMsgIdToBluesky(encoded_in_reply_to_status_id_int)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid in_reply_to_status_id format")
		}
	}

	thread, err := blueskyapi.UpdateStatus(*pds, *oauthToken, *my_did, status, in_reply_to_status_id)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update status")
	}

	if thread.Thread.Parent == nil {
		return c.JSON(TranslatePostToTweet(thread.Thread.Post, "", "", nil, nil, *oauthToken, *pds))
	} else {
		return c.JSON(TranslatePostToTweet(thread.Thread.Post, thread.Thread.Parent.URI, thread.Thread.Parent.Author.DID, &thread.Thread.Parent.Record.CreatedAt, nil, *oauthToken, *pds))
	}
}

// https://web.archive.org/web/20120407091252/https://dev.twitter.com/docs/api/1/post/statuses/retweet/%3Aid
func retweet(c *fiber.Ctx) error {
	postId := c.Params("id")
	user_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Get our IDs
	idBigInt, ok := new(big.Int).SetString(postId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postIdPtr, _, _, err := bridge.TwitterMsgIdToBluesky(idBigInt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postId = *postIdPtr

	err, originalPost, retweetPostURI := blueskyapi.ReTweet(*pds, *oauthToken, postId, *user_did)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update status")
	}

	var retweet bridge.Tweet
	if originalPost.Thread.Parent == nil {
		retweet = TranslatePostToTweet(originalPost.Thread.Post, "", "", nil, nil, *oauthToken, *pds)
	} else {
		retweet = TranslatePostToTweet(originalPost.Thread.Post, originalPost.Thread.Parent.URI, originalPost.Thread.Parent.Author.DID, &originalPost.Thread.Parent.Record.CreatedAt, nil, *oauthToken, *pds)
	}
	retweet.Retweeted = true
	now := time.Now() // pain, also fix this to use the proper timestamp according to the server.
	retweet.ID = bridge.BskyMsgToTwitterID(*retweetPostURI, &now, user_did)
	retweet.IDStr = retweet.ID.String()

	return c.JSON(bridge.Retweet{
		Tweet: retweet,
		RetweetedStatus: func() bridge.Tweet { // TODO: make this respond with proper retweet data
			if originalPost.Thread.Parent == nil {
				return TranslatePostToTweet(originalPost.Thread.Post, "", "", nil, nil, *oauthToken, *pds)
			} else {
				return TranslatePostToTweet(originalPost.Thread.Post, originalPost.Thread.Parent.URI, originalPost.Thread.Parent.Author.DID, &originalPost.Thread.Parent.Record.CreatedAt, nil, *oauthToken, *pds)
			}
		}(),
	})
}

// https://web.archive.org/web/20120412065707/https://dev.twitter.com/docs/api/1/post/favorites/create/%3Aid
func favourite(c *fiber.Ctx) error {
	postId := c.Params("id")
	user_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Fetch ID
	idBigInt, ok := new(big.Int).SetString(postId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postIdPtr, _, _, err := bridge.TwitterMsgIdToBluesky(idBigInt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postId = *postIdPtr

	fmt.Println("Post ID:", postId)

	err, post := blueskyapi.LikePost(*pds, *oauthToken, postId, *user_did)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to like post")
	}

	var newTweet bridge.Tweet
	if post.Thread.Parent == nil {
		newTweet = TranslatePostToTweet(post.Thread.Post, "", "", nil, nil, *oauthToken, *pds)
	} else {
		newTweet = TranslatePostToTweet(post.Thread.Post, post.Thread.Parent.URI, post.Thread.Parent.Author.DID, &post.Thread.Parent.Record.CreatedAt, nil, *oauthToken, *pds)
	}

	return c.JSON(newTweet)
}

// https://web.archive.org/web/20120412041153/https://dev.twitter.com/docs/api/1/post/favorites/destroy/%3Aid
func Unfavourite(c *fiber.Ctx) error { // yes i am canadian
	postId := c.Params("id")
	user_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Fetch ID
	idBigInt, ok := new(big.Int).SetString(postId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postIdPtr, _, _, err := bridge.TwitterMsgIdToBluesky(idBigInt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postId = *postIdPtr

	err, post := blueskyapi.UnlikePost(*pds, *oauthToken, postId, *user_did)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to unlike post")
	}

	var newTweet bridge.Tweet
	if post.Thread.Parent == nil {
		newTweet = TranslatePostToTweet(post.Thread.Post, "", "", nil, nil, *oauthToken, *pds)
	} else {
		newTweet = TranslatePostToTweet(post.Thread.Post, post.Thread.Parent.URI, post.Thread.Parent.Author.DID, &post.Thread.Parent.Record.CreatedAt, nil, *oauthToken, *pds)
	}

	return c.JSON(newTweet)
}

// This handles deleting a tweet, retweet, or reply
func DeleteTweet(c *fiber.Ctx) error {
	postId := c.Params("id")
	user_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Fetch ID
	idBigInt, ok := new(big.Int).SetString(postId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postIdPtr, _, repostUser, err := bridge.TwitterMsgIdToBluesky(idBigInt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postId = *postIdPtr

	err, postToDelete := blueskyapi.GetPost(*pds, *oauthToken, postId, 0, 0)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get post to delete")
	}

	collection := "app.bsky.feed.post"
	// Check if the post is a retweet
	if repostUser != nil && *repostUser != "" {
		if repostUser != user_did {
			return c.Status(fiber.StatusUnauthorized).SendString("You can only delete your own posts")
		}
		collection = "app.bsky.feed.repost"
		postId = *postToDelete.Thread.Post.Viewer.Repost
	} else {
		if postToDelete.Thread.Post.Author.DID != *user_did {
			return c.Status(fiber.StatusUnauthorized).SendString("You can only delete your own posts")
		}
	}

	if err := blueskyapi.DeleteRecord(*pds, *oauthToken, postId, *user_did, collection); err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete post")
	}

	postToDelete.Thread.Post.URI = postId
	postToDelete.Thread.Post.Author.DID = *user_did

	return c.JSON(
		func() bridge.Tweet { // TODO: make this respond with proper retweet data
			if postToDelete.Thread.Parent == nil {
				return TranslatePostToTweet(postToDelete.Thread.Post, "", "", nil, nil, *oauthToken, *pds)
			} else {
				return TranslatePostToTweet(postToDelete.Thread.Post, postToDelete.Thread.Parent.URI, postToDelete.Thread.Parent.Author.DID, &postToDelete.Thread.Parent.Record.CreatedAt, nil, *oauthToken, *pds)
			}
		}(),
	)
}
