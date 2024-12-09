package bridge

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

type Retweet struct {
	Tweet
	RetweetedStatus Tweet `json:"retweeted_status"`
}

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
	Retweeted       bool   `json:"retweeted"`
	RetweetedStatus *Tweet `json:"retweeted_status,omitempty"`
}

// TODO: Find a better way of doing this.
type TweetWithoutUserData struct {
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
	Source               string      `json:"source"`
	InReplyToStatusID    *big.Int    `json:"in_reply_to_status_id"`
	InReplyToStatusIDStr *string     `json:"in_reply_to_status_id_str"`
	InReplyToScreenName  *string     `json:"in_reply_to_screen_name"`

	// The following aren't found in home_timeline, but can be found when directly fetching a tweet.

	PossiblySensitive bool `json:"possibly_sensitive"`

	// Tweet... stats?
	RetweetCount int `json:"retweet_count"`

	// Our user's interaction with the tweet
	Retweeted       bool   `json:"retweeted"`
	RetweetedStatus *Tweet `json:"retweeted_status"`
}

type TwitterUser struct {
	Name                      string `json:"name" xml:"name"`
	ProfileSidebarBorderColor string `json:"profile_sidebar_border_color" xml:"profile_sidebar_border_color"`
	ProfileBackgroundTile     bool   `json:"profile_background_tile" xml:"profile_background_tile"`
	ProfileSidebarFillColor   string `json:"profile_sidebar_fill_color" xml:"profile_sidebar_fill_color"`
	CreatedAt                 string `json:"created_at" xml:"created_at"`
	ProfileImageURL           string `json:"profile_image_url" xml:"profile_image_url"`
	// ProfileImageURLHttps      string  `json:"profile_image_url_https" xml:"profile_image_url_https"`
	Location            string  `json:"location" xml:"location"`
	ProfileLinkColor    string  `json:"profile_link_color" xml:"profile_link_color"`
	FollowRequestSent   bool    `json:"follow_request_sent" xml:"follow_request_sent"`
	URL                 string  `json:"url" xml:"url"`
	FavouritesCount     int     `json:"favourites_count" xml:"favourites_count"`
	ContributorsEnabled bool    `json:"contributors_enabled" xml:"contributors_enabled"`
	UtcOffset           *int    `json:"utc_offset" xml:"utc_offset"`
	ID                  big.Int `json:"id" xml:"id"`
	// IDStr                          string  `json:"id_str" xml:"id_str"`
	ProfileUseBackgroundImage bool    `json:"profile_use_background_image" xml:"profile_use_background_image"`
	ProfileTextColor          string  `json:"profile_text_color" xml:"profile_text_color"`
	Protected                 bool    `json:"protected" xml:"protected"`
	FollowersCount            int     `json:"followers_count" xml:"followers_count"`
	Lang                      string  `json:"lang" xml:"lang"`
	Notifications             *bool   `json:"notifications" xml:"notifications"`
	TimeZone                  *string `json:"time_zone" xml:"time_zone"`
	Verified                  bool    `json:"verified" xml:"verified"`
	ProfileBackgroundColor    string  `json:"profile_background_color" xml:"profile_background_color"`
	GeoEnabled                bool    `json:"geo_enabled" xml:"geo_enabled"`
	Description               string  `json:"description" xml:"description"`
	FriendsCount              int     `json:"friends_count" xml:"friends_count"`
	StatusesCount             int     `json:"statuses_count" xml:"statuses_count"`
	ProfileBackgroundImageURL string  `json:"profile_background_image_url" xml:"profile_background_image_url"`
	// ProfileBackgroundImageURLHttps string  `json:"profile_background_image_url_https" xml:"profile_background_image_url_https"`
	Following           *bool  `json:"following" xml:"following"`
	ScreenName          string `json:"screen_name" xml:"screen_name"`
	ShowAllInlineMedia  bool   `json:"show_all_inline_media" xml:"show_all_inline_media"`
	IsTranslator        bool   `json:"is_translator" xml:"is_translator"`
	ListedCount         int    `json:"listed_count" xml:"listed_count"`
	DefaultProfile      bool   `json:"default_profile" xml:"default_profile"`
	DefaultProfileImage bool   `json:"default_profile_image" xml:"default_profile_image"`
}

type TwitterUserWithStatus struct {
	TwitterUser
	// Status TweetWithoutUserData `json:"status"`
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
	EndTime   *string `json:"end_time" xml:"end_time"`
	Enabled   bool    `json:"enabled" xml:"enabled"`
	StartTime *string `json:"start_time" xml:"start_time"`
}

type PlaceType struct {
	Name string `json:"name" xml:"name"`
	Code int    `json:"code" xml:"code"`
}

