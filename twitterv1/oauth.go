package twitterv1

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var (
	tempTokens          = sync.Map{}
	tempTokenExpiration = 10 * time.Minute
)

type OAuthParams struct {
	Callback        string
	ConsumerKey     string
	Nonce           string
	Signature       string
	SignatureMethod string
	Timestamp       string
	Version         string

	//used in final oauth thingy
	Verifier string
	Token    string
}

type TempToken struct {
	Token     string
	Secret    string
	AuthResp  *blueskyapi.AuthResponse
	AuthPDS   *string
	Verifier  string
	CreatedAt time.Time
	ExpiresIn time.Duration
	Callback  string
}

func cleanupTempTokens() {
	for {
		time.Sleep(tempTokenExpiration)
		now := time.Now()
		tempTokens.Range(func(key, value interface{}) bool {
			if token, ok := value.(TempToken); ok {
				if now.Sub(token.CreatedAt) > tempTokenExpiration {
					tempTokens.Delete(key)
				}
			}
			return true
		})
	}
}

func ParseOAuthHeader(header string) (*OAuthParams, error) {
	if !strings.HasPrefix(header, "OAuth ") {
		return nil, errors.New("invalid OAuth header format")
	}

	params := &OAuthParams{}
	header = strings.TrimPrefix(header, "OAuth ")
	pairs := strings.Split(header, ",")

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.Trim(strings.TrimSpace(kv[1]), "\"")
		value, _ = url.QueryUnescape(value)

		switch key {
		case "oauth_callback":
			params.Callback = value
		case "oauth_consumer_key":
			params.ConsumerKey = value
		case "oauth_nonce":
			params.Nonce = value
		case "oauth_signature":
			params.Signature = value
		case "oauth_signature_method":
			params.SignatureMethod = value
		case "oauth_timestamp":
			params.Timestamp = value
		case "oauth_version":
			params.Version = value
		case "oauth_token":
			params.Token = value
		case "oauth_verifier":
			params.Verifier = value
		}
	}

	return params, nil
}

func VerifyOAuthSignature(params *OAuthParams, method, requestURL, consumerSecret string) bool {
	// Create base string
	baseParams := map[string]string{
		"oauth_callback":         params.Callback,
		"oauth_consumer_key":     params.ConsumerKey,
		"oauth_nonce":            params.Nonce,
		"oauth_signature_method": params.SignatureMethod,
		"oauth_timestamp":        params.Timestamp,
		"oauth_version":          params.Version,
	}

	// Sort parameters alphabetically
	var keys []string
	for k := range baseParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build parameter string
	var paramString string
	for i, k := range keys {
		if i > 0 {
			paramString += "&"
		}
		paramString += fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(baseParams[k]))
	}

	// Create signature base string
	signatureBase := fmt.Sprintf("%s&%s&%s",
		method,
		url.QueryEscape(requestURL),
		url.QueryEscape(paramString))

	// Create signing key
	signingKey := fmt.Sprintf("%s&", url.QueryEscape(consumerSecret))

	// Calculate HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(signatureBase))
	calculatedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return calculatedSignature == params.Signature
}

func RequestToken(c *fiber.Ctx) error {
	// Parse OAuth header
	authHeader := c.Get("Authorization")
	oauthParams, err := ParseOAuthHeader(authHeader)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid OAuth header")
	}

	// Verify timestamp is recent
	timestamp, _ := strconv.ParseInt(oauthParams.Timestamp, 10, 64)
	if time.Now().Unix()-timestamp > 300 { // 5 minute window
		return c.Status(fiber.StatusBadRequest).SendString("OAuth timestamp expired")
	}

	// Verify signature
	// If this is intended for an application, you can use this.
	// if !VerifyOAuthSignature(oauthParams, "POST", c.BaseURL()+c.Path(), configData.ConsumerSecret) {
	// 	return c.Status(fiber.StatusUnauthorized).SendString("Invalid OAuth signature")
	// }

	// Generate temporary token and secret
	tempToken := uuid.New().String()
	tempSecret := uuid.New().String()

	// Store the temporary token
	token := TempToken{
		Token:     tempToken,
		Secret:    tempSecret,
		CreatedAt: time.Now(),
		ExpiresIn: tempTokenExpiration,
		Callback:  oauthParams.Callback,
	}
	tempTokens.Store(tempToken, token)

	// Format response according to OAuth 1.0a spec
	response := fmt.Sprintf(
		"oauth_token=%s&oauth_token_secret=%s&oauth_callback_confirmed=true",
		url.QueryEscape(tempToken),
		url.QueryEscape(tempSecret),
	)

	c.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.SendString(response)
}

