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
	Repost            bool `json:"repost"`
	Like              bool `json:"like"`
	ThreadMute        bool `json:"threadMute"`
	ReplyDisabled     bool `json:"replyDisabled"`
	EmbeddingDisabled bool `json:"embeddingDisabled"`
	Pinned            bool `json:"pinned"`
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
	FeedContext string `json:"feedContext"`
}

type Timeline struct {
	Feed   []Feed `json:"feed"`
	Cursor string `json:"cursor"`
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

func GetUserInfo(token string, screen_name string) (*bridge.TwitterUser, error) {
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

	return &bridge.TwitterUser{
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
		UtcOffset:                 0,
		ID:                        *bridge.BlueSkyToTwitterID(user.DID),
		ProfileUseBackgroundImage: false,
		ListedCount:               0,
		ProfileTextColor:          "000000",
		Protected:                 false,
		FollowersCount:            user.FollowersCount,
		Lang:                      "en",
		Notifications:             false,
		Verified:                  false,
		ProfileBackgroundColor:    "c0deed",
		GeoEnabled:                false,
		Description:               user.Description,
		FriendsCount:              user.FollowsCount,
		StatusesCount:             user.PostsCount,
		ScreenName:                user.Handle,
	}, nil
}

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

	feeds := Timeline{}
	if err := json.NewDecoder(resp.Body).Decode(&feeds); err != nil {
		return err, nil
	}

	return nil, &feeds
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
