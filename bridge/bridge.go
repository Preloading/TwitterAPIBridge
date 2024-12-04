package bridge

import (
	"math/big"
	"strings"
	"time"
)

type Tweet struct {
	Coordinates       interface{} `json:"coordinates"` // I do not think anything implients this in modern day
	Favourited        bool        `json:"favorited"`
	CreatedAt         string      `json:"created_at"`
	Truncated         bool        `json:"truncated"`
	Entities          Entities    `json:"entities"`
	Text              string      `json:"text"`
	Annotations       interface{} `json:"annotations"`  // Unknown
	Contributors      interface{} `json:"contributors"` // Unknown
	ID                big.Int     `json:"id"`
	Geo               interface{} `json:"geo"`                 // I do not think anything impliments this in modern day
	Place             interface{} `json:"place"`               // Unknown
	InReplyToUserID   *big.Int    `json:"in_reply_to_user_id"` // Unknown, but guessing int
	User              TwitterUser `json:"user"`
	Source            string      `json:"source"`
	InReplyToStatusID *big.Int    `json:"in_reply_to_status_id"`
}

type TwitterUser struct {
	Name                      string  `json:"name"`
	ProfileSidebarBorderColor string  `json:"profile_sidebar_border_color"` // Hex color (w/o hashtag)
	ProfileBackgroundTile     bool    `json:"profile_background_tile"`
	ProfileSidebarFillColor   string  `json:"profile_sidebar_fill_color"` // Hex color (w/o hashtag)
	CreatedAt                 string  `json:"created_at"`
	ProfileImageURL           string  `json:"profile_image_url"`
	Location                  string  `json:"location"`
	ProfileLinkColor          string  `json:"profile_link_color"` // Hex color (w/o hashtag)
	FollowRequestSent         bool    `json:"follow_request_sent"`
	URL                       string  `json:"url"`
	FavouritesCount           int     `json:"favourites_count"`
	ContributorsEnabled       bool    `json:"contributors_enabled"`
	UtcOffset                 int     `json:"utc_offset"`
	ID                        big.Int `json:"id"`
	ProfileUseBackgroundImage bool    `json:"profile_use_background_image"`
	ProfileTextColor          string  `json:"profile_text_color"` // Hex color (w/o hashtag)
	Protected                 bool    `json:"protected"`
	FollowersCount            int     `json:"followers_count"`
	Lang                      string  `json:"lang"`
	Notifications             bool    `json:"notifications"`
	TimeZone                  string  `json:"time_zone"` // oh god it's in text form aaaa
	Verified                  bool    `json:"verified"`
	ProfileBackgroundColor    string  `json:"profile_background_color"` // Hex color (w/o hashtag)
	GeoEnabled                bool    `json:"geo_enabled"`              // No clue what this does
	Description               string  `json:"description"`
	FriendsCount              int     `json:"friends_count"`
	StatusesCount             int     `json:"statuses_count"`
	ProfileBackgroundImageURL string  `json:"profile_background_image_url"`
	Following                 bool    `json:"following"`
	ScreenName                string  `json:"screen_name"`
	ShowAllInlineMedia        bool    `json:"show_all_inline_media"`
	IDStr                     string  `json:"id_str"`
	IsTranslator              bool    `json:"is_translator"`
	ListedCount               int     `json:"listed_count"`
}

type MediaSize struct {
	W      int    `json:"w"`
	Resize string `json:"resize"`
	H      int    `json:"h"`
}

type Media struct {
	ID            big.Int              `json:"id"`
	IDStr         string               `json:"id_str"`
	MediaURL      string               `json:"media_url"`
	MediaURLHttps string               `json:"media_url_https"`
	URL           string               `json:"url"`
	DisplayURL    string               `json:"display_url"`
	ExpandedURL   string               `json:"expanded_url"`
	Sizes         map[string]MediaSize `json:"sizes"`
	Type          string               `json:"type"`
	Indices       []int                `json:"indices"`
}

