package twitterv1

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/Preloading/TwitterAPIBridge/bridge"
	"github.com/Preloading/TwitterAPIBridge/cryption"
	"github.com/Preloading/TwitterAPIBridge/db_controller"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, bridge.AuthToken{
			Version:          2,
			Platform:         "bluesky",
			DID:              res.DID,
			CryptoKey:        encryptionkey,
			TokenUUID:        *uuid,
			ServerIdentifier: configData.ServerIdentifier,
			ServerURLs:       configData.ServerURLs,
			RegisteredClaims: &jwt.RegisteredClaims{
				// No ExpiresAt field means the token never expires
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				NotBefore: jwt.NewNumericDate(time.Now()),
				Issuer:    configData.ServerIdentifier,
				ExpiresAt: jwt.NewNumericDate(time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)), // it dies without this. i guess it's also nice to have ig
			},
		})

		oauth_token, err := token.SignedString(configData.SecretKeyBytes)
		if err != nil {
			return c.SendStatus(500)
		}

		db_controller.StoreAnalyticData(db_controller.AnalyticData{
			DataType:             "auth",
			IPAddress:            c.IP(),
			UserAgent:            c.Get("User-Agent"),
			Language:             c.Get("Accept-Language"),
			TwitterClient:        c.Get("X-Twitter-Client"),
			TwitterClientVersion: c.Get("X-Twitter-Client-Version"),
			Timestamp:            time.Now(),
		})

		return c.SendString(fmt.Sprintf("oauth_token=%s&oauth_token_secret=%s&user_id=%s&screen_name=%s&x_auth_expires=0", oauth_token, oauth_token, fmt.Sprintf("%d", bridge.BlueSkyToTwitterID(res.DID)), url.QueryEscape(authUsername))) // TODO: make this the actual screenname
	} else if authMode == "exchange_auth" {
		return c.Status(fiber.StatusUnauthorized).SendString("i have no idea what this should respond with, but it works if i don't have it implemented, so thats what im doing. If you do know what this does, lmk! <3")
		// this is a hack
		// auth_header := "oauth_token=\"" + c.FormValue("x_auth_access_secret") + "\""
		// c.Request().Header.Set("Authorization", auth_header)
		// my_did, pds, _, oauth_token, err := GetAuthFromReq(c)
		// if err != nil {
		// 	return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		// }
		// authenticating_user, err := blueskyapi.GetUserInfo(*pds, *oauth_token, *my_did, true)
		// if err != nil {
		// 	return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
		// }

		// return c.SendString(fmt.Sprintf("oauth_token=%s&oauth_token_secret=%s&user_id=%s&screen_name=%s&x_auth_expires=0", *oauth_token, *oauth_token, fmt.Sprintf("%d", bridge.BlueSkyToTwitterID(*my_did)), url.QueryEscape(authenticating_user.ScreenName)))
	}
	// We have an unknown request. huh. Probably registration, i'll find a way to send an error msg for that later, as registration is out of scope.
	return c.SendStatus(501)
}

