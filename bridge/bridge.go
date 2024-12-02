package bridge

import (
	"fmt"
	"strconv"
	"strings"
)

type Tweet struct {
	Coordinates interface{} `json:"coordinates"` // I do not think anything implients this in modern day
	Favourited  bool        `json:"favorited"`
	CreatedAt   string      `json:"created_at"`
	Truncated   bool        `json:"truncated"`
	// lets agree for now that entities don't exist. that seems like a lot of effort
	Text            string      `json:"text"`
	Annotations     interface{} `json:"annotations"`  // Unknown
	Contributors    interface{} `json:"contributors"` // Unknown
	ID              int         `json:"id"`
	Geo             interface{} `json:"geo"`                 // I do not think anything impliments this in modern day
	Place           interface{} `json:"place"`               // Unknown
	InReplyToUserID int         `json:"in_reply_to_user_id"` // Unknown, but guessing int
	User            TweetUser   `json:"user"`
	Source          string      `json:"source"`
}

type TweetUser struct {
	Name                      string `json:"name"`
	ProfileSidebarBorderColor string `json:"profile_sidebar_border_color"` // Hex color (w/o hashtag)
	ProfileBackgroundTile     bool   `json:"profile_background_tile"`
	ProfileSidebarFillColor   string `json:"profile_sidebar_fill_color"` // Hex color (w/o hashtag)
	CreatedAt                 string `json:"created_at"`
	ProfileImageURL           string `json:"profile_image_url"`
	Location                  string `json:"location"`
	ProfileLinkColor          string `json:"profile_link_color"` // Hex color (w/o hashtag)
	FollowRequestSent         bool   `json:"follow_request_sent"`
	URL                       string `json:"url"`
	FavouritesCount           int    `json:"favourites_count"`
	ContributorsEnabled       bool   `json:"contributors_enabled"`
	UtcOffset                 int    `json:"utc_offset"`
	ID                        int    `json:"id"`
	ProfileUseBackgroundImage bool   `json:"profile_use_background_image"`
	ProfileTextColor          string `json:"profile_text_color"` // Hex color (w/o hashtag)
	Protected                 bool   `json:"protected"`
	FollowersCount            int    `json:"followers_count"`
	Lang                      string `json:"lang"`
	Notifications             bool   `json:"notifications"`
	TimeZone                  string `json:"time_zone"` // oh god it's in text form aaaa
	Verified                  bool   `json:"verified"`
	ProfileBackgroundColor    string `json:"profile_background_color"` // Hex color (w/o hashtag)
	GeoEnabled                bool   `json:"geo_enabled"`              // No clue what this does
	Description               string `json:"description"`
	FriendsCount              int    `json:"friends_count"`
	StatusesCount             int    `json:"statuses_count"`
	ProfileBackgroundImageURL string `json:"profile_background_image_url"`
	Following                 bool   `json:"following"`
	ScreenName                string `json:"screen_name"`
}

type TwitterUser struct {
	ProfileSidebarFillColor   string `json:"profile_sidebar_fill_color"` // Hex color (w/o hashtag)
	Name                      string `json:"name"`
	ProfileSidebarBorderColor string `json:"profile_sidebar_border_color"` // Hex color (w/o hashtag)
	ProfileBackgroundTile     bool   `json:"profile_background_tile"`
	CreatedAt                 string `json:"created_at"`
	ProfileImageURL           string `json:"profile_image_url"`
	Location                  string `json:"location"`
	ProfileLinkColor          string `json:"profile_link_color"` // Hex color (w/o hashtag)
	FollowRequestSent         bool   `json:"follow_request_sent"`
	IDStr                     string `json:"id_str"`
	IsTranslator              bool   `json:"is_translator"`
	ContributorsEnabled       bool   `json:"contributors_enabled"`
	URL                       string `json:"url"`
	FavouritesCount           int    `json:"favourites_count"`
	UtcOffset                 int    `json:"utc_offset"`
	ID                        int    `json:"id"`
	ProfileUseBackgroundImage bool   `json:"profile_use_background_image"`
	ListedCount               int    `json:"listed_count"`
	ProfileTextColor          string `json:"profile_text_color"` // Hex color (w/o hashtag)
	Protected                 bool   `json:"protected"`
	FollowersCount            int    `json:"followers_count"`
	Lang                      string `json:"lang"`
	Notifications             bool   `json:"notifications"`
	TimeZone                  string `json:"time_zone"` // oh god it's in text form aaaa
	Verified                  bool   `json:"verified"`
	ProfileBackgroundColor    string `json:"profile_background_color"` // Hex color (w/o hashtag)
	GeoEnabled                bool   `json:"geo_enabled"`              // No clue what this does
	Description               string `json:"description"`
	FriendsCount              int    `json:"friends_count"`
	StatusesCount             int    `json:"statuses_count"`
	ProfileBackgroundImageURL string `json:"profile_background_image_url"`
	Following                 bool   `json:"following"`
	ScreenName                string `json:"screen_name"`
	ShowAllInlineMedia        bool   `json:"show_all_inline_media"`
}

// Bluesky's API returns a letter ID for each user,
// While twitter uses a numeric ID, meaning we
// need to convert between the two

// Base36 characters (digits and lowercase letters)
const base36Chars = "0123456789abcdefghijklmnopqrstuvwxyz"

// BlueSkyToTwitterID converts a letter ID to a compact numeric representation using Base36
func BlueSkyToTwitterID(letterID string) int {
	// Remove the prefix "did:plc:"
	letterID = strings.TrimPrefix(letterID, "did:plc:")

	var numericID strings.Builder
	for _, char := range letterID {
		// Convert each character to its ASCII value
		asciiValue := int(char)
		// Encode the ASCII value using Base36
		numericID.WriteString(encodeBase36(asciiValue))
	}

	twitterID, _ := strconv.Atoi(numericID.String())
	return twitterID
}

// encodeBase36 encodes an integer to a Base36 string
func encodeBase36(num int) string {
	if num == 0 {
		return string(base36Chars[0])
	}

	var encoded strings.Builder
	for num > 0 {
		remainder := num % 36
		encoded.WriteByte(base36Chars[remainder])
		num = num / 36
	}

	// Reverse the encoded string
	result := encoded.String()
	runes := []rune(result)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// TwitterToBlueSky converts a compact numeric representation back to the original letter ID
func TwitterToBlueSky(numericID string) (string, error) {
	var letterID strings.Builder

	for i := 0; i < len(numericID); i += 2 {
		// Take two characters at a time
		chunk := numericID[i : i+2]
		asciiValue, err := decodeBase36(chunk)
		if err != nil {
			return "", err
		}
		letterID.WriteByte(byte(asciiValue))
	}

	// Add the prefix "did:plc:" back to the letter ID
	return "did:plc:" + letterID.String(), nil
}

// decodeBase36 decodes a Base36 string to an integer
func decodeBase36(encoded string) (int, error) {
	var num int
	for _, char := range encoded {
		index := strings.IndexRune(base36Chars, char)
		if index == -1 {
			return 0, fmt.Errorf("invalid character: %c", char)
		}
		num = num*36 + index
	}
	return num, nil
}
