package bridge

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/Preloading/TwitterAPIBridge/db_controller"
	"github.com/golang-jwt/jwt/v5"
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

// RetweetedTweet is a special case of Tweet for retweets to avoid XML naming conflicts
type RetweetedTweet struct {
	Tweet
	XMLName xml.Name `xml:"retweeted_status" json:"-"` // Ahhhh, XML f u n.
}

// https://web.archive.org/web/20120708212016/https://dev.twitter.com/docs/platform-objects/tweets
type Tweet struct {
	XMLName      xml.Name    `xml:"status" json:"-"`
	Coordinates  interface{} `json:"coordinates" xml:"coordinates"`
	Favourited   bool        `json:"favorited" xml:"favorited"`
	CreatedAt    string      `json:"created_at" xml:"created_at"`
	Truncated    bool        `json:"truncated" xml:"truncated"`
	Entities     Entities    `json:"entities" xml:"entities"`
	Text         string      `json:"text" xml:"text"`
	Annotations  interface{} `json:"annotations" xml:"annotations"`
	Contributors interface{} `json:"contributors" xml:"contributors"`
	ID           int64       `json:"id" xml:"id"`
	IDStr        string      `json:"id_str" xml:"-"`
	Geo          interface{} `json:"geo" xml:"geo"`
	Place        interface{} `json:"place" xml:"place"`
	User         TwitterUser `json:"user,omitempty" xml:"user,omitempty"`
	Source       string      `json:"source" xml:"source"`

	// Reply stuff
	InReplyToUserID      *int64  `json:"in_reply_to_user_id" xml:"in_reply_to_user_id"`
	InReplyToUserIDStr   *string `json:"in_reply_to_user_id_str" xml:"in_reply_to_user_id_str"`
	InReplyToStatusID    *int64  `json:"in_reply_to_status_id" xml:"in_reply_to_status_id"`
	InReplyToStatusIDStr *string `json:"in_reply_to_status_id_str" xml:"in_reply_to_status_id_str"`
	InReplyToScreenName  *string `json:"in_reply_to_screen_name" xml:"in_reply_to_screen_name"`

	// The following aren't found in home_timeline, but can be found when directly fetching a tweet.

	PossiblySensitive bool `json:"possibly_sensitive" xml:"possibly_sensitive"`

	// Tweet... stats?
	RetweetCount int `json:"retweet_count" xml:"retweet_count"`

	// Our user's interaction with the tweet
	Retweeted          bool                `json:"retweeted" xml:"retweeted"`
	RetweetedStatus    *RetweetedTweet     `json:"retweeted_status,omitempty" xml:"retweeted_status,omitempty"`
	CurrentUserRetweet *CurrentUserRetweet `json:"current_user_retweet,omitempty" xml:"current_user_retweet,omitempty"`
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
	ProfileBannerURL          string   `json:"profile_banner_url" xml:"profile_banner_url"`
	ProfileBannerURLHttps     string   `json:"profile_banner_url_https" xml:"profile_banner_url_https"`
	ProfileBackgroundImageURL string   `json:"profile_background_image_url" xml:"profile_background_image_url"`
	// ProfileBackgroundImageURLHttps string  `json:"profile_background_image_url_https" xml:"profile_background_image_url_https"`
	Following           *bool  `json:"following" xml:"following"`
	ScreenName          string `json:"screen_name" xml:"screen_name"`
	ShowAllInlineMedia  bool   `json:"show_all_inline_media" xml:"show_all_inline_media"`
	IsTranslator        bool   `json:"is_translator" xml:"is_translator"`
	ListedCount         int    `json:"listed_count" xml:"listed_count"`
	DefaultProfile      bool   `json:"default_profile" xml:"default_profile"`
	DefaultProfileImage bool   `json:"default_profile_image" xml:"default_profile_image"`
	Status              *Tweet `json:"status,omitempty" xml:"status,omitempty"`
}

type TwitterActivitiySummary struct {
	Favourites      []int64 `json:"favoriters"` // Pretty sure this is the User ID of the favouriters
	FavouritesCount string  `json:"favoriters_count"`
	Repliers        []int64 `json:"repliers"`
	RepliersCount   string  `json:"repliers_count"`
	Retweets        []int64 `json:"retweeters"`
	RetweetsCount   string  `json:"retweeters_count"`
}

