package twitterv1

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"regexp"
	"strconv"
	"time"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/Preloading/TwitterAPIBridge/db_controller"
	"github.com/gofiber/fiber/v2"
)

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/post/statuses/update
func status_update(c *fiber.Ctx) error {
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return MissingAuth(c, err)
	}

	status := c.FormValue("status")

	// Status parsing!
	mentions := findHandleInstances(status)
	links := findUrlInstances(status)
	tags := findTagInstances(status)

	//	trim_user := c.FormValue("trim_user") // Unused
	encoded_in_reply_to_status_id_str := c.FormValue("in_reply_to_status_id")
	encoded_in_reply_to_status_id_int, err := strconv.ParseInt(encoded_in_reply_to_status_id_str, 10, 64)
	var in_reply_to_status_id *string
	if err == nil {
		in_reply_to_status_id, _, _, err = bridge.TwitterMsgIdToBluesky(&encoded_in_reply_to_status_id_int)
		if err != nil {
			return ReturnError(c, "The in_reply_to_status_id was not found", 144, fiber.StatusNotFound)
		}
	}

	thread, err := blueskyapi.UpdateStatus(*pds, *oauthToken, *my_did, status, in_reply_to_status_id, mentions, links, tags, nil, []int{})

	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.createRecord", status_update)
	}

	db_controller.StoreAnalyticData(db_controller.AnalyticData{
		DataType:             "status_update",
		IPAddress:            c.IP(),
		UserAgent:            c.Get("User-Agent"),
		Language:             c.Get("Accept-Language"),
		TwitterClient:        c.Get("X-Twitter-Client"),
		TwitterClientVersion: c.Get("X-Twitter-Client-Version"),
		Timestamp:            time.Now(),
	})

	if thread.Thread.Parent == nil {
		return EncodeAndSend(c, TranslatePostToTweet(thread.Thread.Post, "", "", "", nil, nil, *oauthToken, *pds))
	} else {
		return EncodeAndSend(c, TranslatePostToTweet(thread.Thread.Post, thread.Thread.Parent.Post.URI, thread.Thread.Parent.Post.Author.DID, thread.Thread.Parent.Post.Author.Handle, &thread.Thread.Parent.Post.Record.CreatedAt.Time, nil, *oauthToken, *pds))
	}
}

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/post/statuses/update
func status_update_with_media(c *fiber.Ctx) error {
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return MissingAuth(c, err)
	}

	status := c.FormValue("status")

	// The docs say it's an array, but I can only upload one imageData.... so idk
	imageData, err := c.FormFile("media") // i love it when things dont follow the docs
	if err != nil {
		imageData, err = c.FormFile("media[]")
		if err != nil {
			fmt.Println("Error:", err)
			return ReturnError(c, "Please upload an image", 195, fiber.StatusForbidden)
		}
	}

	// read the image file content
	file, err := imageData.Open()
	if err != nil {
		fmt.Println("Error:", err)
		return ReturnError(c, "An invalid image was uploaded", 195, fiber.StatusForbidden)
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error:", err)
		return ReturnError(c, "Failed to process image", 131, fiber.StatusInternalServerError)
	}

	// Get image resolution
	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil {
		fmt.Println("Error:", err)
		return ReturnError(c, "Failed to process image", 131, fiber.StatusInternalServerError)
	}

	// upload our new profile picture
	imageBlob, err := blueskyapi.UploadBlob(*pds, *oauthToken, imageBytes, c.Get("Content-Type"))
	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.uploadBlob", status_update_with_media)
	}

	// Status parsing!
	mentions := findHandleInstances(status)
	links := findUrlInstances(status)
	tags := findTagInstances(status)

	//	trim_user := c.FormValue("trim_user") // Unused
	encoded_in_reply_to_status_id_str := c.FormValue("in_reply_to_status_id")
	encoded_in_reply_to_status_id_int, err := strconv.ParseInt(encoded_in_reply_to_status_id_str, 10, 64)
	var in_reply_to_status_id *string
	if err == nil {
		in_reply_to_status_id, _, _, err = bridge.TwitterMsgIdToBluesky(&encoded_in_reply_to_status_id_int)
		if err != nil {
			return ReturnError(c, "The in_reply_to_status_id was not found", 144, fiber.StatusNotFound)
		}
	}

	thread, err := blueskyapi.UpdateStatus(*pds, *oauthToken,
		*my_did,
		status,
		in_reply_to_status_id,
		mentions,
		links,
		tags,
		imageBlob,
		[]int{imageConfig.Height, imageConfig.Width},
	)

	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.createRecord", status_update_with_media)
	}

	db_controller.StoreAnalyticData(db_controller.AnalyticData{
		DataType:             "status_update",
		IPAddress:            c.IP(),
		UserAgent:            c.Get("User-Agent"),
		Language:             c.Get("Accept-Language"),
		TwitterClient:        c.Get("X-Twitter-Client"),
		TwitterClientVersion: c.Get("X-Twitter-Client-Version"),
		Timestamp:            time.Now(),
	})

	if thread.Thread.Parent == nil {
		return EncodeAndSend(c, TranslatePostToTweet(thread.Thread.Post, "", "", "", nil, nil, *oauthToken, *pds))
	} else {
		return EncodeAndSend(c, TranslatePostToTweet(thread.Thread.Post, thread.Thread.Parent.Post.URI, thread.Thread.Parent.Post.Author.DID, thread.Thread.Parent.Post.Author.Handle, &thread.Thread.Parent.Post.Record.CreatedAt.Time, nil, *oauthToken, *pds))
	}
}

