package twitterv1

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/gofiber/fiber/v2"
)

func UserSearch(c *fiber.Ctx) error {
	searchQuery := c.Query("q")
	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return MissingAuth(c)
	}
	// Search for users
	bskyUsers, err := blueskyapi.UserSearch(*pds, *oauthToken, searchQuery)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to search")
	}
	// Get complete user info.
	// We must do this as the search API only returns a subset of the user info, and twitter wants all of it.

	// Extract the dids into a string array
	var dids []string
	for _, user := range bskyUsers {
		dids = append(dids, user.DID)
	}
	if len(dids) == 0 {
		return EncodeAndSend(c, []bridge.TwitterUser{})
	}
	users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, dids, false)
	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
	}

	return EncodeAndSend(c, users)
}

// Something interesting with this endpoint is that the twitter client requested:
// /i/search/typeahead.json?count=2500&prefetch=true&result_type=users&send_error_codes=1
// Is this trying to get the first 2500 users for speed reasons?
// Or is this for some inital search suggestions?
// It was sent on right after login sooooo idk.

// https://web.archive.org/web/20220427214446/https://twitter.com/i/search/typeahead.json?count=20&filters=true&result_type=true&src=COMPOSE&q=firat_ber
func SearchAhead(c *fiber.Ctx) error {
	// for completed in time
	start := time.Now()

	if strings.Contains(c.Query("result_type"), "users") {
		// unimplemented, so we'll return it blank.
		return EncodeAndSend(c, bridge.SearchAhead{
			NumberOfResults: 0,
			Query:           c.Query("q"),
			CompletedIn:     time.Since(start).Seconds(),
		})
	}

	searchQuery := c.Query("q")
	if searchQuery == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing search query (or that we don't support prefetch right now)")
	}

	_, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return MissingAuth(c)
	}

	limit := c.Query("count")
	if limit == "" {
		limit = "10"
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid limit parameter")
	}

	// Search for users
	bskyUsers, err := blueskyapi.UserSearchAhead(*pds, *oauthToken, searchQuery, limitInt)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to search")
	}

	if len(bskyUsers) == 0 {
		return EncodeAndSend(c, bridge.SearchAhead{
			NumberOfResults: 0,
			Query:           c.Query("q"),
			CompletedIn:     time.Since(start).Seconds(),
		})
	}

	// Converting into a summarized version of the user
	users := make([]bridge.SummarisedUser, len(bskyUsers))
	for i, user := range bskyUsers {
		userId := bridge.BlueSkyToTwitterID(user.DID)
		pfp_url := configData.CdnURL + "/cdn/img/?url=" + url.QueryEscape(user.Avatar) + ":profile_bigger"
		users[i] = bridge.SummarisedUser{
			ID:                   *userId,
			IDStr:                strconv.FormatInt(*userId, 10),
			ScreenName:           user.Handle,
			Name:                 user.DisplayName,
			IsDMAble:             true,  // take too much effort to figure this out
			IsBlocked:            false, // same for this
			ProfileImageURL:      pfp_url,
			ProfileImageURLHttps: pfp_url,
			Location:             "",
			IsProtected:          false,
			Verified:             false,
			ConnectedUserCount:   0,
			ConnectedUserIds:     []int64{},
			RoundedScore:         69420,
			SocialProofsOrdered:  []string{},
			SocialContext: bridge.SocialContext{ // Kinda expensive to get this
				Following:  false,
				FollowedBy: false,
			},
			Inline: false,
		}
	}

	return EncodeAndSend(c, bridge.SearchAhead{
		NumberOfResults: len(users),
		Users:           users,
		Query:           c.Query("q"),
		CompletedIn:     time.Since(start).Seconds(),
	})
}

type TweetWithURI struct {
	Tweet *bridge.Tweet
	URI   string
}

