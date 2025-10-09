package twitterv1

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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
}

type TempToken struct {
	Token     string
	Secret    string
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
