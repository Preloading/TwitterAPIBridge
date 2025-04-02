package blueskyapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/Preloading/TwitterAPIBridge/config"
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
	Embed     Embed         `json:"embed,omitempty"`
	Facets    []Facet       `json:"facets,omitempty"`
	Langs     []string      `json:"langs,omitempty"`
	Text      string        `json:"text,omitempty"`
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
	Images []Image `json:"images,omitempty"`
	Video  `json:",omitempty"`
}

type Image struct {
	Alt         string      `json:"alt"`
	AspectRatio AspectRatio `json:"aspectRatio"`
	Image       Blob        `json:"image"`
}

type Video struct {
	Alt         string      `json:"alt"`
	AspectRatio AspectRatio `json:"aspectRatio"`
	Video       *Blob       `json:"video"`
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

type FollowersTimeline struct {
	Subject   User   `json:"subject"`
	Followers []User `json:"followers"`
	Cursor    string `json:"cursor"`
}

type FollowsTimeline struct {
	Subject   User   `json:"subject"`
	Followers []User `json:"follows"`
	Cursor    string `json:"cursor"`
}

type Thread struct {
	Type    string    `json:"$type"`
	Post    Post      `json:"post"`
	Parent  *Thread   `json:"parent"`
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

type UpdateRecordPayload struct {
	CreateRecordPayload
	RKey       string `json:"rkey"`
	SwapRecord string `json:"swapRecord"`
}

type DeleteRecordPayload struct {
	Collection string `json:"collection"`
	Repo       string `json:"repo"`
	RKey       string `json:"rkey"`
}

type PostInteractionRecord struct {
	Type      string      `json:"$type"`
	CreatedAt string      `json:"createdAt"`
	Subject   interface{} `json:"subject"`
}

type CreatePostRecord struct {
	Type      string        `json:"$type"`
	Text      string        `json:"text"`
	CreatedAt time.Time     `json:"createdAt"`
	Reply     *ReplySubject `json:"reply,omitempty"`
	Facets    []Facet       `json:"facets,omitempty"`
	Embed     *Embed        `json:"embed,omitempty"`
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

type PostSearchResult struct {
	Posts     []Post `json:"posts"`
	HitsTotal int    `json:"hitsTotal"`
	Cursor    string `json:"cursor"`
}

type UserSearchResult struct {
	Actors []User `json:"actors"`
}

type OtherActorSuggestions struct {
	Actors []User `json:"suggestions"`
}

type RecordResponse struct {
	URI   string      `json:"uri"`
	CID   string      `json:"cid"`
	Value RecordValue `json:"value"`
}

type RecordValue struct { // TODO: Figure out how to make it get different types of records
	Reply       *ReplySubject `json:"reply,omitempty"`
	CreatedAt   time.Time     `json:"createdAt,omitempty"`
	Description string        `json:"description,omitempty"`
	DisplayName string        `json:"displayName,omitempty"`
	Avatar      Blob          `json:"avatar,omitempty"`
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

type TrendingTopics struct {
	Topics    []TrendingTopic `json:"topics"`
	Suggested []TrendingTopic `json:"suggested"`
}

type TrendingTopic struct {
	Topic string `json:"topic"`
	Link  string `json:"link"`
}

type Notification struct {
	URI           string                `json:"uri"`
	CID           string                `json:"cid"`
	Author        User                  `json:"author"`
	Reason        string                `json:"reason"`
	ReasonSubject string                `json:"reasonSubject"`
	Record        PostInteractionRecord `json:"record"` // i think this is the correct object?
	IsRead        bool                  `json:"isRead"`
	IndexedAt     time.Time             `json:"indexedAt"`
}

type Notifications struct {
	Notifications []Notification `json:"notifications"`
	Cursor        string         `json:"cursor"`
	Priority      bool           `json:"priority"`
	SeenAt        time.Time      `json:"seenAt"`
}

type ListInfo struct {
	URI     string `json:"uri"`
	CID     string `json:"cid"`
	Creator User   `json:"creator"`
	Name    string `json:"name"`
	// ignoring purpose

	Description       string     `json:"description"`
	DescriptionFacets []Facet    `json:"descriptionFacets"`
	Avatar            string     `json:"avatar"`
	ListItemCount     int        `json:"listItemCount"`
	IndexedAt         time.Time  `json:"indexedAt"`
	Viewer            PostViewer `json:"viewer"`
}

type ListItem struct {
	URI     string `json:"uri"`
	Subject User   `json:"subject"`
}

type Lists struct {
	Lists  []ListInfo `json:"lists"`
	Cursor string     `json:"cursor"`
}

type ListDetailed struct {
	List   ListInfo   `json:"list"`
	Cursor string     `json:"cursor"`
	Items  []ListItem `json:"items"`
}

var (
	configData *config.Config
)

func InitConfig(config *config.Config) {
	configData = config
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
	req.Header.Set("UserAgent", "ATwitterAPIBridge/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func SendRequestWithContentType(token *string, method string, url string, body io.Reader, content_type string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if token != nil {
		req.Header.Set("Authorization", "Bearer "+*token)
	}
	req.Header.Set("Content-Type", content_type) // 99% sure all bluesky requests are json.

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func GetUserInfo(pds string, token string, screen_name string, nocache bool) (*bridge.TwitterUser, error) {
	if !nocache {
		if user, found := userCache.Get(screen_name); found {
			return &user, nil
		}
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

func GetUserInfoRaw(pds string, token string, screen_name string) (*User, error) {
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

	return &author, nil
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

			url := pds + "/xrpc/app.bsky.actor.getProfiles" + "?actors=" + strings.Join(c, "&actors=")
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

// https://web.archive.org/web/20121029153120/https://dev.twitter.com/docs/platform-objects/users
func AuthorTTB(author User) *bridge.TwitterUser {
	id := bridge.BlueSkyToTwitterID(author.DID)
	pfp_url := configData.CdnURL + "/cdn/img/?url=" + url.QueryEscape(author.Avatar) + ":profile_bigger"
	banner_url := ""
	if author.Banner != "" {
		banner_components := strings.Split(author.Banner, "/")

		if len(banner_components) >= 8 {
			banner_component_blob := strings.Split(banner_components[7], "@")
			banner_url = configData.CdnURL + "/cdn/img/bsky/" + banner_components[6] + "/" + banner_component_blob[0] + ".png"
		}
	}
	user := &bridge.TwitterUser{
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
		ProfileImageURLHttps:      pfp_url,
		ProfileImageURL:           pfp_url,

		ProfileUseBackgroundImage: false,

		ProfileBannerURL:      banner_url,
		ProfileBannerURLHttps: banner_url,

		Location:            "",
		ProfileLinkColor:    "0000ff",
		IsTranslator:        false,
		ContributorsEnabled: false,
		URL:                 "",
		UtcOffset:           nil,
		ID:                  *id,
		IDStr:               strconv.FormatInt(*id, 10),
		ListedCount:         0,
		ProfileTextColor:    "000000",
		Protected:           false,

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
	return user
}

// https://docs.bsky.app/docs/api/app-bsky-feed-get-feed
func GetTimeline(pds string, token string, context string, feed string, limit int) (error, *Timeline) {
	url := pds + "/xrpc/app.bsky.feed.getTimeline?limit=" + fmt.Sprintf("%d", limit)
	if context != "" {
		url = pds + "/xrpc/app.bsky.feed.getTimeline?cursor=" + context + "&limit=" + fmt.Sprintf("%d", limit)
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
func GetUserTimeline(pds string, token string, context string, actor string, limit int) (error, *Timeline) {
	apiURL := pds + "/xrpc/app.bsky.feed.getAuthorFeed?actor=" + url.QueryEscape(actor) + "&limit=" + fmt.Sprintf("%d", limit)
	if context != "" {
		apiURL = pds + "/xrpc/app.bsky.feed.getAuthorFeed?actor=" + url.QueryEscape(actor) + "&cursor=" + context + "&limit=" + fmt.Sprintf("%d", limit)
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

func GetMediaTimeline(pds string, token string, context string, actor string, limit int) (error, *Timeline) {
	apiURL := pds + "/xrpc/app.bsky.feed.getAuthorFeed?actor=" + url.QueryEscape(actor) + "&limit=" + fmt.Sprintf("%d", limit) + "&filter=posts_with_media"
	if context != "" {
		apiURL = pds + "/xrpc/app.bsky.feed.getAuthorFeed?actor=" + url.QueryEscape(actor) + "&cursor=" + context + "&limit=" + fmt.Sprintf("%d", limit) + "&filter=posts_with_media"
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

// https://docs.bsky.app/docs/api/app-bsky-graph-get-list
func GetListTimeline(pds string, token string, context string, listURI string, limit int) (error, *Timeline) {
	apiURL := pds + "/xrpc/app.bsky.feed.getListFeed?list=" + url.QueryEscape(listURI) + "&limit=" + fmt.Sprintf("%d", limit)
	if context != "" {
		apiURL = pds + "/xrpc/app.bsky.feed.getListFeed?list=" + url.QueryEscape(listURI) + "&cursor=" + context + "&limit=" + fmt.Sprintf("%d", limit)
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

// https://docs.bsky.app/docs/api/app-bsky-feed-get-posts
func GetPosts(pds string, token string, items []string) ([]*Post, error) {
	var results []*Post

	// Parallel fetching for chunks of up to 25 at a time
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := 0; i < len(items); i += 25 {
		end := i + 25
		if end > len(items) {
			end = len(items)
		}
		chunk := items[i:end]

		wg.Add(1)
		go func(c []string) {
			defer wg.Done()

			url := pds + "/xrpc/app.bsky.feed.getPosts" + "?uris=" + strings.Join(c, "&uris=")
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

			var posts struct {
				Posts []Post `json:"posts"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
				return
			}

			mu.Lock()
			for _, post := range posts.Posts {
				results = append(results, &post)
			}
			mu.Unlock()
		}(chunk)
	}

	wg.Wait()
	return results, nil
}

// https://docs.bsky.app/docs/api/app-bsky-graph-get-followers
func GetFollowers(pds string, token string, context string, actor string) (*FollowersTimeline, error) {
	apiURL := pds + "/xrpc/app.bsky.graph.getFollowers?actor=" + url.QueryEscape(actor)
	if context != "" {
		apiURL = pds + "/xrpc/app.bsky.graph.getFollowers?actor=" + url.QueryEscape(actor) + "&cursor=" + context
	}

	resp, err := SendRequest(&token, http.MethodGet, apiURL, nil)
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
		return nil, errors.New("failed to fetch followers")
	}

	feeds := FollowersTimeline{}
	if err := json.NewDecoder(resp.Body).Decode(&feeds); err != nil {
		return nil, err
	}

	return &feeds, nil
}

// https://docs.bsky.app/docs/api/app-bsky-graph-get-follows
func GetFollows(pds string, token string, context string, actor string) (*FollowsTimeline, error) {
	apiURL := pds + "/xrpc/app.bsky.graph.getFollows?actor=" + url.QueryEscape(actor)
	if context != "" {
		apiURL = pds + "/xrpc/app.bsky.graph.getFollows?actor=" + url.QueryEscape(actor) + "&cursor=" + context
	}

	resp, err := SendRequest(&token, http.MethodGet, apiURL, nil)
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
		return nil, errors.New("failed to fetch followers")
	}

	feeds := FollowsTimeline{}
	if err := json.NewDecoder(resp.Body).Decode(&feeds); err != nil {
		return nil, err
	}

	return &feeds, nil
}

// This handles both normal & replys
func UpdateStatus(pds string, token string, my_did string, status string, in_reply_to *string, mentions []bridge.FacetParsing, urls []bridge.FacetParsing, tags []bridge.FacetParsing, imageBlob *Blob, imageRes []int) (*ThreadRoot, error) {
	url := pds + "/xrpc/com.atproto.repo.createRecord"

	var replySubject *ReplySubject
	var err error
	facets := []Facet{}
	embeds := Embed{}

	// find mention's DID
	handles := []string{}
	for _, mention := range mentions {
		handles = append(handles, mention.Item)
	}
	mentionedUsers, err := GetUsersInfo(pds, token, handles, false)

	// add mentions to the facets
	if err == nil {
		for _, mention := range mentions {
			var mentionDID *string
			for _, user := range mentionedUsers {
				if user.ScreenName == mention.Item {
					mentionDID, _ = bridge.TwitterIDToBlueSky(&user.ID) // efficency is poor
					break
				}
			}

			if mentionDID == nil {
				continue
			}

			facets = append(facets, Facet{
				Index: Index{
					ByteStart: mention.Start,
					ByteEnd:   mention.End,
				},
				Features: []Feature{
					{
						Type: "app.bsky.richtext.facet#mention",
						Did:  *mentionDID,
					},
				},
			})
		}
	}

	// add URLs to the facets
	for _, url := range urls {
		facets = append(facets, Facet{
			Index: Index{
				ByteStart: url.Start,
				ByteEnd:   url.End,
			},
			Features: []Feature{
				{
					Type: "app.bsky.richtext.facet#link",
					Uri:  url.Item,
				},
			},
		})
	}

	// add tags (#something) to the facets
	for _, tag := range tags {
		facets = append(facets, Facet{
			Index: Index{
				ByteStart: tag.Start,
				ByteEnd:   tag.End,
			},
			Features: []Feature{
				{
					Type: "app.bsky.richtext.facet#tag",
					Tag:  tag.Item,
				},
			},
		})
	}

	// Replying
	if in_reply_to != nil && *in_reply_to != "" {
		replySubject, err = GetReplyRefs(pds, token, *in_reply_to)
		if err != nil {
			return nil, errors.New("failed to fetch reply refs")
		}
	}

	// Images
	if imageBlob != nil {
		embeds = Embed{
			Type: "app.bsky.embed.images",
			Images: []Image{
				{
					Alt: "", // Twitter doesn't have alt text (poor accessibility)
					// AspectRatio: AspectRatio{Height: 1, Width: 1}, // lets see if it works without the aspect ratio ;)
					Image: *imageBlob,
					AspectRatio: AspectRatio{
						Height: imageRes[0],
						Width:  imageRes[1],
					},
				},
			},
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
			Facets:    facets,
			Embed: func() *Embed {
				if embeds.Type == "" {
					return nil
				}
				return &embeds
			}(),
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("failed to marshal payload")
	}
	fmt.Println(string(reqBody))
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
		return errors.New("failed to like: " + bodyString), nil
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
		return errors.New("failed to unlike: " + bodyString), nil
	}

	likeRes := CreateRecordResult{}
	if err := json.NewDecoder(resp.Body).Decode(&likeRes); err != nil {
		return err, nil
	}

	emptyString := ""
	thread.Thread.Post.Viewer.Like = &emptyString

	return nil, thread
}

func FollowUser(pds string, token string, targetActor string, my_did string) (error, *User) {
	url := pds + "/xrpc/com.atproto.repo.createRecord"

	targetUser, err := GetUserInfoRaw(pds, token, targetActor)
	if err != nil {
		return errors.New("failed to fetch post"), nil
	}

	if targetUser.Viewer.Following != nil {
		return errors.New("already following user"), nil
	}

	payload := CreateRecordPayload{
		Collection: "app.bsky.graph.follow",
		Repo:       my_did,
		Record: PostInteractionRecord{
			Type:      "app.bsky.graph.follow",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Subject:   targetUser.DID,
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

	followRes := CreateRecordResult{}
	if err := json.NewDecoder(resp.Body).Decode(&followRes); err != nil {
		return err, nil
	}

	targetUser.Viewer.Following = &strings.Split(followRes.URI, "/app.bsky.graph.follow/")[1]
	targetUser.FollowersCount++

	return nil, targetUser
}

func UnfollowUser(pds string, token string, targetActor string, my_did string) (error, *User) {
	url := pds + "/xrpc/com.atproto.repo.deleteRecord"

	targetUser, err := GetUserInfoRaw(pds, token, targetActor)
	if err != nil {
		return errors.New("failed to fetch post"), nil
	}

	if targetUser.Viewer.Following == nil {
		return errors.New("not following user"), nil
	}

	payload := DeleteRecordPayload{
		Collection: "app.bsky.graph.follow",
		Repo:       my_did,
		RKey:       strings.Split(*targetUser.Viewer.Following, "/app.bsky.graph.follow/")[1],
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
		return errors.New("failed to unfollow: " + bodyString), nil
	}

	unfollowRes := CreateRecordResult{}
	if err := json.NewDecoder(resp.Body).Decode(&unfollowRes); err != nil {
		return err, nil
	}

	emptyString := ""
	targetUser.Viewer.Following = &emptyString

	return nil, targetUser
}

func GetPostLikes(pds string, token string, uri string, limit int) (*Likes, error) {
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
		return nil, errors.New("failed to fetch likes")
	}

	likes := Likes{}
	if err := json.NewDecoder(resp.Body).Decode(&likes); err != nil {
		return nil, err
	}

	return &likes, nil
}

// https://docs.bsky.app/docs/api/app-bsky-feed-get-actor-likes
// Bluesky for SOME REASON limits viewing the likes to your own user. WHy?
// What is the point of having an "actor" field if you can only use 1 actor?
// I'm still gonna implement it, we can hope it will be expanded in the future.
func GetActorLikes(pds string, token string, context string, actor string, limit int) (error, *Timeline) {
	url := fmt.Sprintf(pds+"/xrpc/app.bsky.feed.getActorLikes?limit=%d&actor=%s", limit, actor)
	if context != "" {
		url = fmt.Sprintf(pds+"/xrpc/app.bsky.feed.getActorLikes?limit=%d&actor=%s&cursor=%s", limit, actor, context)
	}

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
		return errors.New("failed to fetch likes"), nil
	}

	likes := Timeline{}
	if err := json.NewDecoder(resp.Body).Decode(&likes); err != nil {
		return err, nil
	}

	return nil, &likes
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
		return nil, errors.New("failed to fetch retweet authors")
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

func PostSearch(pds string, token string, query string, since *time.Time, until *time.Time) ([]Post, error) {
	url := pds + "/xrpc/app.bsky.feed.searchPosts?sort=top&q=" + url.QueryEscape(query)
	if since != nil {
		url += "&since=" + since.Format(time.RFC3339)
	}
	if until != nil {
		url += "&until=" + until.Format(time.RFC3339)
	}

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

	posts := PostSearchResult{}
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}
	return posts.Posts, nil
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

func GetRecordWithUri(pds string, uri string) (*RecordResponse, error) {
	collection, repo, rkey := GetURIComponents(uri)
	return GetRecord(pds, collection, repo, rkey)
}

func GetRecord(pds string, collection string, repo string, rkey string) (*RecordResponse, error) {

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

func UpdateRecord(pds string, token string, collection string, repo string, rkey string, swapRecord string, newRecord interface{}) error {
	url := pds + "/xrpc/com.atproto.repo.putRecord"

	payload := UpdateRecordPayload{
		CreateRecordPayload: CreateRecordPayload{
			Collection: collection,
			Repo:       repo,
			Record:     newRecord,
		},
		RKey:       rkey,
		SwapRecord: swapRecord,
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
		return errors.New("failed to update: " + bodyString)
	}
	return nil
}

// This feature is still in beta, and is likely to break in the future
func GetTrends(pds string, token string) (*TrendingTopics, error) {
	url := pds + "/xrpc/app.bsky.unspecced.getTrendingTopics"

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
		return nil, errors.New("failed to fetch trends")
	}

	trends := TrendingTopics{}
	if err := json.NewDecoder(resp.Body).Decode(&trends); err != nil {
		return nil, err
	}

	return &trends, nil
}

func GetUsersLists(pds string, token string, actor string, limit int, cursor string) (*Lists, error) {
	url := fmt.Sprintf(pds+"/xrpc/app.bsky.graph.getLists?limit=%d&actor=%s", limit, actor)
	if cursor != "" {
		url = fmt.Sprintf(pds+"/xrpc/app.bsky.graph.getLists?limit=%d&actor=%s&cursor=%s", limit, actor, cursor)
	}

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
		return nil, errors.New("failed to fetch a user's lists")
	}

	lists := Lists{}
	if err := json.NewDecoder(resp.Body).Decode(&lists); err != nil {
		return nil, err
	}

	return &lists, nil
}

func GetList(pds string, token string, listURI string, limit int, cursor string) (*ListDetailed, error) {
	url := fmt.Sprintf(pds+"/xrpc/app.bsky.graph.getList?limit=%d&list=%s", limit, listURI)
	if cursor != "" {
		url = fmt.Sprintf(pds+"/xrpc/app.bsky.graph.getList?limit=%d&list=%s&cursor=%s", limit, listURI, cursor)
	}

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
		return nil, errors.New("failed to fetch a user's lists")
	}

	list := ListDetailed{}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}

	return &list, nil
}

func GetMySuggestedUsers(pds string, token string, limit int) ([]User, error) {
	url := pds + "/xrpc/app.bsky.actor.getSuggestions?limit=" + fmt.Sprintf("%d", limit)

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
		return nil, errors.New("failed to fetch suggested users")
	}

	users := UserSearchResult{}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}
	return users.Actors, nil
}

func GetOthersSuggestedUsers(pds string, token string, limit int, actor string) ([]User, error) {
	url := pds + "/xrpc/app.bsky.graph.getSuggestedFollowsByActor?limit=" + fmt.Sprintf("%d", limit) + "&actor=" + actor

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
		return nil, errors.New("failed to fetch suggested users")
	}

	users := OtherActorSuggestions{}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}
	return users.Actors, nil
}

func GetNotifications(pds string, token string, limit int, context string) (*Notifications, error) {
	url := pds + "/xrpc/app.bsky.notification.listNotifications?limit=" + fmt.Sprintf("%d", limit)
	if context != "" {
		url = pds + "/xrpc/app.bsky.notification.listNotifications?cursor=" + context + "&limit=" + fmt.Sprintf("%d", limit)
	}

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
		return nil, errors.New("failed to fetch notifications")
	}

	notifications := Notifications{}
	if err := json.NewDecoder(resp.Body).Decode(&notifications); err != nil {
		return nil, err
	}

	return &notifications, nil
}

func UploadBlob(pds string, token string, data []byte, content_type string) (*Blob, error) {
	url := pds + "/xrpc/com.atproto.repo.uploadBlob"

	resp, err := SendRequestWithContentType(&token, http.MethodPost, url, bytes.NewReader(data), content_type)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to upload blob")
	}

	blob := struct {
		Blob Blob `json:"blob"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&blob); err != nil {
		return nil, err
	}

	return &blob.Blob, nil
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