type TrendLocation struct {
	Name        string    `json:"name" xml:"name"`
	Woeid       int       `json:"woeid" xml:"woeid"`
	PlaceType   PlaceType `json:"placeType" xml:"placeType"`
	Country     string    `json:"country" xml:"country"`
	URL         string    `json:"url" xml:"url"`
	CountryCode *string   `json:"countryCode" xml:"countryCode"`
}

type TimeZone struct {
	Name       string `json:"name" xml:"name"`
	TzinfoName string `json:"tzinfo_name" xml:"tzinfo_name"`
	UtcOffset  int    `json:"utc_offset" xml:"utc_offset"`
}

type Config struct {
	SleepTime           SleepTime       `json:"sleep_time" xml:"sleep_time"`
	TrendLocation       []TrendLocation `json:"trend_location" xml:"trend_location"`
	Language            string          `json:"language" xml:"language"`
	AlwaysUseHttps      bool            `json:"always_use_https" xml:"always_use_https"`
	DiscoverableByEmail bool            `json:"discoverable_by_email" xml:"discoverable_by_email"`
	TimeZone            TimeZone        `json:"time_zone" xml:"time_zone"`
	GeoEnabled          bool            `json:"geo_enabled" xml:"geo_enabled"`
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

func BskyMsgToTwitterID(uri string, creationTime time.Time, retweetUserId *string) big.Int {
	var encodedId *big.Int
	if retweetUserId != nil {
		encodedId = BlueSkyToTwitterID(fmt.Sprintf("%s:/:%s", uri, *retweetUserId))
	} else {
		encodedId = BlueSkyToTwitterID(uri)
	}

	// Convert the numeric ID to a string and pad with zeros if necessary
	encodedIdStr := encodedId.Text(10)
	if len(encodedIdStr) < 167 {
		encodedIdStr = strings.Repeat("0", 167-len(encodedIdStr)) + encodedIdStr
	} else if len(encodedIdStr) > 167 {
		fmt.Println("Encoded ID exceeds 167 digits")
	}

	// Add the Unix timestamp at the start
	finalId := fmt.Sprintf("%d%s", creationTime.Unix(), encodedIdStr)

	finalBigInt, _ := new(big.Int).SetString(finalId, 10)
	return *finalBigInt
}

// This is here soley because we have to use psudo ids for retweets. And because we need to store the unix time inside of it
func TwitterMsgIdToBluesky(id *big.Int) (string, time.Time, *string) {
	idStr := id.Text(10)
	if len(idStr) <= 10 {
		return TwitterIDToBlueSky(id), time.Time{}, nil
	}

	unixTimeStr := idStr[:9]
	encodedIdStr := idStr[9:]

	encodedId, _ := new(big.Int).SetString(encodedIdStr, 10)
	uri, retweetUserId := TwitterIDToBlueSky(encodedId), ""

	if strings.Contains(uri, ":/:") {
		parts := strings.Split(uri, ":/:")
		if len(parts) == 2 {
			uri = parts[0]
			retweetUserId = parts[1]
		}
	}

	unixTime, _ := strconv.ParseInt(unixTimeStr, 10, 64)
	return uri, time.Unix(unixTime, 0), &retweetUserId
}

// FormatTime converts Go's time.Time into the format "Wed Sep 01 00:00:00 +0000 2021"
func TwitterTimeConverter(t time.Time) string {
	return t.Format("Mon Jan 02 15:04:05 -0700 2006")
}

func TwitterTimeParser(timeStr string) (time.Time, error) {
	layout := "Mon Jan 02 15:04:05 -0700 2006"
	return time.Parse(layout, timeStr)
}

func XMLEncoder(data interface{}, oldHeaderName string, newHeaderName string) (*string, error) {
	// Encode the data to XML
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(data); err != nil {
		fmt.Println("Error encoding XML:", err)
		return nil, err
	}

	// Remove the root element and replace with custom header
	xmlContent := buf.Bytes()
	start := bytes.Index(xmlContent, []byte("<"+oldHeaderName+">"))
	end := bytes.LastIndex(xmlContent, []byte("</"+oldHeaderName+">"))
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("invalid XML format")
	}
	xmlContent = xmlContent[start+len("<"+oldHeaderName+">") : end]

	// Add custom XML header and root element
	customHeader := []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<` + newHeaderName + `>` + "\n")
	xmlContent = append(customHeader, xmlContent...)

	// Add custom footer
	customFooter := []byte("\n</" + newHeaderName + ">")
	xmlContent = append(xmlContent, customFooter...)

	result := string(xmlContent)
	return &result, nil
}