// /i/activity/about_me.json?contributor_details=1&include_entities=true&include_my_retweet=true&send_error_codes=true
func GetMyActivity(c *fiber.Ctx) error {
	// Thank you so much @Savefade for what this returns for follows.
	// This function very optimized because before it would take 7 seconds lmao
	// we thank our AI overloads.
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return MissingAuth(c)
	}

	// Handle pagination
	context := ""
	maxID := c.Query("max_id")
	if maxID != "" {
		maxIDInt, err := strconv.ParseInt(maxID, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid max_id")
		}
		maxIDInt--
		max_time := time.UnixMilli(maxIDInt)
		context = max_time.Format(time.RFC3339)
	}

	// Handle count
	count := 20
	if countStr := c.Query("count"); countStr != "" {
		if countInt, err := strconv.Atoi(countStr); err == nil {
			count = countInt
		}
	}

	// Get notifications
	bskyNotifications, err := blueskyapi.GetNotifications(*pds, *oauthToken, count, context)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get notifications")
	}

	// Track unique users and posts
	uniqueUsers := make(map[string]bool)
	uniquePosts := make(map[string]bool)

	// First pass: collect unique users and posts
	for _, notification := range bskyNotifications.Notifications {
		uniqueUsers[notification.Author.DID] = true
		if notification.ReasonSubject != "" {
			uniquePosts[notification.ReasonSubject] = true
		}
		if notification.Reason == "mention" || notification.Reason == "reply" {
			uniquePosts[notification.URI] = true
		}
	}

	// Convert maps to slices
	usersToLookUp := make([]string, 0, len(uniqueUsers))
	postsToLookUp := make([]string, 0, len(uniquePosts))
	for user := range uniqueUsers {
		usersToLookUp = append(usersToLookUp, user)
	}
	for post := range uniquePosts {
		postsToLookUp = append(postsToLookUp, post)
	}

	// Create thread-safe maps for results
	var userCache sync.Map
	var postCache sync.Map

	// Process in parallel
	var wg sync.WaitGroup

	// Fetch users in chunks
	wg.Add(1)
	go func() {
		defer wg.Done()
		users, err := blueskyapi.GetUsersInfo(*pds, *oauthToken, usersToLookUp, false)
		if err == nil {
			for _, user := range users {
				// Store by DID instead of screenName
				userCache.Store(user.ScreenName[strings.LastIndex(user.ScreenName, "/")+1:], user)
			}
		}
	}()

	// Fetch posts in parallel chunks
	postChunks := chunkSlice(postsToLookUp, 10) // Process 10 posts at a time
	for _, chunk := range postChunks {
		wg.Add(1)
		go func(posts []string) {
			defer wg.Done()
			for _, postID := range posts {
				if err, post := blueskyapi.GetPost(*pds, *oauthToken, postID, 0, 1); err == nil {
					tweet := TranslatePostToTweet(
						post.Thread.Post,
						func() string {
							if post.Thread.Parent != nil {
								return post.Thread.Parent.Post.URI
							}
							return ""
						}(),
						func() string {
							if post.Thread.Parent != nil {
								return post.Thread.Parent.Post.Author.DID
							}
							return ""
						}(),
						func() string {
							if post.Thread.Parent != nil {
								return post.Thread.Parent.Post.Author.Handle
							}
							return ""
						}(),
						func() *time.Time {
							if post.Thread.Parent != nil {
								return &post.Thread.Parent.Post.IndexedAt
							}
							return nil
						}(),
						nil,
						*oauthToken,
						*pds,
					)
					postCache.Store(postID, &tweet)
				}
			}
		}(chunk)
	}

	wg.Wait()

	// Process notifications in groups
	twitterNotifications := []bridge.MyActivity{}
	groupedNotifications := groupNotifications(bskyNotifications.Notifications)

	// Convert each group to a Twitter activity
	for _, group := range groupedNotifications {
		if activity := processNotificationGroup(group, &userCache, &postCache, my_did); activity != nil {
			twitterNotifications = append(twitterNotifications, *activity)
		}
	}

	return EncodeAndSend(c, twitterNotifications)
}

// Helper functions

func chunkSlice(slice []string, chunkSize int) [][]string {
	var chunks [][]string
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

type notificationGroup struct {
	notifications []blueskyapi.Notification
	reason        string
	subject       string
}

func groupNotifications(notifications []blueskyapi.Notification) []notificationGroup {
	groups := []notificationGroup{}
	currentGroup := notificationGroup{}

	for _, notification := range notifications {
		if currentGroup.reason != notification.Reason ||
			(notification.ReasonSubject != "" && currentGroup.subject != notification.ReasonSubject) {
			if currentGroup.reason != "" {
				groups = append(groups, currentGroup)
			}
			currentGroup = notificationGroup{
				reason:  notification.Reason,
				subject: notification.ReasonSubject,
			}
		}
		currentGroup.notifications = append(currentGroup.notifications, notification)
	}

	if currentGroup.reason != "" {
		groups = append(groups, currentGroup)
	}

	return groups
}

func processNotificationGroup(group notificationGroup, userCache *sync.Map, postCache *sync.Map, myDID *string) *bridge.MyActivity {
	if len(group.notifications) == 0 {
		return nil
	}

	first := group.notifications[0]
	last := group.notifications[len(group.notifications)-1]

	var sources []bridge.TwitterUser
	for _, notification := range group.notifications {
		// Load by DID
		if user, ok := userCache.Load(notification.Author.Handle); ok {
			sources = append(sources, *user.(*bridge.TwitterUser))
		}
	}

	activity := &bridge.MyActivity{
		Action:        getActionType(group.reason),
		CreatedAt:     bridge.TwitterTimeConverter(first.IndexedAt),
		MinPosition:   strconv.FormatInt(first.IndexedAt.UnixMilli(), 10),
		MaxPosition:   strconv.FormatInt(last.IndexedAt.UnixMilli(), 10),
		Sources:       sources,
		Targets:       []bridge.Tweet{},
		TargetObjects: []bridge.Tweet{},
	}

	if group.subject != "" {
		if post, ok := postCache.Load(group.subject); ok {
			tweet := post.(*bridge.Tweet)
			switch group.reason {
			case "like":
				activity.Targets = []bridge.Tweet{*tweet}
			case "repost":
				activity.TargetObjects = []bridge.Tweet{*tweet}
			case "reply":
				if post, ok := postCache.Load(group.notifications[0].URI); ok {
					replytweet := post.(*bridge.Tweet)
					activity.Targets = []bridge.Tweet{*replytweet}
					activity.TargetObjects = []bridge.Tweet{*tweet}
				}
			}
		}
	} else {
		switch group.reason {
		case "mention", "quote":
			if post, ok := postCache.Load(group.notifications[0].URI); ok {
				tweet := post.(*bridge.Tweet)
				activity.Sources = []bridge.TwitterUser{}
				activity.TargetObjects = []bridge.Tweet{*tweet}
			}
		}
	}

	return activity
}

func getActionType(reason string) string {
	switch reason {
	case "follow":
		return "follow"
	case "like":
		return "favorite"
	case "repost":
		return "retweet"
	case "mention":
		return "mention"
	case "quote":
		return "mention"
	case "reply":
		return "reply"
	default:
		return reason
	}
}
