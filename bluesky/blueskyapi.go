package blueskyapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Preloading/MastodonTwitterAPI/bridge"
)

type AuthResponse struct {
	AccessJwt  string `json:"accessJwt"`
	RefreshJwt string `json:"refreshJwt"`
	DID        string `json:"did"`
}

type User struct {
	DID            string `json:"did"`
	Handle         string `json:"handle"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	Avatar         string `json:"avatar"`
	Banner         string `json:"banner"`
	FollowersCount int    `json:"followersCount"`
	FollowsCount   int    `json:"followsCount"`
	PostsCount     int    `json:"postsCount"`
	IndexedAt      string `json:"indexedAt"`
	CreatedAt      string `json:"createdAt"`
}
type AuthRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type Author struct {
	DID         string `json:"did"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
	Associated  struct {
		Lists        int  `json:"lists"`
		FeedGens     int  `json:"feedgens"`
		StarterPacks int  `json:"starterPacks"`
		Labeler      bool `json:"labeler"`
		//chat
		CreatedAt time.Time `json:"created_at"`
		//viewer
	}
}

type PostRecord struct {
	Type      string    `json:"$type"`
	CreatedAt time.Time `json:"createdAt"`
	Embed     Embed     `json:"embed"`
	Facets    []Facet   `json:"facets"`
	Langs     []string  `json:"langs"`
	Text      string    `json:"text"`
}

// Specifically for reposts
type PostReason struct {
	CreatedAt time.Time `json:"createdAt"`
	Type      string    `json:"$type"`
	By        Author    `json:"by"`
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
	Did  string `json:"did"`
}

type Index struct {
	ByteEnd   int `json:"byteEnd"`
	ByteStart int `json:"byteStart"`
}

type PostViewer struct {
	Repost            *string `json:"repost"`
	Like              bool    `json:"like"`
	Muted             bool    `json:"muted"`
	BlockedBy         bool    `json:"blockedBy"`
	ThreadMute        bool    `json:"threadMute"`
	ReplyDisabled     bool    `json:"replyDisabled"`
	EmbeddingDisabled bool    `json:"embeddingDisabled"`
	Pinned            bool    `json:"pinned"`
}
type Post struct {
	URI    string     `json:"uri"`
	CID    string     `json:"cid"`
	Author Author     `json:"author"`
	Record PostRecord `json:"record"`
	// Embed  Embed      `json:"embed"`
	ReplyCount  int        `json:"replyCount"`
	RepostCount int        `json:"repostCount"`
	LikeCount   int        `json:"likeCount"`
	QuoteCount  int        `json:"quoteCount"`
	IndexedAt   string     `json:"indexedAt"`
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
	Type    string `json:"$type"`
	Post    Post   `json:"post"`
	Parent  Post   `json:"parent"`
	Replies []Post `json:"replies"`
}

// This is solely for the purpose of unmarshalling the response from the API
type ThreadRoot struct {
	Thread Thread `json:"thread"`
}

// Reposting/Retweeting
type RepostPayload struct {
	Collection string       `json:"collection"`
	Repo       string       `json:"repo"`
	Record     RepostRecord `json:"record"`
}

type RepostRecord struct {
	Type      string        `json:"$type"`
	CreatedAt string        `json:"createdAt"`
	Subject   RepostSubject `json:"subject"`
}

type RepostSubject struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

type Commit struct {
	CID string `json:"cid"`
	Rev string `json:"rev"`
}

type Repost struct {
	URI              string `json:"uri"`
	CID              string `json:"cid"`
	Commit           Commit `json:"commit"`
	ValidationStatus string `json:"validationStatus"`
}

func Authenticate(username, password string) (*AuthResponse, error) {
	url := "https://bsky.social/xrpc/com.atproto.server.createSession"

	authReq := AuthRequest{
		Identifier: username,
		Password:   password,
	}

	reqBody, err := json.Marshal(authReq)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("authentication failed")
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	return &authResp, nil
}

// TODO: This looks like it's a bsky.social specific endpoint, can we get the user's server?
func RefreshToken(refreshToken string) (*AuthResponse, error) {
	url := "https://bsky.social/xrpc/com.atproto.server.refreshSession"

	client := &http.Client{}

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+refreshToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("reauth failed")
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	return &authResp, nil
}