// https://web.archive.org/web/20120407091252/https://dev.twitter.com/docs/api/1/post/statuses/retweet/%3Aid
func retweet(c *fiber.Ctx) error {
	postId := c.Params("id")
	user_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return MissingAuth(c, err)
	}

	// Get our IDs
	idBigInt, err := strconv.ParseInt(postId, 10, 64)
	if err != nil {
		return ReturnError(c, "Invalid ID format", 195, 403)
	}
	postIdPtr, _, _, err := bridge.TwitterMsgIdToBluesky(&idBigInt)
	if err != nil {
		return ReturnError(c, "Id not found.", 144, fiber.StatusNotFound)
	}
	postId = *postIdPtr

	originalPost, blueskyRepostURI, err := blueskyapi.ReTweet(*pds, *oauthToken, postId, *user_did)

	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.createRecord", retweet)
	}

	var retweet bridge.Tweet
	if originalPost.Thread.Parent == nil {
		retweet = TranslatePostToTweet(originalPost.Thread.Post, "", "", "", nil, nil, *oauthToken, *pds)
	} else {
		retweet = TranslatePostToTweet(originalPost.Thread.Post, originalPost.Thread.Parent.Post.URI, originalPost.Thread.Parent.Post.Author.DID, originalPost.Thread.Parent.Post.Author.Handle, &originalPost.Thread.Parent.Post.Record.CreatedAt.Time, nil, *oauthToken, *pds)
	}
	retweet.Retweeted = true
	now := time.Now() // pain, also fix this to use the proper timestamp according to the server.
	retweetId := bridge.BskyMsgToTwitterID(originalPost.Thread.Post.URI, &now, user_did)
	retweet.ID = *retweetId
	retweet.IDStr = strconv.FormatInt(retweet.ID, 10)
	originalPost.Thread.Post.Viewer.Repost = blueskyRepostURI

	db_controller.StoreAnalyticData(db_controller.AnalyticData{
		DataType:             "retweeted",
		IPAddress:            c.IP(),
		UserAgent:            c.Get("User-Agent"),
		Language:             c.Get("Accept-Language"),
		TwitterClient:        c.Get("X-Twitter-Client"),
		TwitterClientVersion: c.Get("X-Twitter-Client-Version"),
		Timestamp:            time.Now(),
	})

	return EncodeAndSend(c, bridge.Retweet{
		Tweet: retweet,
		RetweetedStatus: func() bridge.Tweet { // TODO: make this respond with proper retweet data
			if originalPost.Thread.Parent == nil {
				return TranslatePostToTweet(originalPost.Thread.Post, "", "", "", nil, nil, *oauthToken, *pds)
			} else {
				return TranslatePostToTweet(originalPost.Thread.Post, originalPost.Thread.Parent.Post.URI, originalPost.Thread.Parent.Post.Author.DID, originalPost.Thread.Parent.Post.Author.Handle, &originalPost.Thread.Parent.Post.Record.CreatedAt.Time, nil, *oauthToken, *pds)
			}
		}(),
	})
}

