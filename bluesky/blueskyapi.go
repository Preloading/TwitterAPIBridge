package blueskyapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Preloading/MastodonTwitterAPI/bridge"
)

type AuthResponse struct {
	AccessJwt  string `json:"accessJwt"`
	RefreshJwt string `json:"refreshJwt"`
	DID        string `json:"did"`
}

type AuthRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// https://docs.bsky.app/docs/api/app-bsky-actor-get-profile
type User struct {
	DID            string    `json:"did"`
	Handle         string    `json:"handle"`
	DisplayName    string    `json:"displayName"`
	Description    string    `json:"description"`
	Avatar         string    `json:"avatar"`
	Banner         string    `json:"banner"`
	FollowersCount int       `json:"followersCount"`
	FollowsCount   int       `json:"followsCount"`
	PostsCount     int       `json:"postsCount"`
	IndexedAt      time.Time `json:"indexedAt"`
	CreatedAt      time.Time `json:"createdAt"`
	Associated     struct {
		Lists        int       `json:"lists"`
		FeedGens     int       `json:"feedgens"`
		StarterPacks int       `json:"starterPacks"`
		Labeler      bool      `json:"labeler"`
		CreatedAt    time.Time `json:"created_at"`
	} `json:"associated"`
	Viewer struct {
		Muted bool `json:"muted"`
		// MutedByList
		BlockedBy bool    `json:"blockedBy"`
		Blocking  *string `json:"blocking,omitempty"`
		// BlockingByList
		Following  *string `json:"following,omitempty"`
		FollowedBy *string `json:"followedBy,omitempty"`
		// KnownFollowers
	} `json:"viewer"`
}

type PostRecord struct {
	Type      string        `json:"$type"`
	CreatedAt time.Time     `json:"createdAt"`
	Embed     Embed         `json:"embed"`
	Facets    []Facet       `json:"facets"`
	Langs     []string      `json:"langs"`
	Text      string        `json:"text"`
	Reply     *ReplySubject `json:"reply,omitempty"`
}

// Specifically for reposts
type PostReason struct {
	Type      string    `json:"$type"`
	By        User      `json:"by"`
	IndexedAt time.Time `json:"indexedAt"`
}

type Embed struct {
	Type   string  `json:"$type"`
	Images []Image `json:"images"`
}

type Image struct {
	Alt         string      `json:"alt"`
	AspectRatio AspectRatio `json:"aspectRatio"`
	Image       Blob        `json:"image"`
}

type AspectRatio struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

type Blob struct {
	Type     string `json:"$type"`
	Ref      Ref    `json:"ref"`
	MimeType string `json:"mimeType"`
	Size     int    `json:"size"`
}

type Ref struct {
	Link string `json:"$link"`
}

type Facet struct {
	Features []Feature `json:"features"`
	Index    Index     `json:"index"`
}

type Feature struct {
	Type string `json:"$type"`
	Tag  string `json:"tag"`
	Did  string `json:"did,omitempty"`
	Uri  string `json:"uri,omitempty"`
}

type Index struct {
	ByteEnd   int `json:"byteEnd"`
	ByteStart int `json:"byteStart"`
}

type PostViewer struct {
	Repost            *string `json:"repost"`
	Like              *string `json:"like"` // Can someone please tell me why this is a string.
	Muted             bool    `json:"muted"`
	BlockedBy         bool    `json:"blockedBy"`
	ThreadMute        bool    `json:"threadMute"`
	ReplyDisabled     bool    `json:"replyDisabled"`
	EmbeddingDisabled bool    `json:"embeddingDisabled"`
	Pinned            bool    `json:"pinned"`
}
type Post struct {
	Subject
	Author User       `json:"author"`
	Record PostRecord `json:"record"`
	// Embed  Embed      `json:"embed"`
	ReplyCount  int        `json:"replyCount"`
	RepostCount int        `json:"repostCount"`
	LikeCount   int        `json:"likeCount"`
	QuoteCount  int        `json:"quoteCount"`
	IndexedAt   time.Time  `json:"indexedAt"`
	Viewer      PostViewer `json:"viewer"`
}

