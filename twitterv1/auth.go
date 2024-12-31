package twitterv1

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/Preloading/MastodonTwitterAPI/cryption"
	"github.com/Preloading/MastodonTwitterAPI/db_controller"
	"github.com/gofiber/fiber/v2"
)

// https://developer.x.com/en/docs/authentication/api-reference/access_token
// and
// https://web.archive.org/web/20120708225149/https://dev.twitter.com/docs/oauth/xauth
func access_token(c *fiber.Ctx) error {
	// Parse the form data
	//sendErrorCodes := c.FormValue("send_error_codes")
	authMode := c.FormValue("x_auth_mode")
	authPassword := c.FormValue("x_auth_password")
	authUsername := c.FormValue("x_auth_username")

	if authMode == "client_auth" {
		res, pds, err := blueskyapi.Authenticate(authUsername, authPassword)
		if err != nil {
			fmt.Println("Error:", err)
			return c.SendStatus(401)
		}

		// Our bluesky authentication was sucessful! Now we should store the auth info, encryted, in the DB
		encryptionkey, err := cryption.GenerateKey()
		if err != nil {
			fmt.Println("Error:", err)
			return c.SendStatus(500)
		}

		access_token_expiry, err := cryption.GetJWTTokenExpirationUnix(res.AccessJwt)
		if err != nil {
			fmt.Println("Error:", err)
			return c.SendStatus(500)
		}
		refresh_token_expiry, err := cryption.GetJWTTokenExpirationUnix(res.RefreshJwt)
		if err != nil {
			fmt.Println("Error:", err)
			return c.SendStatus(500)
		}

		uuid, err := db_controller.StoreToken(res.DID, *pds, res.AccessJwt, res.RefreshJwt, encryptionkey, *access_token_expiry, *refresh_token_expiry)

		if err != nil {
			fmt.Println("Error:", err)
			return c.SendStatus(500)
		}
		encryptionkey = strings.ReplaceAll(encryptionkey, "+", "-")
		encryptionkey = strings.ReplaceAll(encryptionkey, "/", "_")
		encryptionkey = strings.ReplaceAll(encryptionkey, "=", "") // remove padding

		oauth_token := fmt.Sprintf("%s.%s.%s", cryption.Base64URLEncode(res.DID), cryption.Base64URLEncode(*uuid), encryptionkey)

		return c.SendString(fmt.Sprintf("oauth_token=%s&oauth_token_secret=%s&user_id=%s&screen_name=twitterapi&x_auth_expires=%f", oauth_token, oauth_token, fmt.Sprintf("%d", bridge.BlueSkyToTwitterID(res.DID)), *access_token_expiry))
	}
	// We have an unknown request. huh. Probably registration, i'll find a way to send an error msg for that later, as registration is out of scope.
	return c.SendStatus(501)
}

// https://web.archive.org/web/20120508075505/https://dev.twitter.com/docs/api/1/get/users/show
func VerifyCredentials(c *fiber.Ctx) error {
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	userinfo, err := blueskyapi.GetUserInfo(*pds, *oauthToken, *my_did, false)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	return c.JSON(userinfo)
}

// GetAuthFromReq is a helper function to get the user DID and access token from the request.
// Also does some maintenance tasks like refreshing the access token if it has expired.
//
// @return: userDID, pds, tokenUUID, accessJwt, error
func GetAuthFromReq(c *fiber.Ctx) (*string, *string, *string, *string, error) {
	authHeader := c.Get("Authorization")
	if configData.DeveloperMode {
		fmt.Println("Auth Header:", authHeader)
	}
	// Define a regular expression to match the oauth_token
	re := regexp.MustCompile(`oauth_token="([^"]+)"`)
	matches := re.FindStringSubmatch(authHeader)
	fallbackRoute := "https://public.api.bsky.app"

	if len(matches) < 2 {
		return nil, &fallbackRoute, nil, nil, errors.New("oauth token not found")
	}

	oauthToken := matches[1]
	oauthTokenSegments := strings.Split(oauthToken, ".")

	// Replace URL-friendly characters with original base64 characters

	// Get user DID
	userDID, err := cryption.Base64URLDecode(oauthTokenSegments[0])

	if err != nil {
		return nil, &fallbackRoute, nil, nil, err
	}

	// Get our token UUID. This is used to look up the token in the database.
	tokenUUID, err := cryption.Base64URLDecode(oauthTokenSegments[1])

	if err != nil {
		return nil, &fallbackRoute, nil, nil, err
	}

	// Get the encryption key for the data.
	encryptionKey := oauthTokenSegments[2] + "="
	encryptionKey = strings.ReplaceAll(encryptionKey, "-", "+")
	encryptionKey = strings.ReplaceAll(encryptionKey, "_", "/")

	// Now onto getting the access token from the database.
	accessJwt, refreshJwt, access_expiry, refresh_expiry, userPDS, err := db_controller.GetToken(string(userDID), string(tokenUUID), encryptionKey)

	if err != nil {
		return nil, &fallbackRoute, nil, nil, err
	}

	if configData.DeveloperMode {
		fmt.Println("Access Token", *accessJwt)
	}

	// Check if the access token has expired
	if time.Unix(int64(*access_expiry), 0).Before(time.Now()) {
		// Our access token has expired. We need to refresh it.

		// Lets check if our refresh token has expired
		if time.Unix(int64(*refresh_expiry), 0).Before(time.Now()) {
			// Our refresh token has expired. We need to re-authenticate.
			return nil, &fallbackRoute, nil, nil, errors.New("refresh token has expired")
		}

		// Our refresh token is still valid. Lets refresh our access token.
		new_auth, err := blueskyapi.RefreshToken(*userPDS, *refreshJwt)

		if err != nil {
			return nil, &fallbackRoute, nil, nil, err
		}

		accessJwt = &new_auth.AccessJwt

		access_token_expiry, err := cryption.GetJWTTokenExpirationUnix(new_auth.AccessJwt)
		if err != nil {
			return nil, &fallbackRoute, nil, nil, errors.New("failed to get access token expiry")
		}
		refresh_token_expiry, err := cryption.GetJWTTokenExpirationUnix(new_auth.RefreshJwt)
		if err != nil {
			return nil, &fallbackRoute, nil, nil, errors.New("failed to get refresh token expiry")
		}

		// TODO: Recheck if the user id is still bound to that PDS
		db_controller.UpdateToken(string(tokenUUID), string(userDID), *userPDS, new_auth.AccessJwt, new_auth.RefreshJwt, encryptionKey, *access_token_expiry, *refresh_token_expiry)
	}

	userDIDStr := string(userDID)
	return &userDIDStr, userPDS, &tokenUUID, accessJwt, nil
}

func GetEncryptionKeyFromRequest(c *fiber.Ctx) (*string, error) {
	authHeader := c.Get("Authorization")
	// Define a regular expression to match the oauth_token
	re := regexp.MustCompile(`oauth_token="([^"]+)"`)
	matches := re.FindStringSubmatch(authHeader)

	if len(matches) < 2 {
		return nil, errors.New("oauth token not found")
	}

	oauthToken := matches[1]
	oauthTokenSegments := strings.Split(oauthToken, ".")

	// Get the encryption key for the data.
	encryptionKey := oauthTokenSegments[2] + "="
	encryptionKey = strings.ReplaceAll(encryptionKey, "-", "+")
	encryptionKey = strings.ReplaceAll(encryptionKey, "_", "/")

	return &encryptionKey, nil
}