// https://web.archive.org/web/20120412065707/https://dev.twitter.com/docs/api/1/post/favorites/create/%3Aid
func favourite(c *fiber.Ctx) error {
	postId := c.Params("id")
	user_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return MissingAuth(c, err)
	}

	// Fetch ID
	idBigInt, err := strconv.ParseInt(postId, 10, 64)
	if err != nil {
		return ReturnError(c, "Invalid ID format", 195, 403)
	}
	postIdPtr, _, _, err := bridge.TwitterMsgIdToBluesky(&idBigInt)
	if err != nil {
		return ReturnError(c, "Id not found.", 144, fiber.StatusNotFound)
	}
	postId = *postIdPtr

	post, err := blueskyapi.LikePost(*pds, *oauthToken, postId, *user_did)

	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.createRecord", favourite)
	}

	var newTweet bridge.Tweet
	if post.Thread.Parent == nil {
		newTweet = TranslatePostToTweet(post.Thread.Post, "", "", "", nil, nil, *oauthToken, *pds)
	} else {
		newTweet = TranslatePostToTweet(post.Thread.Post, post.Thread.Parent.Post.URI, post.Thread.Parent.Post.Author.DID, post.Thread.Parent.Post.Author.Handle, &post.Thread.Parent.Post.Record.CreatedAt.Time, nil, *oauthToken, *pds)
	}

	db_controller.StoreAnalyticData(db_controller.AnalyticData{
		DataType:             "favorited",
		IPAddress:            c.IP(),
		UserAgent:            c.Get("User-Agent"),
		Language:             c.Get("Accept-Language"),
		TwitterClient:        c.Get("X-Twitter-Client"),
		TwitterClientVersion: c.Get("X-Twitter-Client-Version"),
		Timestamp:            time.Now(),
	})

	return EncodeAndSend(c, newTweet)
}

// https://web.archive.org/web/20120412041153/https://dev.twitter.com/docs/api/1/post/favorites/destroy/%3Aid
func Unfavourite(c *fiber.Ctx) error { // yes i am canadian
	postId := c.Params("id")
	user_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return MissingAuth(c, err)
	}

	// Fetch ID
	idBigInt, err := strconv.ParseInt(postId, 10, 64)
	if err != nil {
		return ReturnError(c, "Invalid ID format", 195, 403)
	}
	postIdPtr, _, _, err := bridge.TwitterMsgIdToBluesky(&idBigInt)
	if err != nil {
		return ReturnError(c, "ID not found.", 144, fiber.StatusNotFound)
	}
	postId = *postIdPtr

	post, err := blueskyapi.UnlikePost(*pds, *oauthToken, postId, *user_did)

	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.deleteRecord", Unfavourite)
	}

	var newTweet bridge.Tweet
	if post.Thread.Parent == nil {
		newTweet = TranslatePostToTweet(post.Thread.Post, "", "", "", nil, nil, *oauthToken, *pds)
	} else {
		newTweet = TranslatePostToTweet(post.Thread.Post, post.Thread.Parent.Post.URI, post.Thread.Parent.Post.Author.DID, post.Thread.Parent.Post.Author.Handle, &post.Thread.Parent.Post.Record.CreatedAt.Time, nil, *oauthToken, *pds)
	}

	db_controller.StoreAnalyticData(db_controller.AnalyticData{
		DataType:             "unfavorite",
		IPAddress:            c.IP(),
		UserAgent:            c.Get("User-Agent"),
		Language:             c.Get("Accept-Language"),
		TwitterClient:        c.Get("X-Twitter-Client"),
		TwitterClientVersion: c.Get("X-Twitter-Client-Version"),
		Timestamp:            time.Now(),
	})

	return EncodeAndSend(c, newTweet)
}