type Entities struct {
	Media        []Media       `json:"media"`
	Urls         []interface{} `json:"urls"` // TODO
	UserMentions []UserMention `json:"user_mentions"`
	Hashtags     []interface{} `json:"hashtags"` // TODO
}

type UserMention struct {
	Name       string  `json:"name"`
	ID         big.Int `json:"id"`
	Indices    []int   `json:"indices"` // Indices[0] is how many charectors till the first letter of the mention, Indicies[1] is how many charectors till the last letter of the mention
	ScreenName string  `json:"screen_name"`
}

type SleepTime struct {
	EndTime   *string `json:"end_time"`
	Enabled   bool    `json:"enabled"`
	StartTime *string `json:"start_time"`
}

type PlaceType struct {
	Name string `json:"name"`
	Code int    `json:"code"`
}

type TrendLocation struct {
	Name        string    `json:"name"`
	Woeid       int       `json:"woeid"`
	PlaceType   PlaceType `json:"placeType"`
	Country     string    `json:"country"`
	URL         string    `json:"url"`
	CountryCode *string   `json:"countryCode"`
}

type TimeZone struct {
	Name       string `json:"name"`
	TzinfoName string `json:"tzinfo_name"`
	UtcOffset  int    `json:"utc_offset"`
}

type Config struct {
	SleepTime           SleepTime       `json:"sleep_time"`
	TrendLocation       []TrendLocation `json:"trend_location"`
	Language            string          `json:"language"`
	AlwaysUseHttps      bool            `json:"always_use_https"`
	DiscoverableByEmail bool            `json:"discoverable_by_email"`
	TimeZone            TimeZone        `json:"time_zone"`
	GeoEnabled          bool            `json:"geo_enabled"`
}

// Bluesky's API returns a letter ID for each user,
// While twitter uses a numeric ID, meaning we
// need to convert between the two

// Base36 characters (digits and lowercase letters)
const base38Chars = "0123456789abcdefghijklmnopqrstuvwxyz:/."

// BlueSkyToTwitterID converts a letter ID to a compact numeric representation using Base37
func BlueSkyToTwitterID(letterID string) *big.Int {
	numericID := big.NewInt(0)
	base := big.NewInt(39)

	for _, char := range letterID {
		base37Value := strings.IndexRune(base38Chars, char)
		if base37Value == -1 {
			// Handle invalid characters
			continue
		}
		numericID.Mul(numericID, base)
		numericID.Add(numericID, big.NewInt(int64(base37Value)))
	}

	return numericID
}

// TwitterIDToBlueSky converts a numeric ID to a letter ID representation using Base37
func TwitterIDToBlueSky(numericID *big.Int) string {
	if numericID.Cmp(big.NewInt(0)) == 0 {
		return string(base38Chars[0])
	}

	base := big.NewInt(39)
	letterID := ""

	for numericID.Cmp(big.NewInt(0)) > 0 {
		remainder := new(big.Int)
		numericID.DivMod(numericID, base, remainder)
		letterID = string(base38Chars[remainder.Int64()]) + letterID
	}

	return letterID
}

// EncodeIDs concatenates two IDs into one string with a delimiter
func EncodeBlueskyMessageID(userid, messageid string) big.Int {
	return *BlueSkyToTwitterID(userid + "/" + messageid)
}

// DecodeIDs splits the encoded string back into the original two IDs
func DecodeBlueskyMessageID(encoded *big.Int) (string, string) {
	ids := strings.Split(TwitterIDToBlueSky(encoded), "/")
	if len(ids) != 2 {
		return "", ""
	}
	return ids[0], ids[1]
}

// FormatTime converts Go's time.Time into the format "Wed Sep 01 00:00:00 +0000 2021"
func TwitterTimeConverter(t time.Time) string {
	return t.Format("Mon Jan 02 15:04:05 -0700 2006")
}