type Feed struct {
	Post  Post `json:"post"`
	Reply struct {
		Root   Post `json:"root"`
		Parent Post `json:"parent"`
	} `json:"reply"`
	Reason      *PostReason `json:"reason"`
	FeedContext string      `json:"feedContext"`
}

type Timeline struct {
	Feed   []Feed `json:"feed"`
	Cursor string `json:"cursor"`
}

type Thread struct {
	Type    string    `json:"$type"`
	Post    Post      `json:"post"`
	Parent  *Post     `json:"parent"`
	Replies *[]Thread `json:"replies"`
}

// This is solely for the purpose of unmarshalling the response from the API
type ThreadRoot struct {
	Thread Thread `json:"thread"`
}

// Reposting/Retweeting
type CreateRecordPayload struct {
	Collection string      `json:"collection"`
	Repo       string      `json:"repo"`
	Record     interface{} `json:"record"`
}

type DeleteRecordPayload struct {
	Collection string `json:"collection"`
	Repo       string `json:"repo"`
	RKey       string `json:"rkey"`
}

type PostInteractionRecord struct {
	Type      string  `json:"$type"`
	CreatedAt string  `json:"createdAt"`
	Subject   Subject `json:"subject"`
}

type CreatePostRecord struct {
	Type      string        `json:"$type"`
	Text      string        `json:"text"`
	CreatedAt time.Time     `json:"createdAt"`
	Reply     *ReplySubject `json:"reply,omitempty"`
}

type Subject struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

type ReplySubject struct {
	Root   Subject `json:"root"`
	Parent Subject `json:"parent"`
}

type Commit struct {
	CID string `json:"cid"`
	Rev string `json:"rev"`
}

type CreateRecordResult struct {
	Subject
	Commit           Commit `json:"commit"`
	ValidationStatus string `json:"validationStatus"`
}

type RepostedBy struct {
	Subject
	Cursor     string `json:"cursor"`
	RepostedBy []User `json:"repostedBy"`
}
type Likes struct {
	Subject
	Cursor string           `json:"cursor"`
	Likes  []ItemByWithDate `json:"likes"`
}

type ItemByWithDate struct {
	IndexedAt time.Time `json:"indexedAt"`
	CreatedAt time.Time `json:"createdAt"`
	Actor     User      `json:"actor"`
}

type UserSearchResult struct {
	Actors []User `json:"actors"`
}

type RecordResponse struct {
	URI   string      `json:"uri"`
	CID   string      `json:"cid"`
	Value RecordValue `json:"value"`
}

type RecordValue struct {
	Reply     *ReplySubject `json:"reply,omitempty"`
	CreatedAt time.Time     `json:"createdAt,omitempty"`
}

type Relationships struct {
	DID        string `json:"did"`
	Following  string `json:"following"`
	FollowedBy string `json:"followedBy"`
}

type RelationshipsRes struct {
	Actor         string          `json:"actor"`
	Relationships []Relationships `json:"relationships"`
}

var userCache = bridge.NewCache(5 * time.Minute) // Cache TTL of 5 minutes

func SendRequest(token *string, method string, url string, body io.Reader) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if token != nil {
		req.Header.Set("Authorization", "Bearer "+*token)
	}
	req.Header.Set("Content-Type", "application/json") // 99% sure all bluesky requests are json.

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func GetUserInfo(pds string, token string, screen_name string) (*bridge.TwitterUser, error) {
	if user, found := userCache.Get(screen_name); found {
		return &user, nil
	}

	url := pds + "/xrpc/app.bsky.actor.getProfile" + "?actor=" + screen_name

	resp, err := SendRequest(&token, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to fetch user info")
	}

	author := User{}
	if err := json.NewDecoder(resp.Body).Decode(&author); err != nil {
		return nil, err
	}

	twitterUser := AuthorTTB(author)

	userCache.SetMultiple([]string{author.DID, author.Handle}, *twitterUser)

	return twitterUser, nil
}