// This handles deleting a tweet, retweet, or reply
func DeleteTweet(c *fiber.Ctx) error {
	postId := c.Params("id")
	user_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return MissingAuth(c, err)
	}

	// Fetch ID
	idBigInt, err := strconv.ParseInt(postId, 10, 64)
	if err != nil {
		return ReturnError(c, "Invalid ID format", 195, 403)
	}
	postIdPtr, _, repostUser, err := bridge.TwitterMsgIdToBluesky(&idBigInt)
	if err != nil {
		return ReturnError(c, "Id not found.", 144, fiber.StatusNotFound)
	}
	postId = *postIdPtr

	postToDelete, err := blueskyapi.GetPost(*pds, *oauthToken, postId, 0, 0)

	if err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.deleteRecord", DeleteTweet)
	}

	collection := "app.bsky.feed.post"
	// Check if the post is a retweet
	if repostUser != nil && *repostUser != "" {
		if *repostUser != *user_did {
			return ReturnError(c, "You can only delete your own retweets", 195, fiber.StatusForbidden)
		}
		collection = "app.bsky.feed.repost"
		if postToDelete.Thread.Post.Viewer.Repost == nil {
			return ReturnError(c, "You have to retweet before you can unretweet", 195, fiber.StatusForbidden)
		}
		postId = *postToDelete.Thread.Post.Viewer.Repost
	} else {
		if postToDelete.Thread.Post.Author.DID != *user_did {
			return ReturnError(c, "You can only delete your own tweets", 195, fiber.StatusForbidden)
		}
	}

	if err := blueskyapi.DeleteRecord(*pds, *oauthToken, postId, *user_did, collection); err != nil {
		fmt.Println("Error:", err)
		return HandleBlueskyError(c, err.Error(), "com.atproto.repo.deleteRecord", DeleteTweet)
	}

	db_controller.StoreAnalyticData(db_controller.AnalyticData{
		DataType:             "deleted_post",
		IPAddress:            c.IP(),
		UserAgent:            c.Get("User-Agent"),
		Language:             c.Get("Accept-Language"),
		TwitterClient:        c.Get("X-Twitter-Client"),
		TwitterClientVersion: c.Get("X-Twitter-Client-Version"),
		Timestamp:            time.Now(),
	})

	postToDelete.Thread.Post.URI = postId
	postToDelete.Thread.Post.Author.DID = *user_did

	return EncodeAndSend(c,
		func() bridge.Tweet { // TODO: make this respond with proper retweet data
			if postToDelete.Thread.Parent == nil {
				return TranslatePostToTweet(postToDelete.Thread.Post, "", "", "", nil, nil, *oauthToken, *pds)
			} else {
				return TranslatePostToTweet(postToDelete.Thread.Post, postToDelete.Thread.Parent.Post.URI, postToDelete.Thread.Parent.Post.Author.DID, postToDelete.Thread.Parent.Post.Author.Handle, &postToDelete.Thread.Parent.Post.Record.CreatedAt.Time, nil, *oauthToken, *pds)
			}
		}(),
	)
}

func findHandleInstances(input string) []bridge.FacetParsing {
	regex := regexp.MustCompile(`@([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?`)
	matches := regex.FindAllStringIndex(input, -1)
	results := []bridge.FacetParsing{}
	for _, match := range matches {
		results = append(results, bridge.FacetParsing{
			Start: match[0],
			End:   match[1],
			Item:  input[match[0]+1 : match[1]], // +1 to skip the '@' character
		})
	}
	return results
}

func findUrlInstances(input string) []bridge.FacetParsing {
	regex := regexp.MustCompile(`[$|\W](https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*[-a-zA-Z0-9@%_\+~#//=])?)`)
	matches := regex.FindAllStringIndex(input, -1)
	results := []bridge.FacetParsing{}
	for _, match := range matches {
		results = append(results, bridge.FacetParsing{
			Start: match[0] + 1,
			End:   match[1],
			Item:  input[match[0]+1 : match[1]],
		})
	}
	return results
}

func findTagInstances(input string) []bridge.FacetParsing {
	regex := regexp.MustCompile(`#[a-zA-Z0-9_]+`)
	matches := regex.FindAllStringIndex(input, -1)
	results := []bridge.FacetParsing{}
	for _, match := range matches {
		results = append(results, bridge.FacetParsing{
			Start: match[0],
			End:   match[1],
			Item:  input[match[0]+1 : match[1]],
		})
	}
	return results
}
