package bridge

import (
	"math/big"
	"strings"
	"time"
)

type Tweet struct {
	Coordinates          interface{} `json:"coordinates"`
	Favourited           bool        `json:"favorited"`
	CreatedAt            string      `json:"created_at"`
	Truncated            bool        `json:"truncated"`
	Entities             Entities    `json:"entities"`
	Text                 string      `json:"text"`
	Annotations          interface{} `json:"annotations"`
	Contributors         interface{} `json:"contributors"`
	ID                   big.Int     `json:"id"`
	IDStr                string      `json:"id_str"`
	Geo                  interface{} `json:"geo"`
	Place                interface{} `json:"place"`
	InReplyToUserID      *big.Int    `json:"in_reply_to_user_id"`
	InReplyToUserIDStr   *string     `json:"in_reply_to_user_id_str"`
	User                 TwitterUser `json:"user"`
	Source               string      `json:"source"`
	InReplyToStatusID    *big.Int    `json:"in_reply_to_status_id"`
	InReplyToStatusIDStr *string     `json:"in_reply_to_status_id_str"`
	InReplyToScreenName  *string     `json:"in_reply_to_screen_name"`

	// The following aren't found in home_timeline, but can be found when directly fetching a tweet.

	PossiblySensitive bool `json:"possibly_sensitive"`

	// Tweet... stats?
	RetweetCount int `json:"retweet_count"`

	// Our user's interaction with the tweet
	Retweeted bool `json:"retweeted"`
}

type TwitterUser struct {
	Name                           string  `json:"name"`
	ProfileSidebarBorderColor      string  `json:"profile_sidebar_border_color"`
	ProfileBackgroundTile          bool    `json:"profile_background_tile"`
	ProfileSidebarFillColor        string  `json:"profile_sidebar_fill_color"`
	CreatedAt                      string  `json:"created_at"`
	ProfileImageURL                string  `json:"profile_image_url"`
	ProfileImageURLHttps           string  `json:"profile_image_url_https"`
	Location                       string  `json:"location"`
	ProfileLinkColor               string  `json:"profile_link_color"`
	FollowRequestSent              bool    `json:"follow_request_sent"`
	URL                            string  `json:"url"`
	FavouritesCount                int     `json:"favourites_count"`
	ContributorsEnabled            bool    `json:"contributors_enabled"`
	UtcOffset                      *int    `json:"utc_offset"`
	ID                             big.Int `json:"id"`
	IDStr                          string  `json:"id_str"`
	ProfileUseBackgroundImage      bool    `json:"profile_use_background_image"`
	ProfileTextColor               string  `json:"profile_text_color"`
	Protected                      bool    `json:"protected"`
	FollowersCount                 int     `json:"followers_count"`
	Lang                           string  `json:"lang"`
	Notifications                  *bool   `json:"notifications"` // TODO: Are we sure this is a bool? It's set to null on https://web.archive.org/web/20120708204036/https://dev.twitter.com/docs/api/1/get/statuses/show/%3Aid
	TimeZone                       *string `json:"time_zone"`
	Verified                       bool    `json:"verified"`
	ProfileBackgroundColor         string  `json:"profile_background_color"`
	GeoEnabled                     bool    `json:"geo_enabled"`
	Description                    string  `json:"description"`
	FriendsCount                   int     `json:"friends_count"`
	StatusesCount                  int     `json:"statuses_count"`
	ProfileBackgroundImageURL      string  `json:"profile_background_image_url"`
	ProfileBackgroundImageURLHttps string  `json:"profile_background_image_url_https"`
	Following                      *bool   `json:"following"` // TODO: Are we sure this is a bool? It's set to null on https://web.archive.org/web/20120708204036/https://dev.twitter.com/docs/api/1/get/statuses/show/%3Aid
	ScreenName                     string  `json:"screen_name"`
	ShowAllInlineMedia             bool    `json:"show_all_inline_media"`
	IsTranslator                   bool    `json:"is_translator"`
	ListedCount                    int     `json:"listed_count"`

	// not found in home_timeline
	DefaultProfile      bool `json:"default_profile"`
	DefaultProfileImage bool `json:"default_profile_image"`
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
	Urls         []URL         `json:"urls"`
	UserMentions []UserMention `json:"user_mentions"`
	Hashtags     []Hashtag     `json:"hashtags"`
}

type URL struct {
	ExpandedURL string `json:"expanded_url"`
	URL         string `json:"url"`
	Indices     []int  `json:"indices"`
	DisplayURL  string `json:"display_url"`
}

type Hashtag struct {
	Text    string `json:"text"`
	Indices []int  `json:"indices"`
}

type UserMention struct {
	Name       string  `json:"name"`
	ID         big.Int `json:"id"`
	IDStr      string  `json:"id_str"`
	Indices    []int   `json:"indices"`
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
