package twitterv1

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/Preloading/TwitterAPIBridge/cryption"
	"github.com/Preloading/TwitterAPIBridge/db_controller"
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

		db_controller.StoreAnalyticData(db_controller.AnalyticData{
			DataType:             "auth",
			IPAddress:            c.IP(),
			UserAgent:            c.Get("User-Agent"),
			Language:             c.Get("Accept-Language"),
			TwitterClient:        c.Get("X-Twitter-Client"),
			TwitterClientVersion: c.Get("X-Twitter-Client-Version"),
			Timestamp:            time.Now(),
		})

		return c.SendString(fmt.Sprintf("oauth_token=%s&oauth_token_secret=%s&user_id=%s&screen_name=%s&x_auth_expires=0", oauth_token, oauth_token, fmt.Sprintf("%d", bridge.BlueSkyToTwitterID(res.DID)), url.QueryEscape(authUsername)))
	} else if authMode == "exchange_auth" {
		return c.SendStatus(200) // uuuuuuuuh idk what this responds. I'll figure it out later.
	}
	// We have an unknown request. huh. Probably registration, i'll find a way to send an error msg for that later, as registration is out of scope.
	return c.SendStatus(501)
}

func VerifyCredentials(c *fiber.Ctx) error {
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	userinfo, err := blueskyapi.GetUserInfo(*pds, *oauthToken, *my_did, false)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	return EncodeAndSend(c, userinfo)
}

// GetAuthFromReq is a helper function to get the user DID and access token from the request.
// Also does some maintenance tasks like refreshing the access token if it has expired.
//
// @return: userDID, pds, tokenUUID, accessJwt, error
func GetAuthFromReq(c *fiber.Ctx) (*string, *string, *string, *string, error) {
	authHeader := c.Get("Authorization")
	fallbackRoute := "https://public.api.bsky.app"
	if configData.DeveloperMode {
		fmt.Println("Auth Header:", authHeader)
	}
	var accessJwt, refreshJwt, userPDS, basicHashSalt, basicAuthSalt, basicUUID *string
	var userDID, tokenUUID, encryptionKey, basicAuthUsernamePassword, authPassword string
	var access_expiry, refresh_expiry *float64
	var err error

	isBasic := false
	var username string

	// Define a regular expression to match the oauth_token
	if strings.HasPrefix(authHeader, "Basic ") {
		// This really should be rewritten. If you can, send a PR :)
		isBasic = true
		// This is using basic authentication. Basic authentication, sucks. We have to somehow store the password, and i do not like that.
		// But if we want iOS 2, we have to do this.
		base64pass := strings.TrimPrefix(authHeader, "Basic ")
		var did *string
		basicAuthUsernamePassword, err = cryption.Base64URLDecode(base64pass)
		if err != nil {
			return nil, &fallbackRoute, nil, nil, err
		}

		// separate the username and password
		username = strings.Split(basicAuthUsernamePassword, ":")[0]
		authPassword = strings.Split(basicAuthUsernamePassword, ":")[1]

		accessJwt, refreshJwt, access_expiry, refresh_expiry, userPDS, did, basicHashSalt, basicAuthSalt, basicUUID, err = db_controller.GetTokenViaBasic(username, authPassword)
		fmt.Println(err)
		if err != nil {
			// We might just not be signed in.
			if err.Error() == "invalid credentials" {
				// test if password is an app password thru regex
				if !regexp.MustCompile(`^[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}$`).MatchString(authPassword) {
					return nil, &fallbackRoute, nil, nil, errors.New("invalid app password")
				}

				res, pds, err := blueskyapi.Authenticate(username, authPassword)
				if err != nil {
					return nil, &fallbackRoute, nil, nil, err
				}

				access_token_expiry, err := cryption.GetJWTTokenExpirationUnix(res.AccessJwt)
				if err != nil {
					return nil, &fallbackRoute, nil, nil, errors.New("failed to get access token expiry")
				}
				refresh_token_expiry, err := cryption.GetJWTTokenExpirationUnix(res.RefreshJwt)
				if err != nil {
					return nil, &fallbackRoute, nil, nil, errors.New("failed to get refresh token expiry")
				}

				basicUUID, err = db_controller.StoreTokenBasic(res.DID, *pds, res.AccessJwt, res.RefreshJwt, username, authPassword, *access_token_expiry, *refresh_token_expiry)

				if err != nil {
					return nil, &fallbackRoute, nil, nil, err
				}

				return &res.DID, pds, nil, &res.AccessJwt, nil // TODO: Maybe change the uuid to something for here?
			} else {
				return nil, &fallbackRoute, nil, nil, err
			}
		}

		userDID = *did
	} else {
		re := regexp.MustCompile(`oauth_token="([^"]+)"`)
		matches := re.FindStringSubmatch(authHeader)
		if len(matches) < 2 {
			return nil, &fallbackRoute, nil, nil, errors.New("oauth token not found")
		}

		oauthToken := matches[1]
		oauthTokenSegments := strings.Split(oauthToken, ".")

		// Replace URL-friendly characters with original base64 characters

		// Check that we have at least 3 segments
		if len(oauthTokenSegments) != 3 {
			return nil, &fallbackRoute, nil, nil, errors.New("invalid oauth token")
		}

		// Get user DID
		userDID, err = cryption.Base64URLDecode(oauthTokenSegments[0])

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
		accessJwt, refreshJwt, access_expiry, refresh_expiry, userPDS, err = db_controller.GetToken(string(userDID), string(tokenUUID), encryptionKey)

		if err != nil {
			return nil, &fallbackRoute, nil, nil, err
		}
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
			// Delete this entry from the database
			if isBasic {
				db_controller.DeleteTokenViaBasic(username, authPassword)
			} else {
				db_controller.DeleteToken(string(userDID), string(tokenUUID))
			}
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
		if isBasic {
			db_controller.UpdateTokenBasic(userDID, *userPDS, new_auth.AccessJwt, new_auth.RefreshJwt, *access_token_expiry, *refresh_token_expiry, username, authPassword, *basicHashSalt, *basicAuthSalt, *basicUUID)
		} else {
			db_controller.UpdateToken(string(tokenUUID), string(userDID), *userPDS, new_auth.AccessJwt, new_auth.RefreshJwt, encryptionKey, *access_token_expiry, *refresh_token_expiry)
		}
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
