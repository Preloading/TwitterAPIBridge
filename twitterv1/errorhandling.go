package twitterv1

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type Error struct {
	Code    int    `json:"code" xml:"code,attr"`
	Message string `json:"message" xml:",chardata"`
}

type BlueskyError struct {
	Error   string
	Message string
}

type Errors struct {
	XMLName xml.Name `json:"-" xml:"errors"`
	Error   []Error  `json:"errors"`
}

func ReturnError(c *fiber.Ctx, message string, error_code int, http_error int) error {
	if c.Query("suppress_response_codes") != "true" {
		c.Status(http_error)
	}

	err := Errors{
		Error: []Error{
			{
				Code:    error_code,
				Message: message,
			},
		},
	}

	return EncodeAndSend(c, err)
}

func HandleBlueskyError(c *fiber.Ctx, responseJson string, lexicon string, function func(c *fiber.Ctx) error) error {
	// json decode responce
	res := BlueskyError{}
	if err := json.Unmarshal([]byte(responseJson), &res); err != nil {
		fmt.Println(string(responseJson))
		switch responseJson {
		case "invalid handle", "user does not exist":
			return ReturnError(c, "Incorrect username", 32, fiber.StatusUnauthorized)
		}
		return ReturnError(c, "An unknown error occured.", 0, fiber.StatusInternalServerError)
	}

	switch res.Error {
	// Generic
	case "InvalidRequest":
		switch res.Message {
		case "Profile not found":
			return ReturnError(c, "User not found.", 50, fiber.StatusNotFound)
		default:
			return ReturnError(c, "Invalid request.", 0, fiber.StatusBadRequest) // unknown
		}

	case "InvalidSwap":
		return ReturnError(c, "An error occured during record manipulation (InvalidSwap)", 131, fiber.StatusInternalServerError)

	// Feed
	case "InvalidFeed":
		return ReturnError(c, "The feed specified was invalid", 0, fiber.StatusBadRequest) // unknown
	case "NotFound": // could probably be
		return ReturnError(c, "Post was not found. (or was deleted)", 144, fiber.StatusNotFound)

	// Search
	case "BadQueryString":
		return ReturnError(c, "Invalid query.", 0, fiber.StatusBadRequest)

	// Auth
	case "ExpiredToken":
		return ReturnError(c, "Expired token.", 89, 403) // TODO: If this occurs, refresh the token instead of failing.
	case "InvalidToken":
		return ReturnError(c, "Invalid token.", 89, 403)
	case "AccountTakedown":
		return ReturnError(c, "Your account has been suspended. Check your email for details.", 64, fiber.StatusForbidden)
	case "AuthFactorTokenRequired":
		return ReturnError(c, "Two-factor authentication is required, use an app password.", 32, fiber.StatusUnauthorized) // Unsure about this error code.
	case "AuthMissing":
		return ReturnError(c, "Incorrect username/password.", 32, fiber.StatusUnauthorized)
	case "AuthenticationRequired":
		return ReturnError(c, "Incorrect username/password.", 32, fiber.StatusUnauthorized)
	case "RateLimitExceeded":
		return ReturnError(c, "Rate limit exceeded contacting Bluesky. Please try again later.", 88, fiber.StatusTooManyRequests)

	default:
		// Handle other errors
		fmt.Println("An unknown error occured! Error: " + res.Error + ". Message: " + res.Message)
		return ReturnError(c, "An unknown error occured: "+res.Message, 0, fiber.StatusInternalServerError)
	}
}

func MissingAuth(c *fiber.Ctx, err error) error {
	if err != nil {
		switch err.Error() {
		case "oauth token not found":
			return ReturnError(c, "Missing authentication token.", 215, 400)
		case "invalid token":
			return ReturnError(c, "Invalid authentication token.", 89, 403)
		case "refresh token has expired":
			return ReturnError(c, "Authentication token has expired.", 89, 403)
		case "incorrect server":
			return ReturnError(c, "Wrong server for your login. Please verify your URLs match between applications.", 89, 403)
		case "invalid app password":
			return ReturnError(c, "App passwords required on this app", 215, 400)
		}

	}
	return ReturnError(c, "Missing authentication token.", 215, 400)
}
