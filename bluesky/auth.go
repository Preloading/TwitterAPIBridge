package blueskyapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
)

type DIDDoc struct {
	// yes i know this isn't all of the fields, but it's good enuff
	Service []struct {
		ID              string `json:"id"`
		Type            string `json:"type"`
		ServiceEndpoint string `json:"serviceEndpoint"`
	} `json:"service"`
}

func Authenticate(username, password string) (*AuthResponse, *string, error) {
	_, userPDS, err := GetUserAuthData(username)
	if err != nil {
		return nil, nil, err
	}

	url := *userPDS + "/xrpc/com.atproto.server.createSession"

	authReq := AuthRequest{
		Identifier: username,
		Password:   password,
	}

	reqBody, err := json.Marshal(authReq)
	if err != nil {
		return nil, nil, err
	}

	resp, err := SendRequest(nil, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, nil, errors.New("authentication failed")
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, nil, err
	}

	return &authResp, userPDS, nil
}

func RefreshToken(pds string, refreshToken string) (*AuthResponse, error) {
	url := pds + "/xrpc/com.atproto.server.refreshSession"

	resp, err := SendRequest(&refreshToken, http.MethodPost, url, nil)
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

// This function is to get: the user DID and the user's PDS this should **ONLY** be used during authentication.
//
// @results: userDID, userPDS, error
func GetUserAuthData(handle string) (*string, *string, error) {
	// thank you https://discord.com/channels/1097580399187738645/1097580399187738648/1318477650485973004 (ducky.ws) on https://discord.gg/zYvmrHAr8M for explaining this to me

	// Validate our handle
	if !regexp.MustCompile(`^([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$`).MatchString(handle) {
		return nil, nil, errors.New("invalid handle")
	}
	userDID := ""

	// Get the handle's DID

	// Get DID thru .well-known, since this is what the most common handle PDS, bsky.social uses.
	wellKnownDIDResp, err := http.Get(fmt.Sprintf("https://%s/.well-known/atproto-did", handle))
	if err == nil {
		bodyBytes, _ := io.ReadAll(wellKnownDIDResp.Body)
		bodyString := string(bodyBytes)

		// Remove newline charectors, since it likes to fail the regex with them
		bodyString = strings.ReplaceAll(bodyString, "\n", "")
		// Check if the body is a DID
		if regexp.MustCompile(`^did:[a-z]+:[a-zA-Z0-9._:%-]*[a-zA-Z0-9._-]$`).MatchString(bodyString) {
			userDID = bodyString
		}
		wellKnownDIDResp.Body.Close()
	}
	if userDID == "" {
		// Get DID through _atproto DNS records
		txts, err := net.LookupTXT(fmt.Sprintf("_atproto.%s", handle))
		if err == nil {
			for _, txt := range txts {
				txt = strings.ReplaceAll(txt, "\n", "")
				if regexp.MustCompile(`^did=did:[a-z]+:[a-zA-Z0-9._:%-]*[a-zA-Z0-9._-]$`).MatchString(txt) {
					userDID = txt[4:] // Extract the DID without the 'did=' prefix
					break
				}
			}
		}
	}

	if userDID == "" {
		return nil, nil, errors.New("user does not exist")
	}

	// Get the user's PDS

	// we must do different things depending on the DID type.
	didDocReqUrl := ""
	switch strings.Split(userDID, ":")[1] {
	case "plc":
		// https://plc.directory/did:plc:<id>
		didDocReqUrl = fmt.Sprintf("https://plc.directory/%s", userDID)
	case "web":
		didDocReqUrl = fmt.Sprintf("https://%s/.well-known/did.json", strings.Split(userDID, ":")[2])
	}

	// get the DID doc
	didDocReq, err := http.Get(didDocReqUrl)
	if err != nil {
		return nil, nil, errors.New("could not find PDS")
	}
	bodyBytes, err := io.ReadAll(didDocReq.Body)
	if err != nil {
		return nil, nil, err
	}
	var userDIDDoc DIDDoc
	err = json.Unmarshal(bodyBytes, &userDIDDoc)
	didDocReq.Body.Close()
	if err != nil {
		return nil, nil, errors.New("could not find PDS")
	}

	// get the user's PDS
	userPDS := ""
	for _, service := range userDIDDoc.Service {
		if service.ID == "#atproto_pds" {
			userPDS = service.ServiceEndpoint
			break
		}
	}
	if userPDS == "" {
		return nil, nil, errors.New("could not find PDS")
	}

	// and finally, return our data
	return &userDID, &userPDS, nil
}