func GetUserInfo(token string, screen_name string) (*bridge.TwitterUserWithStatus, error) {
	url := "https://public.api.bsky.app/xrpc/app.bsky.actor.getProfile" + "?actor=" + screen_name

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
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

	user := User{}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &bridge.TwitterUserWithStatus{
		TwitterUser: bridge.TwitterUser{
			ProfileSidebarFillColor:   "e0ff92",
			Name:                      user.DisplayName,
			ProfileSidebarBorderColor: "87bc44",
			ProfileBackgroundTile:     false,
			CreatedAt:                 user.CreatedAt,
			ProfileImageURL:           user.Avatar,
			Location:                  "",
			ProfileLinkColor:          "0000ff",
			IsTranslator:              false,
			ContributorsEnabled:       false,
			URL:                       "",
			FavouritesCount:           0,
			UtcOffset:                 nil,
			ID:                        *bridge.BlueSkyToTwitterID(user.DID),
			// IDStr:                     bridge.BlueSkyToTwitterID(user.DID).String(),
			ProfileUseBackgroundImage: false,
			ListedCount:               0,
			ProfileTextColor:          "000000",
			Protected:                 false,
			FollowersCount:            user.FollowersCount,
			Lang:                      "en",
			Notifications:             nil,
			Verified:                  false,
			ProfileBackgroundColor:    "c0deed",
			GeoEnabled:                false,
			Description:               user.Description,
			FriendsCount:              user.FollowsCount,
			StatusesCount:             user.PostsCount,
			ScreenName:                user.Handle,
		},
	}, nil
}

// https://docs.bsky.app/docs/api/app-bsky-feed-get-feed
func GetTimeline(token string) (error, *Timeline) {
	url := "https://public.bsky.social/xrpc/app.bsky.feed.getTimeline"

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err, nil
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
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

func GetPost(token string, uri string, depth int, parentHeight int) (error, *ThreadRoot) {
	// Example URL at://did:plc:dqibjxtqfn6hydazpetzr2w4/app.bsky.feed.post/3lchbospvbc2j

	url := "https://public.bsky.social/xrpc/app.bsky.feed.getPostThread?depth=" + fmt.Sprintf("%d", depth) + "&parentHeight=" + fmt.Sprintf("%d", parentHeight) + "&uri=" + uri

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err, nil
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
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

func UpdateStatus(token string, status string) error {
	url := "https://public.bsky.social/xrpc/com.atproto.repo.createRecord"

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil
	}
	return errors.New("failed to update status")
}

func ReTweet(token string, id string, my_did string) (error, *ThreadRoot, *string) {
	url := "https://bsky.social/xrpc/com.atproto.repo.createRecord"

	err, thread := GetPost(token, id, 0, 1)

	if err != nil {
		return errors.New("failed to fetch post"), nil, nil
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err, nil, nil
	}
	payload := RepostPayload{
		Collection: "app.bsky.feed.repost",
		Repo:       my_did,
		Record: RepostRecord{
			Type:      "app.bsky.feed.repost",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Subject: RepostSubject{
				URI: thread.Thread.Post.URI,
				CID: thread.Thread.Post.CID,
			},
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return errors.New("failed to marshal payload"), nil, nil
	}

	req.Body = io.NopCloser(bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// if it works, we should get something like:
	// {"uri":"at://did:plc:khcyntihpu7snjszuojjgjc4/app.bsky.feed.repost/3lcm7b2pjio22","cid":"bafyreidw2uvnhns5bacdii7gozrou4rg25cpcxhe6cbhfws2c5hpsvycdm","commit":{"cid":"bafyreicu7db6k3vxbvtwiumggynbps7cuozsofbvo3kq7lz723smvpxne4","rev":"3lcm7b2ptb622"},"validationStatus":"valid"}
	resp, err := client.Do(req)
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

	repost := Repost{}
	if err := json.NewDecoder(resp.Body).Decode(&repost); err != nil {
		return err, nil, nil
	}

	return nil, thread, &repost.URI
}