func GetUsersInfo(pds string, token string, items []string, ignoreCache bool) ([]*bridge.TwitterUser, error) {
	var results []*bridge.TwitterUser
	var missing []string

	if !ignoreCache {
		for _, screen_name := range items {
			if user, found := userCache.Get(screen_name); found {
				results = append(results, &user)
			} else {
				missing = append(missing, screen_name)
			}
		}

		if len(missing) == 0 {
			return results, nil
		}
	} else {
		missing = items
	}

	// Parallel fetching for chunks of up to 25 at a time
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := 0; i < len(missing); i += 25 {
		end := i + 25
		if end > len(missing) {
			end = len(missing)
		}
		chunk := missing[i:end]

		wg.Add(1)
		go func(c []string) {
			defer wg.Done()

			url := pds + "xrpc/app.bsky.actor.getProfiles" + "?actors=" + strings.Join(c, "&actors=")
			resp, err := SendRequest(&token, http.MethodGet, url, nil)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				fmt.Println("Response Status:", resp.StatusCode)
				fmt.Println("Response Body:", string(bodyBytes))
				return
			}

			var authors struct {
				Profiles []User `json:"profiles"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&authors); err != nil {
				return
			}

			mu.Lock()
			for _, author := range authors.Profiles {
				userObj := AuthorTTB(author)
				userCache.SetMultiple([]string{author.DID, author.Handle}, *userObj)
				results = append(results, userObj)
			}
			mu.Unlock()
		}(chunk)
	}

	wg.Wait()
	return results, nil
}

// TODO: Combine this with GetUsersInfo... somehow
func GetUsersInfoRaw(pds string, token string, items []string, ignoreCache bool) ([]*User, error) {
	var results []*User
	var missing []string

	missing = items // hack

	// Parallel fetching for chunks of up to 25 at a time
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := 0; i < len(missing); i += 25 {
		end := i + 25
		if end > len(missing) {
			end = len(missing)
		}
		chunk := missing[i:end]

		wg.Add(1)
		go func(c []string) {
			defer wg.Done()

			url := pds + "/xrpc/app.bsky.actor.getProfiles" + "?actors=" + strings.Join(c, "&actors=")
			fmt.Println(url)
			resp, err := SendRequest(&token, http.MethodGet, url, nil)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				fmt.Println("Response Status:", resp.StatusCode)
				fmt.Println("Response Body:", string(bodyBytes))
				return
			}

			var authors struct {
				Profiles []User `json:"profiles"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&authors); err != nil {
				return
			}

			mu.Lock()
			for _, author := range authors.Profiles {
				results = append(results, &author)
			}
			mu.Unlock()
		}(chunk)
	}

	wg.Wait()
	return results, nil
}

// https://docs.bsky.app/docs/api/app-bsky-graph-get-relationships
func GetRelationships(pds string, token string, source string, others []string) (*RelationshipsRes, error) {
	url := pds + "/xrpc/app.bsky.graph.getRelationships" + "?actor=" + url.QueryEscape(source) + "&others=" + url.QueryEscape(strings.Join(others, ","))

	resp, err := SendRequest(&token, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// // Print the response body for debugging
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to fetch relationships")
	}

	feeds := RelationshipsRes{}
	if err := json.NewDecoder(resp.Body).Decode(&feeds); err != nil {
		return nil, err
	}

	return &feeds, nil
}