type Size struct {
	W      int    `json:"w" xml:"w"`
	Resize string `json:"resize" xml:"resize"`
	H      int    `json:"h" xml:"h"`
}

type MediaSize struct {
	Thumb  Size `json:"thumb" xml:"thumb"`
	Small  Size `json:"small" xml:"small"`
	Medium Size `json:"medium" xml:"medium"`
	Large  Size `json:"large" xml:"large"`
}

type MediaXML struct {
	XMLName       xml.Name  `xml:"creative" json:"-"`
	ID            int64     `xml:"id"`
	MediaURL      string    `xml:"media_url"`
	MediaURLHttps string    `xml:"media_url_https"`
	URL           string    `xml:"url"`
	DisplayURL    string    `xml:"display_url"`
	ExpandedURL   string    `xml:"expanded_url"`
	Sizes         MediaSize `xml:"sizes"`
	Type          string    `xml:"type"`
	Start         int       `xml:"start,attr"`
	End           int       `xml:"end,attr"`
}

type Media struct {
	XMLName       xml.Name  `xml:"media" json:"-"`
	XMLFormat     MediaXML  `xml:",innerxml" json:"-"`
	ID            int64     `json:"id" xml:"id"`
	IDStr         string    `json:"id_str" xml:"-"`
	MediaURL      string    `json:"media_url" xml:"-"`
	MediaURLHttps string    `json:"media_url_https" xml:"-"`
	URL           string    `json:"url,omitempty" xml:"-"`
	DisplayURL    string    `json:"display_url,omitempty" xml:"-"`
	ExpandedURL   string    `json:"expanded_url,omitempty" xml:"-"`
	Sizes         MediaSize `json:"sizes" xml:"-"`
	// Sizes         map[string]MediaSize `json:"sizes"`
	Type    string `json:"type" xml:"-"`
	Indices []int  `json:"indices,omitempty" xml:"-"`
}

type Entities struct {
	Media        []Media       `json:"media" xml:"media"`
	Urls         []URL         `json:"urls" xml:"urls"`
	UserMentions []UserMention `json:"user_mentions" xml:"user_mentions"`
	Hashtags     []Hashtag     `json:"hashtags" xml:"hashtags"`
}

type URL struct {
	XMLName     xml.Name     `xml:"urls" json:"-"`
	XMLFormat   URLXMLFormat `xml:",innerxml" json:"-"`
	URL         string       `json:"url" xml:"-"`
	DisplayURL  string       `json:"display_url" xml:"-"`
	ExpandedURL string       `json:"expanded_url" xml:"-"`
	Indices     []int        `json:"indices" xml:"-"`
	Start       int          `json:"start" xml:"-"`
	End         int          `json:"end" xml:"-"`
}

type URLXMLFormat struct {
	XMLName     xml.Name `xml:"url" json:"-"`
	Start       int      `xml:"start,attr"`
	End         int      `xml:"end,attr"`
	URL         string   `xml:"url"`
	ExpandedURL string   `xml:"expanded_url"`
	DisplayURL  string   `xml:"display_url"`
}

type Hashtag struct {
	Text    string `json:"text" xml:"text"`
	Indices []int  `json:"indices" xml:"-"`
	Start   int    `json:"-" xml:"start,attr"`
	End     int    `json:"-" xml:"end,attr"`
}

type UserMention struct {
	Name       string `json:"name" xml:"name"`
	ID         *int64 `json:"id" xml:"id"`
	IDStr      string `json:"id_str" xml:"id_str"`
	Indices    []int  `json:"indices" xml:"-"`
	Start      int    `json:"-" xml:"start,attr"`
	End        int    `json:"-" xml:"end,attr"`
	ScreenName string `json:"screen_name" xml:"screen_name"`
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
	DiscoverableByEmail bool            `json:"discoverable_by_email"`
	TimeZone            TimeZone        `json:"time_zone" xml:"time_zone"`
	GeoEnabled          bool            `json:"geo_enabled" xml:"geo_enabled"`
}

