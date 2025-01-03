package bridge

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/Preloading/MastodonTwitterAPI/db_controller"
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
	ID           int64       `json:"id"`
	IDStr        string      `json:"id_str"`
	Geo          interface{} `json:"geo"`
	Place        interface{} `json:"place"`
	User         TwitterUser `json:"user,omitempty"`
	Source       string      `json:"source"`

	// Reply stuff
	InReplyToUserID      *int64  `json:"in_reply_to_user_id"`
	InReplyToUserIDStr   *string `json:"in_reply_to_user_id_str"`
	InReplyToStatusID    *int64  `json:"in_reply_to_status_id"`
	InReplyToStatusIDStr *string `json:"in_reply_to_status_id_str"`
	InReplyToScreenName  *string `json:"in_reply_to_screen_name"`

	// The following aren't found in home_timeline, but can be found when directly fetching a tweet.

	PossiblySensitive bool `json:"possibly_sensitive"`

	// Tweet... stats?
	RetweetCount int `json:"retweet_count"`

	// Our user's interaction with the tweet
	Retweeted          bool                `json:"retweeted"`
	RetweetedStatus    *Tweet              `json:"retweeted_status,omitempty"`
	CurrentUserRetweet *CurrentUserRetweet `json:"current_user_retweet,omitempty"`
}
type CurrentUserRetweet struct {
	ID    int64  `json:"id"`
	IDStr string `json:"id_str"`
}

type TwitterUser struct {
	XMLName                   xml.Name `xml:"user" json:"-"`
	Name                      string   `json:"name" xml:"name"`
	ProfileSidebarBorderColor string   `json:"profile_sidebar_border_color" xml:"profile_sidebar_border_color"`
	ProfileBackgroundTile     bool     `json:"profile_background_tile" xml:"profile_background_tile"`
	ProfileSidebarFillColor   string   `json:"profile_sidebar_fill_color" xml:"profile_sidebar_fill_color"`
	CreatedAt                 string   `json:"created_at" xml:"created_at"`
	ProfileImageURL           string   `json:"profile_image_url" xml:"profile_image_url"`
	ProfileImageURLHttps      string   `json:"profile_image_url_https" xml:"profile_image_url_https"`
	Location                  string   `json:"location" xml:"location"`
	ProfileLinkColor          string   `json:"profile_link_color" xml:"profile_link_color"`
	FollowRequestSent         bool     `json:"follow_request_sent" xml:"follow_request_sent"`
	URL                       string   `json:"url" xml:"url"`
	FavouritesCount           int      `json:"favourites_count" xml:"favourites_count"`
	ContributorsEnabled       bool     `json:"contributors_enabled" xml:"contributors_enabled"`
	UtcOffset                 *int     `json:"utc_offset" xml:"utc_offset"`
	ID                        int64    `json:"id" xml:"id"`
	IDStr                     string   `json:"id_str" xml:"id_str"`
	ProfileUseBackgroundImage bool     `json:"profile_use_background_image" xml:"profile_use_background_image"`
	ProfileTextColor          string   `json:"profile_text_color" xml:"profile_text_color"`
	Protected                 bool     `json:"protected" xml:"protected"`
	FollowersCount            int      `json:"followers_count" xml:"followers_count"`
	Lang                      string   `json:"lang" xml:"lang"`
	Notifications             *bool    `json:"notifications" xml:"notifications"`
	TimeZone                  *string  `json:"time_zone" xml:"time_zone"`
	Verified                  bool     `json:"verified" xml:"verified"`
	ProfileBackgroundColor    string   `json:"profile_background_color" xml:"profile_background_color"`
	GeoEnabled                bool     `json:"geo_enabled" xml:"geo_enabled"`
	Description               string   `json:"description" xml:"description"`
	FriendsCount              int      `json:"friends_count" xml:"friends_count"`
	StatusesCount             int      `json:"statuses_count" xml:"statuses_count"`
	ProfileBackgroundImageURL string   `json:"profile_background_image_url" xml:"profile_background_image_url"`
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
	Favourites      []int64 `json:"favoriters"` // Pretty sure this is the User ID of the favouriters
	FavouritesCount string  `json:"favoriters_count"`
	Repliers        []int64 `json:"repliers"`
	RepliersCount   string  `json:"repliers_count"`
	Retweets        []int64 `json:"retweeters"`
	RetweetsCount   string  `json:"retweeters_count"`
}

type MediaSize struct {
	W      int    `json:"w"`
	Resize string `json:"resize"`
	H      int    `json:"h"`
}

type Media struct {
	ID            int64                `json:"id"`
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
	Name       string `json:"name"`
	ID         *int64 `json:"id"`
	IDStr      string `json:"id_str"`
	Indices    []int  `json:"indices"`
	ScreenName string `json:"screen_name"`
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
	XMLName             xml.Name        `xml:"settings" json:"-"`
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
	XMLName     xml.Name    `xml:"relationship" json:"-"`
	Name        string      `json:"name" xml:"name"`
	IDStr       string      `json:"id_str" xml:"id_str"`
	ID          int64       `json:"id" xml:"id"`
	Connections Connections `json:"connections" xml:"connections"`
	ScreenName  string      `json:"screen_name" xml:"screen_name"`
}

type Connection struct {
	XMLName xml.Name `xml:"connection" json:"-"`
	Value   string   `xml:",chardata"`
}

type Connections struct {
	XMLName    xml.Name     `xml:"connections" json:"-"`
	Connection []Connection `xml:"connection"`
}