func AuthorTTB(author User) *bridge.TwitterUser {
	return &bridge.TwitterUser{
		ProfileSidebarFillColor: "e0ff92",
		Name: func() string {
			if author.DisplayName == "" {
				return author.Handle
			}
			return author.DisplayName
		}(),
		ProfileSidebarBorderColor: "87bc44",
		ProfileBackgroundTile:     false,
		CreatedAt:                 bridge.TwitterTimeConverter(author.CreatedAt),
		ProfileImageURL:           "http://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape(author.Avatar) + ":profile_bigger",
		Location:                  "",
		ProfileLinkColor:          "0000ff",
		IsTranslator:              false,
		ContributorsEnabled:       false,
		URL:                       "",
		UtcOffset:                 nil,
		ID:                        *bridge.BlueSkyToTwitterID(author.DID),
		ProfileUseBackgroundImage: false,
		ListedCount:               0,
		ProfileTextColor:          "000000",
		Protected:                 false,

		Lang:                   "en",
		Notifications:          nil,
		Verified:               false,
		ProfileBackgroundColor: "c0deed",
		GeoEnabled:             false,
		Description:            author.Description,
		FriendsCount:           author.FollowsCount,
		FollowersCount:         author.FollowersCount,
		StatusesCount:          author.PostsCount,
		//FavouritesCount:        author.,
		ScreenName: author.Handle,
	}
}

// https://docs.bsky.app/docs/api/app-bsky-feed-get-feed
func GetTimeline(pds string, token string, context string, feed string) (error, *Timeline) {
	url := pds + "/xrpc/app.bsky.feed.getTimeline"
	if context != "" {
		url = pds + "/xrpc/app.bsky.feed.getTimeline?cursor=" + context
	}

	resp, err := SendRequest(&token, http.MethodGet, url, nil)
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()

	// // Print the response body for debugging
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to fetch timeline"), nil
	}

	feeds := Timeline{}
	if err := json.NewDecoder(resp.Body).Decode(&feeds); err != nil {
		return err, nil
	}

	return nil, &feeds
}

// https://docs.bsky.app/docs/api/app-bsky-feed-get-author-feed
func GetUserTimeline(pds string, token string, context string, actor string) (error, *Timeline) {
	apiURL := pds + "/xrpc/app.bsky.feed.getAuthorFeed?actor=" + url.QueryEscape(actor)
	if context != "" {
		apiURL = pds + "/xrpc/app.bsky.feed.getAuthorFeed?actor=" + url.QueryEscape(actor) + "&cursor=" + context
	}

	resp, err := SendRequest(&token, http.MethodGet, apiURL, nil)
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()

	// // Print the response body for debugging
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to fetch timeline"), nil
	}

	feeds := Timeline{}
	if err := json.NewDecoder(resp.Body).Decode(&feeds); err != nil {
		return err, nil
	}

	return nil, &feeds
}

func GetPost(pds string, token string, uri string, depth int, parentHeight int) (error, *ThreadRoot) {
	// Example URL at://did:plc:dqibjxtqfn6hydazpetzr2w4/app.bsky.feed.post/3lchbospvbc2j

	url := pds + "/xrpc/app.bsky.feed.getPostThread?depth=" + fmt.Sprintf("%d", depth) + "&parentHeight=" + fmt.Sprintf("%d", parentHeight) + "&uri=" + uri

	resp, err := SendRequest(&token, http.MethodGet, url, nil)
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()

	// // Print the response body
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to fetch timeline"), nil
	}

	thread := ThreadRoot{}
	if err := json.NewDecoder(resp.Body).Decode(&thread); err != nil {
		return err, nil
	}

	return nil, &thread
}

