package blueskyapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type AuthResponse struct {
	OAuthToken       string `json:"oauth_token"`
	OAuthTokenSecret string `json:"oauth_token_secret"`
	UserID           string `json:"user_id"`
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

// func GetUserInfo(token string) error {
// 	url := "https://bsky.social/xrpc/com.atproto.server.getUserInfo"

// 	req, err := http.NewRequest(http.MethodGet, url, nil)
// 	if err != nil {
// 		return err
// 	}
// }