type UserRelationships struct {
	XMLName       xml.Name            `xml:"relationships" json:"-"`
	Relationships []UsersRelationship `json:"relationship" xml:"relationship"`
}

// Currently known how it forms follows, but we are missing favourites etc
type MyActivity struct {
	Action        string        `json:"action" xml:"action"`
	CreatedAt     string        `json:"created_at" xml:"created_at"`
	ID            int64         `json:"id" xml:"id"`
	Sources       []TwitterUser `json:"sources" xml:"sources"`
	Targets       []Tweet       `json:"targets,omitempty" xml:"targets,omitempty"`
	TargetObjects []Tweet       `json:"target_objects,omitempty" xml:"target_objects,omitempty"`
}

// https://web.archive.org/web/20120516154953/https://dev.twitter.com/docs/api/1/get/friendships/show
// used in the /friendships/show endpoint
type UserFriendship struct {
	ID                   int64  `json:"id" xml:"id"`
	IDStr                string `json:"id_str" xml:"id_str"`
	ScreenName           string `json:"screen_name" xml:"screen_name"`
	Following            bool   `json:"following" xml:"following"`
	FollowedBy           bool   `json:"followed_by" xml:"followed_by"`
	NotificationsEnabled *bool  `json:"notifications_enabled" xml:"notifications_enabled"` // unknown
	CanDM                *bool  `json:"can_dm,omitempty" xml:"can_dm,omitempty"`
	Blocking             *bool  `json:"blocking" xml:"blocking"`           // unknown
	WantRetweets         *bool  `json:"want_retweets" xml:"want_retweets"` // unknown
	MarkedSpam           *bool  `json:"marked_spam" xml:"marked_spam"`     // unknown
	AllReplies           *bool  `json:"all_replies" xml:"all_replies"`     // unknown
}

type SourceTargetFriendship struct {
	XMLName xml.Name       `xml:"relationship" json:"-"`
	Source  UserFriendship `json:"source" xml:"source"`
	Target  UserFriendship `json:"target" xml:"target"`
}

type Trends struct {
	Created   time.Time       `json:"created_at"` // EVERYWHERE except here it uses a different format for time. why.
	Trends    []Trend         `json:"trends"`
	AsOf      time.Time       `json:"as_of"`
	Locations []TrendLocation `json:"locations"` // no idea when i implenented thsi function, but i digress.
}

type Trend struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Promoted    bool   `json:"promoted"`
	Query       string `json:"query"`
	TweetVolume int    `json:"tweet_volume"`
}

type TwitterUsers struct {
	XMLName xml.Name      `xml:"users" json:"-"`
	Users   []TwitterUser `xml:"user"`
}

type TwitterRecommendation struct {
	UserID int64       `json:"user_id"`
	User   TwitterUser `json:"user"`
	Token  string      `json:"token"`
}

type InternalSearchResult struct {
	Statuses []Tweet `json:"statuses"`
}

type FacetParsing struct {
	Start int
	End   int
	Item  string
}

func encodeToUint63(input string) *int64 {
	hasher := fnv.New64a()                  // Create a new FNV-1a 64-bit hash
	hasher.Write([]byte(input))             // Write the input string as bytes
	hash := hasher.Sum64()                  // Get the 64-bit hash
	result := int64(hash & ((1 << 63) - 1)) // Mask the MSB to ensure 63 bits and convert to int64
	return &result
}

// Bluesky's API returns a letter ID for each user,
// While twitter uses a numeric ID, meaning we
// need to convert between the two

func BlueSkyToTwitterID(letterID string) *int64 {
	if letterID == "" {
		return nil
	}
	twitterId := encodeToUint63(letterID)
	if err := db_controller.StoreTwitterIdInDatabase(twitterId, letterID, nil, nil); err != nil {
		fmt.Println("Error storing Twitter ID in database:", err)
		panic(err)
	}
	return twitterId
}

// TwitterIDToBlueSky converts a numeric ID to a letter ID representation using Base37
func TwitterIDToBlueSky(numericID *int64) (*string, error) {
	// Get the letter ID from the database
	letterID, _, _, err := db_controller.GetTwitterIDFromDatabase(numericID)
	if err != nil {
		return nil, err
	}

	return letterID, nil
}

func BskyMsgToTwitterID(uri string, creationTime *time.Time, retweetUserId *string) *int64 {
	if uri == "" {
		return nil
	}

	var encodedId *int64
	if retweetUserId != nil {
		encodedId = encodeToUint63(uri + *retweetUserId + creationTime.Format("20060102150405")) // We add the date to avoid having the same ID for reposts
		if err := db_controller.StoreTwitterIdInDatabase(encodedId, uri, creationTime, retweetUserId); err != nil {
			fmt.Println("Error storing Twitter ID in database:", err)
			panic(err) // TODO: handle this gracefully?
		}
	} else {
		encodedId = encodeToUint63(uri)
		if err := db_controller.StoreTwitterIdInDatabase(encodedId, uri, creationTime, nil); err != nil {
			fmt.Println("Error storing Twitter ID in database:", err)
			panic(err)
		}
	}
	return encodedId
}

// This is here soley because we have to use psudo ids for retweets.
func TwitterMsgIdToBluesky(id *int64) (*string, *time.Time, *string, error) {
	// Get the letter ID from the database
	uri, createdAt, retweetUserId, err := db_controller.GetTwitterIDFromDatabase(id)
	if err != nil {
		return nil, nil, nil, err
	}

	return uri, createdAt, retweetUserId, nil
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