// This handles both normal & replys
func UpdateStatus(pds string, token string, my_did string, status string, in_reply_to *string) (*ThreadRoot, error) {
	url := pds + "/xrpc/com.atproto.repo.createRecord"

	var replySubject *ReplySubject
	var err error

	// Replying
	if in_reply_to != nil && *in_reply_to != "" {
		replySubject, err = GetReplyRefs(pds, token, *in_reply_to)
		if err != nil {
			return nil, errors.New("failed to fetch reply refs")
		}
	}

	payload := CreateRecordPayload{
		Collection: "app.bsky.feed.post",
		Repo:       my_did,
		Record: CreatePostRecord{
			Type:      "app.bsky.feed.post",
			Text:      status,
			CreatedAt: time.Now().UTC(),
			Reply:     replySubject,
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("failed to marshal payload")
	}
	resp, err := SendRequest(&token, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, errors.New("failed to post")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to update status")
	}

	postData := CreateRecordResult{}
	if err := json.NewDecoder(resp.Body).Decode(&postData); err != nil {
		return nil, err
	}

	time.Sleep(100 * time.Millisecond) // Bluesky doesn't update instantly, so we wait a bit before fetching the post

	err, thread := GetPost(pds, token, postData.URI, 0, 1)
	if err != nil {
		return nil, errors.New("failed to fetch made post")
	}

	return thread, nil
}

func DeleteRecord(pds string, token string, id string, my_did string, collection string) error {
	url := pds + "/xrpc/com.atproto.repo.deleteRecord"

	payload := DeleteRecordPayload{
		Collection: collection,
		Repo:       my_did,
		RKey:       strings.Split(id, "/"+collection+"/")[1],
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return errors.New("failed to marshal payload")
	}

	resp, err := SendRequest(&token, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to retweet: " + bodyString)
	}
	return nil
}

func ReTweet(pds string, token string, id string, my_did string) (error, *ThreadRoot, *string) {
	url := pds + "/xrpc/com.atproto.repo.createRecord"

	err, thread := GetPost(pds, token, id, 0, 1)
	if err != nil {
		return errors.New("failed to fetch post"), nil, nil
	}

	payload := CreateRecordPayload{
		Collection: "app.bsky.feed.repost",
		Repo:       my_did,
		Record: PostInteractionRecord{
			Type:      "app.bsky.feed.repost",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Subject: Subject{
				URI: thread.Thread.Post.URI,
				CID: thread.Thread.Post.CID,
			},
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return errors.New("failed to marshal payload"), nil, nil
	}

	resp, err := SendRequest(&token, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err, nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to retweet: " + bodyString), nil, nil
	}

	repost := CreateRecordResult{}
	if err := json.NewDecoder(resp.Body).Decode(&repost); err != nil {
		return err, nil, nil
	}

	return nil, thread, &repost.URI
}

func LikePost(pds string, token string, id string, my_did string) (error, *ThreadRoot) {
	url := pds + "/xrpc/com.atproto.repo.createRecord"

	err, thread := GetPost(pds, token, id, 0, 1)
	if err != nil {
		return errors.New("failed to fetch post"), nil
	}

	payload := CreateRecordPayload{
		Collection: "app.bsky.feed.like",
		Repo:       my_did,
		Record: PostInteractionRecord{
			Type:      "app.bsky.feed.like",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Subject: Subject{
				URI: thread.Thread.Post.URI,
				CID: thread.Thread.Post.CID,
			},
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return errors.New("failed to marshal payload"), nil
	}

	resp, err := SendRequest(&token, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to retweet: " + bodyString), nil
	}

	likeRes := CreateRecordResult{}
	if err := json.NewDecoder(resp.Body).Decode(&likeRes); err != nil {
		return err, nil
	}

	thread.Thread.Post.Viewer.Like = &strings.Split(likeRes.URI, "/app.bsky.feed.like/")[1]

	return nil, thread
}

func UnlikePost(pds string, token string, id string, my_did string) (error, *ThreadRoot) {
	url := pds + "/xrpc/com.atproto.repo.deleteRecord"

	err, thread := GetPost(pds, token, id, 0, 1)
	if err != nil {
		return errors.New("failed to fetch post"), nil
	}

	payload := DeleteRecordPayload{
		Collection: "app.bsky.feed.like",
		Repo:       my_did,
		RKey:       strings.Split(*thread.Thread.Post.Viewer.Like, "/app.bsky.feed.like/")[1],
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return errors.New("failed to marshal payload"), nil
	}

	resp, err := SendRequest(&token, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to retweet: " + bodyString), nil
	}

	likeRes := CreateRecordResult{}
	if err := json.NewDecoder(resp.Body).Decode(&likeRes); err != nil {
		return err, nil
	}

	thread.Thread.Post.Viewer.Like = &likeRes.URI // maybe?

	return nil, thread
}

func GetLikes(pds string, token string, uri string, limit int) (*Likes, error) {
	url := fmt.Sprintf(pds+"/xrpc/app.bsky.feed.getLikes?limit=%d&uri=%s", limit, uri)

	resp, err := SendRequest(&token, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// // Print the response body
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to fetch timeline")
	}

	likes := Likes{}
	if err := json.NewDecoder(resp.Body).Decode(&likes); err != nil {
		return nil, err
	}

	return &likes, nil
}

func GetRetweetAuthors(pds string, token string, uri string, limit int) (*RepostedBy, error) {
	url := fmt.Sprintf(pds+"/xrpc/app.bsky.feed.getRepostedBy?limit=%d&uri=%s", limit, uri)

	resp, err := SendRequest(&token, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// // Print the response body
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to fetch timeline")
	}

	retweetAuthors := RepostedBy{}
	if err := json.NewDecoder(resp.Body).Decode(&retweetAuthors); err != nil {
		return nil, err
	}

	return &retweetAuthors, nil
}

func UserSearch(pds string, token string, query string) ([]User, error) {
	url := pds + "/xrpc/app.bsky.actor.searchActors?q=" + url.QueryEscape(query)

	resp, err := SendRequest(&token, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// // Print the response body
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to fetch search results")
	}

	users := UserSearchResult{}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}
	return users.Actors, nil
}

// thank you https://docs.bsky.app/blog/create-post#replies
func GetReplyRefs(pds string, token string, parentURI string) (*ReplySubject, error) {
	// Get the parent post
	err, parentThread := GetPost(pds, token, parentURI, 0, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch parent post: %w", err)
	}

	// If parent has a reply reference, fetch the root post
	var rootURI string
	var rootCID string

	if parentThread.Thread.Post.Record.Reply != nil {
		// Get the root post
		rootURI = parentThread.Thread.Post.Record.Reply.Root.URI
		err, rootThread := GetPost(pds, token, rootURI, 0, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch root post: %w", err)
		}
		rootCID = rootThread.Thread.Post.CID
	} else {
		// If parent has no reply reference, it's a top-level post, so it's also the root
		rootURI = parentThread.Thread.Post.URI
		rootCID = parentThread.Thread.Post.CID
	}

	return &ReplySubject{
		Root: Subject{
			URI: rootURI,
			CID: rootCID,
		},
		Parent: Subject{
			URI: parentThread.Thread.Post.URI,
			CID: parentThread.Thread.Post.CID,
		},
	}, nil
}

func GetRecord(pds string, uri string) (*RecordResponse, error) {
	collection, repo, rkey := GetURIComponents(uri)

	url := pds + "/xrpc/com.atproto.repo.getRecord?collection=" + collection + "&repo=" + repo + "&rkey=" + rkey

	resp, err := SendRequest(nil, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// // Print the response body
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to fetch record")
	}

	record := RecordResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, err
	}

	return &record, nil
}

// Gets the URI components
//
// @param uri: The URI to ge split
// @return: collection, repo, rkey
func GetURIComponents(uri string) (string, string, string) {
	uriSplit := strings.Split(uri, "/")
	// Example URI
	// at://did:plc:khcyntihpu7snjszuojjgjc4/app.bsky.feed.repost/3lcq7ddjinu2h
	return uriSplit[3], uriSplit[2], uriSplit[4]
}