func VerifyCredentials(c *fiber.Ctx) error {
	my_did, pds, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return MissingAuth(c)
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

				_, err = db_controller.StoreTokenBasic(res.DID, *pds, res.AccessJwt, res.RefreshJwt, username, authPassword, *access_token_expiry, *refresh_token_expiry)

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

		tokenData := &bridge.AuthToken{}
		tokenType := CheckTokenType(oauthToken)

		if tokenType == 1 && configData.MinTokenVersion == 1 {
			tokenData, err = ConvertV1TokenToV2(oauthToken)

			if err != nil {
				return nil, &fallbackRoute, nil, nil, err
			}

		} else {
			token, err := jwt.ParseWithClaims(oauthToken, tokenData, func(token *jwt.Token) (interface{}, error) {
				return configData.SecretKeyBytes, nil
			})

			if err != nil {
				return nil, &fallbackRoute, nil, nil, errors.New("invalid token")
			}

			if !token.Valid {
				if tokenData.ServerIdentifier == "" {
					return nil, &fallbackRoute, nil, nil, errors.New("invalid token")
				} else if tokenData.ServerIdentifier != configData.ServerIdentifier {
					return nil, &fallbackRoute, nil, nil, errors.New("incorrect server")

				}

				return nil, &fallbackRoute, nil, nil, errors.New("invalid token")
			}
		}

		// move all the token data into respective vars (aka technical debt)
		userDID = tokenData.DID
		tokenUUID = tokenData.TokenUUID
		encryptionKey = tokenData.CryptoKey

		// Fix the encryption key
		encryptionKey = strings.ReplaceAll(encryptionKey, "-", "+") + "="
		encryptionKey = strings.ReplaceAll(encryptionKey, "_", "/")

		// Now onto getting the access token from the database.
		accessJwt, refreshJwt, access_expiry, refresh_expiry, userPDS, err = db_controller.GetToken(string(userDID), string(tokenUUID), encryptionKey, tokenType) // Use token version 2 for OAuth

		if err != nil {
			return nil, &fallbackRoute, nil, nil, err
		}
	}

	if configData.DeveloperMode {
		fmt.Println("Access Token", *accessJwt)
	}

	// Check if the access token has expired
	if time.Unix(int64(*access_expiry), 0).Before(time.Now()) {
		// Get a lock before attempting refresh
		userLock := GetLock(tokenUUID)
		userLock.Lock()
		defer userLock.Unlock()

		// Check if we were locked
		if _, _, currentAccessExpiry, _, _, err := db_controller.GetToken(string(userDID), string(tokenUUID), encryptionKey, 2); err == nil && *currentAccessExpiry != *access_expiry {
			// Token data has changed while we were waiting for the lock, let's recall this
			return GetAuthFromReq(c)
		}

		if !time.Unix(int64(*access_expiry), 0).Before(time.Now()) {
			// Token was refreshed by another request while we were waiting
			userDIDStr := string(userDID)
			return &userDIDStr, userPDS, &tokenUUID, accessJwt, nil
		}

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
			db_controller.UpdateToken(string(tokenUUID), string(userDID), *userPDS, new_auth.AccessJwt, new_auth.RefreshJwt, encryptionKey, *access_token_expiry, *refresh_token_expiry, 2)
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

// Checks the token is V1
// Returns the token type
// 1 = V1
// 2 = V2 or unknown
func CheckTokenType(token string) int {
	// Check if this is a V1 token instead of a V2 (JWT)
	// We can do this by checking if the second segment is a base64 UUID
	splitToken := strings.Split(token, ".")
	if len(splitToken) == 3 {
		possibleUUID, err := cryption.Base64URLDecode(splitToken[1])

		if err != nil {
			return 2
		}

		// Check if possibleUUID is a UUID v4
		_, err = uuid.Parse(possibleUUID)
		if err == nil {
			return 1
		}
	}
	return 2
}

// This function converts the legacy V1 token format into a valid V2 token.
// This is used for backwards compatibility with the old V1 tokens.
func ConvertV1TokenToV2(token string) (*bridge.AuthToken, error) {
	// V1 tokens aren't very secure (they can be tampered with easily)

	// Split the token into its components
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token")
	}

	// Check that we have at least 3 segments
	if len(parts) != 3 {
		return nil, errors.New("invalid token")
	}

	// Get user DID
	userDID, err := cryption.Base64URLDecode(parts[0])

	if err != nil {
		return nil, errors.New("invalid token")
	}

	// Get our token UUID. This is used to look up the token in the database.
	tokenUUID, err := cryption.Base64URLDecode(parts[1])

	if err != nil {
		return nil, errors.New("invalid token")
	}

	return &bridge.AuthToken{
		Version:          1,
		Platform:         "bluesky",
		DID:              userDID,
		CryptoKey:        parts[2],
		TokenUUID:        tokenUUID,
		ServerIdentifier: configData.ServerIdentifier,
		ServerURLs:       configData.ServerURLs,
	}, nil
}

// Some stuff to avoid refreshing race conditions

type TokenLockManager struct {
	locks sync.Map
}

type lockInfo struct {
	mutex      *sync.Mutex
	lastAccess time.Time
}

var (
	manager         = &TokenLockManager{}
	cleanupInterval = 5 * time.Minute
)

func init() {
	go cleanup()
}

func cleanup() {
	for {
		time.Sleep(cleanupInterval)
		now := time.Now()
		manager.locks.Range(func(key, value interface{}) bool {
			if lock, ok := value.(*lockInfo); ok {
				if now.Sub(lock.lastAccess) > cleanupInterval {
					manager.locks.Delete(key)
				}
			}
			return true
		})
	}
}

func GetLock(userDID string) *sync.Mutex {
	lock, _ := manager.locks.LoadOrStore(userDID, &lockInfo{
		mutex:      &sync.Mutex{},
		lastAccess: time.Now(),
	})
	lockData := lock.(*lockInfo)
	lockData.lastAccess = time.Now()
	return lockData.mutex
}