// Used in the /friends/lookup endpoint
type UsersRelationship struct {
	XMLName        xml.Name    `xml:"relationship" json:"-"`
	Name           string      `json:"name" xml:"name"`
	IDStr          string      `json:"id_str" xml:"id_str"`
	ID             int64       `json:"id" xml:"id"`
	Connections    []string    `json:"connections" xml:"-"` // JSON representation
	ConnectionsXML Connections `json:"-" xml:"connections"` // XML representation
	ScreenName     string      `json:"screen_name" xml:"screen_name"`
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
// Some details can be found here:
// https://mgng.mugbum.info/Archive/View/year/2011/month/12
type MyActivity struct {
	Action        string        `json:"action" xml:"action"`
	CreatedAt     string        `json:"created_at" xml:"created_at"`
	MaxPosition   string        `json:"max_position" xml:"max_position"`
	MinPosition   string        `json:"min_position" xml:"min_position"`
	Sources       []TwitterUser `json:"sources"`
	Targets       []Tweet       `json:"targets" xml:"targets"`
	TargetObjects []Tweet       `json:"target_objects" xml:"target_objects"`
}

// https://web.archive.org/web/20120516154953/https://dev.twitter.com/docs/api/1/get/friendships/show
// used in the /friendships/show endpoint
type UserFriendship struct {
	ID                   int64  `json:"id" xml:"id"`
	IDStr                string `json:"id_str" xml:"-"`
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

type SourceTargetFriendshipRoot struct {
	XMLName  xml.Name               `xml:"relationships" json:"-"` // ?
	Relation SourceTargetFriendship `json:"relationship" xml:"relationship"`
}

type Trends struct {
	Created   time.Time       `json:"created_at" xml:"created_at"` // EVERYWHERE except here it uses a different format for time. why.
	Trends    []Trend         `json:"trends" xml:"trends"`
	AsOf      time.Time       `json:"as_of" xml:"as_of"`
	Locations []TrendLocation `json:"locations" xml:"locations"` // no idea when i implenented thsi function, but i digress.
}

type Trend struct {
	Name        string `json:"name" xml:"name"`
	URL         string `json:"url" xml:"url"`
	Promoted    bool   `json:"promoted" xml:"promoted"`
	Query       string `json:"query" xml:"query"`
	TweetVolume int    `json:"tweet_volume" xml:"tweet_volume"`
}

type TwitterUsers struct {
	XMLName xml.Name      `xml:"users" json:"-"`
	Users   []TwitterUser `xml:"user"`
}

type TwitterRecommendation struct {
	UserID int64       `json:"user_id" xml:"user_id"`
	User   TwitterUser `json:"user" xml:"user"`
	Token  string      `json:"token" xml:"token"`
}

type InternalSearchResult struct {
	Statuses []Tweet `json:"statuses" xml:"statuses"`
}

type FacetParsing struct {
	Start int
	End   int
	Item  string
}

type Discovery struct {
	Statuses            []Tweet              `json:"statuses" xml:"statuses"`
	Stories             []Story              `json:"stories" xml:"stories"`
	RelatedQueries      []RelatedQuery       `json:"related_queries" xml:"related_queries"`
	SpellingCorrections []SpellingCorrection `json:"spelling_corrections" xml:"spelling_corrections"`
}

type Story struct {
	Type        string      `json:"type" xml:"type"` // news or topic
	Score       float32     `json:"score" xml:"score"`
	Data        StoryData   `json:"data" xml:"data"`
	SocialProof SocialProof `json:"social_proof" xml:"social_proof"`
}

type StoryData struct {
	Title    string        `json:"title" xml:"title"`
	Articles []NewsArticle `json:"articles" xml:"articles"`
}

// cringe that i need these multiple

type StoryURL struct {
	DisplayURL  string `json:"display_url" xml:"display_url"`
	ExpandedURL string `json:"expanded_url" xml:"expanded_url"`
}

type StoryMediaInfo struct {
	MediaURL  string          `json:"media_url" xml:"media_url"`
	Type      string          `json:"type" xml:"type"`
	Width     int             `json:"width" xml:"width"`
	Height    int             `json:"height" xml:"height"`
	VideoInfo *StoryVideoInfo `json:"video_info,omitempty" xml:"video_info,omitempty"`
}

// speculating, is this for proper video support?
type StoryVideoInfo struct {
	Variants []StoryVideoVariant `json:"variants" xml:"variants"`
}

type StoryVideoVariant struct {
	URL         string `json:"url" xml:"url"`
	ContentType string `json:"content_type" xml:"content_type"`
}

type NewsArticle struct {
	Title       string           `json:"title" xml:"title"`
	Url         StoryURL         `json:"url" xml:"url"`
	Description string           `json:"description" xml:"description"`
	TweetCount  int              `json:"tweet_count" xml:"tweet_count"`
	Attribution string           `json:"attribution" xml:"attribution"`
	Score       float32          `json:"score" xml:"score"`
	Query       string           `json:"query" xml:"query"`
	Name        string           `json:"name" xml:"name"` // no clue what the difference is between name and title. i guess the article's author
	Media       []StoryMediaInfo `json:"media" xml:"media"`
}

type SocialProof struct {
	Type         string                    `json:"social_proof_type" xml:"social_proof_type"` // social or query
	ReferencedBy SocialProofedReferencedBy `json:"referenced_by" xml:"referenced_by"`
}

type SocialProofedReferencedBy struct {
	Statuses    []Tweet `json:"statuses" xml:"statuses"`
	GlobalCount int     `json:"global_count" xml:"global_count"`
}

type RelatedQuery struct {
	Query string `json:"query" xml:"query"`
}
type QueryWithIndices struct {
	Query   string `json:"query" xml:"query"`
	Indices []int  `json:"indices" xml:"-"`
	Start   int    `json:"-" xml:"start,attr"`
	End     int    `json:"-" xml:"end,attr"`
}

//arrays upon arrays

type SpellingCorrection struct {
	Results []SpellingCorrectionResults `json:"results" xml:"results"`
}

type SpellingCorrectionResults struct {
	Value QueryWithIndices `json:"value" xml:"value"`
}

type AuthToken struct {
	*jwt.RegisteredClaims
	Version          int      `json:"version"`           // Version of the token
	Platform         string   `json:"platform"`          // What platform (bsky, mastodon, etc) was this token made on.
	CryptoKey        string   `json:"crypto_key"`        // An AES key used to make this mostly stateless.
	DID              string   `json:"did"`               // Bluesky user did
	TokenUUID        string   `json:"token_uuid"`        // The UUID of the token used to identify it.
	ServerIdentifier string   `json:"server_identifier"` // A way to identify the server that issued this token. Useful for any service that wants to use A Twitter Bridge.
	ServerURLs       []string `json:"server_urls"`       // URLs to access that server
}

type TwitterLists struct {
	XMLName xml.Name      `xml:"lists" json:"-"`
	Lists   []TwitterList `json:"lists" xml:"list"`
	Cursors
}

type TwitterListMembers struct {
	XMLName xml.Name       `xml:"users" json:"-"`
	Users   []*TwitterUser `json:"users" xml:"users"`
	Cursors
}

type Cursors struct {
	PreviousCursor    int64  `json:"previous_cursor" xml:"previous_cursor"`
	PreviousCursorStr string `json:"previous_cursor_str" xml:"previous_cursor_str"`
	NextCursor        uint64 `json:"next_cursor" xml:"next_cursor"`
	NextCursorStr     string `json:"next_cursor_str" xml:"next_cursor_str"`
}

type TwitterList struct {
	XMLName         xml.Name    `xml:"list" json:"-"`
	Slug            string      `json:"slug" xml:"slug"`
	Name            string      `json:"name" xml:"name"`
	URI             string      `json:"uri" xml:"uri"`
	IDStr           string      `json:"id_str" xml:"id_str"`
	SubscriberCount int         `json:"subscriber_count" xml:"subscriber_count"`
	MemberCount     int         `json:"member_count" xml:"member_count"`
	Mode            string      `json:"mode" xml:"mode"`
	ID              int64       `json:"id" xml:"id"`
	FullName        string      `json:"full_name" xml:"full_name"`
	Description     string      `json:"description" xml:"description"`
	User            TwitterUser `json:"user" xml:"user"`
	Following       bool        `json:"following" xml:"following"`
}

type TopicSuggestion struct {
	Name string `json:"name" xml:"name"`
	Slug string `json:"slug" xml:"slug"`
	Size int    `json:"size" xml:"size"`
}

// https://web.archive.org/web/20120516160741/https://dev.twitter.com/docs/api/1/get/users/suggestions/%3Aslug
type TopicUserSuggestions struct {
	Name  string         `json:"name" xml:"name"`
	Slug  string         `json:"slug" xml:"slug"`
	Size  int            `json:"size" xml:"size"`
	Users []*TwitterUser `json:"users" xml:"users"`
}

// Search Ahead
// https://web.archive.org/web/20220427214446/https://twitter.com/i/search/typeahead.json?count=20&filters=true&result_type=true&src=COMPOSE&q=firat_ber
type SearchAhead struct {
	NumberOfResults int              `json:"num_results" xml:"num_results"`
	Users           []SummarisedUser `json:"users" xml:"users"`
	Topics          []string         `json:"topics" xml:"topics"`     // unimplemented
	Events          []string         `json:"events" xml:"events"`     // unimplemented
	Lists           []string         `json:"lists" xml:"lists"`       // unimplemented, and i don't think you can search lists (least w/o clearsky)
	Oneclick        []string         `json:"oneclick" xml:"oneclick"` // unimplemented and clueless as to what this is.
	Hashtags        []string         `json:"hashtags" xml:"hashtags"` // unimplemented
	CompletedIn     float64          `json:"completed_in" xml:"completed_in"`
	Query           string           `json:"query" xml:"query"`
}

type SummarisedUser struct {
	XMLName              xml.Name      `xml:"user" json:"-"`
	ID                   int64         `json:"id" xml:"id"`
	IDStr                string        `json:"id_str" xml:"id_str"`
	Verified             bool          `json:"verified" xml:"verified"`
	IsDMAble             bool          `json:"is_dm_able" xml:"is_dm_able"`
	IsBlocked            bool          `json:"is_blocked" xml:"is_blocked"`
	Name                 string        `json:"name" xml:"name"`
	ScreenName           string        `json:"screen_name" xml:"screen_name"`
	ProfileImageURL      string        `json:"profile_image_url" xml:"profile_image_url"`
	ProfileImageURLHttps string        `json:"profile_image_url_https" xml:"profile_image_url_https"`
	Location             string        `json:"location" xml:"location"` // unimplemented
	IsProtected          bool          `json:"is_protected" xml:"is_protected"`
	RoundedScore         int           `json:"rounded_score" xml:"rounded_score"`
	ConnectedUserCount   int           `json:"connected_user_count" xml:"connected_user_count"`
	ConnectedUserIds     []int64       `json:"connected_user_ids" xml:"connected_user_ids"`
	SocialProofsOrdered  []string      `json:"social_proofs_ordered" xml:"social_proofs_ordered"` // Unimplemented
	SocialContext        SocialContext `json:"social_context" xml:"social_context"`
	Tokens               []Token       `json:"tokens" xml:"tokens"`
	Inline               bool          `json:"inline" xml:"inline"` // unimplemented
}

type Token struct {
	Token string `json:"token" xml:"token"`
}

type SocialContext struct {
	Following  bool `json:"following" xml:"following"`
	FollowedBy bool `json:"followed_by" xml:"followed_by"`
}

// https://gist.github.com/ZweiSteinSoft/4733612#file-push_destinations-json, although I checked the code, and it only uses 2 of those values. (on both 5.0.3, and 4.1.3)
type PushDestination struct {
	AvailableLevels int `json:"available_levels" xml:"available-levels"` // This is used by the app, a responce I found online shows this at 1021, but idk what that means.
	EnabledFor      int `json:"enabled_for" xml:"enabled-for"`           // 5
}

// maybe?
type IdsWithCursor struct {
	Ids []int64 `json:"ids"`
	Cursors
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

// TwitterIDToBlueSky converts a numeric ID to a letter ID
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
