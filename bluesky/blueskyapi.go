package blueskyapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

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
	url := "https://bsky.social/xrpc/app.bsky.actor.getProfile" + "?actor=" + screen_name

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

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
		ID:                        bridge.BlueSkyToTwitterID(user.DID),
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