func ServeOAuthLoginPage(c *fiber.Ctx) error {
	oauthToken := c.Query("oauth_token")
	if oauthToken == "" {
		return c.SendString("missing oauth_token")
	}

	_, ok := tempTokens.Load(oauthToken)
	if !ok {
		return c.Status(400).SendString("invalid/expired oauth_token")
	}

	return c.Render("authorize", fiber.Map{
		"RedirectUrl": func() string {
			return "http://127.0.0.1/"
		}(),
		"OauthToken": oauthToken,
	}, "authorize")
}

func GenerateNumericPIN(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("PIN length must be positive")
	}

	const charset = "0123456789"
	pin := make([]byte, length)

	for i := 0; i < length; i++ {
		// Generate a random index within the charset length
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		pin[i] = charset[randomIndex.Int64()]
	}

	return string(pin), nil
}

func AttemptToAuthenticateWithOauth(c *fiber.Ctx) error {
	oauth_token := c.FormValue("oauth_token")
	if oauth_token == "" {
		return c.Status(400).SendString("missing oauth_token")
	}

	tokenDataAny, ok := tempTokens.Load(oauth_token)
	if !ok {
		return c.Status(400).SendString("invalid/expired oauth_token")
	}

	tokenData, ok := tokenDataAny.(TempToken)
	if !ok {
		return c.Status(400).SendString("invalid/expired oauth_token")
	}

	if c.FormValue("auth_type") == "apppassword" {
		username := c.FormValue("username")
		password := c.FormValue("password")

		res, pds, err := blueskyapi.Authenticate(username, password)
		if err != nil {
			// failed auth
			return HandleBlueskyError(c, err.Error(), "com.atproto.server.createSession", access_token)
		}

		// successful auth

		// store auth data for future reference
		tokenData.AuthResp = res
		tokenData.AuthPDS = pds

		// create verifier
		if tokenData.Callback == "oob" {
			// generate our pin
			tokenData.Verifier, err = GenerateNumericPIN(7)
			if err != nil {
				return c.Status(500).SendString("An error occured while generating a random ping, this is likely a bug!")
			}

			tempTokens.Store(oauth_token, tokenData)
			return c.SendString(fmt.Sprintf("Your pin is %s. Use it in the app you are trying to log into.", tokenData.Verifier))
		} else {
			// random base64 data
			tokenData.Verifier = rand.Text()

			baseURL, err := url.Parse(tokenData.Callback)
			if err != nil {
				return c.Status(400).SendString("callback url is invalid")
			}
			query := baseURL.Query()

			query.Add("oauth_token", oauth_token)
			query.Add("oauth_verifier", tokenData.Verifier)

			baseURL.RawQuery = query.Encode()

			tempTokens.Store(oauth_token, tokenData)
			return c.Redirect(baseURL.String())
		}

	} else {
		return c.Status(400).SendString("missing valid auth_type")
	}

}

func OAuthAccessToken(c *fiber.Ctx) error {
	// Parse OAuth header
	authHeader := c.Get("Authorization")
	oauthParams, err := ParseOAuthHeader(authHeader)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid OAuth header")
	}

	// Verify timestamp is recent
	timestamp, _ := strconv.ParseInt(oauthParams.Timestamp, 10, 64)
	if time.Now().Unix()-timestamp > 300 { // 5 minute window
		return c.Status(fiber.StatusBadRequest).SendString("OAuth timestamp expired")
	}

	// Verify signature
	// If this is intended for an application, you can use this.
	// if !VerifyOAuthSignature(oauthParams, "POST", c.BaseURL()+c.Path(), configData.ConsumerSecret) {
	// 	return c.Status(fiber.StatusUnauthorized).SendString("Invalid OAuth signature")
	// }

	tokenDataAny, ok := tempTokens.Load(oauthParams.Token)
	if !ok {
		return c.Status(400).SendString("invalid/expired oauth_token")
	}

	tokenData, ok := tokenDataAny.(TempToken)
	if !ok {
		return c.Status(400).SendString("invalid/expired oauth_token")
	}

	tempTokens.Delete(oauthParams.Verifier)

	if tokenData.Verifier != oauthParams.Verifier {
		return c.Status(400).SendString("incorrect verifier, try again")
	}

	return ReturnSuccessfulAuth(c, *tokenData.AuthResp, *tokenData.AuthPDS)
}
