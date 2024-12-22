package bridge

import (
	"bytes"
	"encoding/base32"
	"encoding/xml"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

type RelatedResultsQuery struct {
	Annotations []Annotations `json:"annotations"` // TODO
	ResultType  string        `json:"resultType"`
	Score       float64       `json:"score"`
	GroupName   string        `json:"groupName"`
	Results     []Results     `json:"results"`
}

type Results struct {
	Kind        string        `json:"kind"`
	Score       float64       `json:"score"`
	Annotations []Annotations `json:"annotations"` // TODO
	Value       Tweet         `json:"value"`
}

// TODO: Figure out what this is for, and how to use this
type Annotations struct {
	ConversationRole string `json:"ConversationRole"`
}

type Retweet struct {
	Tweet
	RetweetedStatus Tweet `json:"retweeted_status"`
}

// https://web.archive.org/web/20120708212016/https://dev.twitter.com/docs/platform-objects/tweets
type Tweet struct {
	Coordinates  interface{} `json:"coordinates"`
	Favourited   bool        `json:"favorited"`
	CreatedAt    string      `json:"created_at"`
	Truncated    bool        `json:"truncated"`
	Entities     Entities    `json:"entities"`
	Text         string      `json:"text"`
	Annotations  interface{} `json:"annotations"`
	Contributors interface{} `json:"contributors"`
	ID           big.Int     `json:"id"`
	IDStr        string      `json:"id_str"`
	Geo          interface{} `json:"geo"`
	Place        interface{} `json:"place"`
	User         TwitterUser `json:"user,omitempty"`
	Source       string      `json:"source"`

	// Reply stuff
	InReplyToUserID      *big.Int `json:"in_reply_to_user_id"`
	InReplyToUserIDStr   *string  `json:"in_reply_to_user_id_str"`
	InReplyToStatusID    *big.Int `json:"in_reply_to_status_id"`
	InReplyToStatusIDStr *string  `json:"in_reply_to_status_id_str"`
	InReplyToScreenName  *string  `json:"in_reply_to_screen_name"`

	// The following aren't found in home_timeline, but can be found when directly fetching a tweet.

	PossiblySensitive bool `json:"possibly_sensitive"`

	// Tweet... stats?
	RetweetCount int `json:"retweet_count"`

	// Our user's interaction with the tweet
	Retweeted       bool   `json:"retweeted"`
	RetweetedStatus *Tweet `json:"retweeted_status,omitempty"`
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
	Status              *Tweet `json:"status,omitempty"`
}

type TwitterActivitiySummary struct {
	Favourites      []big.Int `json:"favoriters"` // Pretty sure this is the User ID of the favouriters
	FavouritesCount int       `json:"favoriters_count"`
	Repliers        []big.Int `json:"repliers"`
	RepliersCount   int       `json:"repliers_count"`
	Retweets        []big.Int `json:"retweeters"`
	RetweetsCount   int       `json:"retweeters_count"`
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

// Used in the /friends/lookup endpoint
type UsersRelationship struct {
	Name        string      `json:"name" xml:"name"`
	IDStr       string      `json:"id_str" xml:"id_str"`
	ID          big.Int     `json:"id" xml:"id"`
	Connections Connections `json:"connections" xml:"connections"`
	ScreenName  string      `json:"screen_name" xml:"screen_name"`
}

type Connection struct {
	XMLName xml.Name `xml:"connection"`
	Value   string   `xml:",chardata"`
}

type Connections struct {
	XMLName    xml.Name     `xml:"connections"`
	Connection []Connection `xml:"connection"`
}

type UserRelationships struct {
	Relationships []UsersRelationship `json:"relationship" xml:"relationship"`
}

// https://web.archive.org/web/20120516154953/https://dev.twitter.com/docs/api/1/get/friendships/show
// used in the /friendships/show endpoint
type UserFriendship struct {
	ID                   big.Int `json:"id" xml:"id"`
	IDStr                string  `json:"id_str" xml:"id_str"`
	ScreenName           string  `json:"screen_name" xml:"screen_name"`
	Following            bool    `json:"following" xml:"following"`
	FollowedBy           bool    `json:"followed_by" xml:"followed_by"`
	NotificationsEnabled *bool   `json:"notifications_enabled" xml:"notifications_enabled"` // unknown
	CanDM                *bool   `json:"can_dm,omitempty" xml:"can_dm,omitempty"`
	Blocking             *bool   `json:"blocking" xml:"blocking"`           // unknown
	WantRetweets         *bool   `json:"want_retweets" xml:"want_retweets"` // unknown
	MarkedSpam           *bool   `json:"marked_spam" xml:"marked_spam"`     // unknown
	AllReplies           *bool   `json:"all_replies" xml:"all_replies"`     // unknown
}

type SourceTargetFriendship struct {
	XMLName xml.Name       `xml:"relationship"`
	Source  UserFriendship `json:"source" xml:"source"`
	Target  UserFriendship `json:"target" xml:"target"`
}

// Bluesky's API returns a letter ID for each user,
// While twitter uses a numeric ID, meaning we
// need to convert between the two

// Base36 characters (digits and lowercase letters)
const base38Chars = "0123456789abcdefghijklmnopqrstuvwxyz:/."

// BlueSkyToTwitterID converts a letter ID to a compact numeric representation using Base37
func BlueSkyToTwitterID(letterID string) *big.Int {
	letterID = strings.ToLower(letterID)
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
func TwitterIDToBlueSky(numericID big.Int) string {
	if numericID.Cmp(big.NewInt(0)) == 0 {
		return string(base38Chars[0])
	}

	base := big.NewInt(39)
	letterID := ""
	tempID := new(big.Int).Set(&numericID) // Create a copy of numericID

	for tempID.Cmp(big.NewInt(0)) > 0 {
		remainder := new(big.Int)
		tempID.DivMod(tempID, base, remainder)
		letterID = string(base38Chars[remainder.Int64()]) + letterID
	}

	return letterID
}

func BskyMsgToTwitterID(uri string, creationTime *time.Time, retweetUserId *string) big.Int {
	var encodedId *big.Int
	if retweetUserId != nil {
		encodedId = BlueSkyToTwitterID(fmt.Sprintf("%s:/:%s:/:%s", uri, creationTime.Format("20060102T15:04:05Z"), *retweetUserId))
	} else {
		encodedId = BlueSkyToTwitterID(fmt.Sprintf("%s:/:%s", uri, creationTime.Format("20060102T15:04:05Z")))
	}
	return *encodedId
}

// This is here soley because we have to use psudo ids for retweets.
func TwitterMsgIdToBluesky(id *big.Int) (*string, *time.Time, *string, error) {
	// If the tweet is not a retweet, we can use the timestamp inside of the bluesky ID
	// Yup! I was also suprised that the timestamp is in the ID
	// See https://atproto.com/specs/tid

	// Although if it is a retweet, we can't rely on the timestamp in the ID, since we don't have the proper ID, only the reposter & the original tweet
	// Theoretically we could hack it together since we know the time of the retweet (and the thing we are missing is the encoded timestamp), but that
	// seems really finky
	encodedId := TwitterIDToBlueSky(*id)
	uri := ""
	retweetUserId := ""
	timestamp := time.Time{}
	err := error(nil)

	parts := strings.Split(encodedId, ":/:")
	if len(parts) == 3 {
		// Retweet
		uri = parts[0]
		retweetUserId = parts[2]
		timestamp, err = time.Parse("20060102T15:04:05Z", strings.ToUpper(parts[1]))
		if err != nil {
			return nil, nil, nil, err
		}
	} else if len(parts) == 2 {
		// Any other type of tweet
		uri = parts[0]
		timestamp, err = time.Parse("20060102T15:04:05Z", strings.ToUpper(parts[1]))
		if err != nil {
			return nil, nil, nil, err
		}
		// Example URI: at://did:plc:sykr3znzovcjo7kkvt4z5ywh/app.bsky.feed.post/3ldlvtjyjwc22
		// uriparts := strings.Split(encodedId, "/")
		// if len(uriparts) != 5 {
		// 	return nil, nil, nil, errors.New("invalid URI format")
		// }
		// timestamp, err = DecodeTID(uriparts[4])
	} else {
		return nil, nil, nil, errors.New("invalid ID format")
	}
	return &uri, &timestamp, &retweetUserId, nil
}

// DecodeTID decodes a base32-sortable encoded TID and extracts the timestamp.
// Bluesky Specific
// Doesn't work, i can't figure out why
func DecodeTID(tid string) (time.Time, error) {
	fmt.Println("TID: " + tid)
	if len(tid) != 13 {
		return time.Time{}, errors.New("invalid TID length")
	}

	// Base32-sortable character set
	base32Chars := "234567abcdefghijklmnopqrstuvwxyz"
	base32Encoding := base32.NewEncoding(base32Chars).WithPadding(base32.NoPadding)

	// Decode the TID
	decoded, err := base32Encoding.DecodeString(tid)
	if err != nil {
		return time.Time{}, err
	}

	// Convert the decoded bytes to a 64-bit integer
	var id uint64
	for _, b := range decoded {
		id = (id << 8) | uint64(b)
	}
	fmt.Println(id)
	// Extract the timestamp (53 bits) and convert to microseconds
	timestampMicroseconds := id >> 10

	// Convert microseconds to time.Time
	timestamp := time.Unix(0, int64(timestampMicroseconds)*1000)
	fmt.Println("Decoded: " + timestamp.String())

	return timestamp, nil
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

	xmlContent = bytes.ReplaceAll(xmlContent, []byte("<"+oldHeaderName+">"), []byte("<"+newHeaderName+">"))
	xmlContent = bytes.ReplaceAll(xmlContent, []byte("</"+oldHeaderName+">"), []byte("</"+newHeaderName+">"))

	// Add custom XML header
	customHeader := []byte(`<?xml version="1.0" encoding="UTF-8"?>`)
	xmlContent = append(customHeader, xmlContent...)

	result := string(xmlContent)
	return &result, nil
}
