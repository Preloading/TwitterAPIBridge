package db_controller

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"strconv"

	"github.com/Preloading/TwitterAPIBridge/config"
	authcrypt "github.com/Preloading/TwitterAPIBridge/cryption"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// I will most likely need to store auth tokens in a database, as I can only really get one auth related value from the user, and I can't change that value.
// I will probably send the oauth token as:
// {user did}/{generated uuid}/{encryptionkey}

// Since the only way to detect if a user has logged out is "/1/account/push_destinations/destroy.xml", I do not have a reliable way to detect if a user has logged out.
// I also don't wanna think what would happen if someone found my DB and stole it. Then a bunch of people would have their auth tokens stolen. Having the encryption key
// lets me store the auth tokens where it's encrypted.

// This means that:
// 1. If they sign out, their token on the DB will be unretirevable.
// 2. If someone steals the DB, they can't do anything with the auth tokens.
// 3. Users have better piece of mind knowing that we can't access their bluesky account whenever.

// Is this overcomplicating things? Yes. But I think it's a good idea.

// I will probably use GORM for the DB aswell. Lets me use SQLite while testing, and MySQL/MarinaDB for prod.

// DB Schema:

// Table: tokens
// Columns:
// 1. user_did (string)
// 2. token_uuid (string)
// 3. encrypted_access_token (string)
// 4. encrypted_refresh_token (string)

// TODO: Later move this to actual documentation?

// Token represents the schema for the tokens table
type Token struct {
	UserDid               string  `gorm:"type:string;primaryKey;not null"`
	UserPDS               string  `gorm:"type:string;not null"`
	TokenUUID             string  `gorm:"type:string;primaryKey;not null"`
	EncryptedAccessToken  string  `gorm:"type:string;not null"`
	EncryptedRefreshToken string  `gorm:"type:string;not null"`
	AccessExpiry          float64 `gorm:"type:float;not null"`
	RefreshExpiry         float64 `gorm:"type:float;not null"`
}

type TwitterIDs struct {
	BlueskyID   string     `gorm:"type:string;not null"`
	TwitterID   string     `gorm:"type:string;primaryKey;not null"` // Ensure this has a unique constraint
	ReposterDid *string    `gorm:"type:string"`
	DateCreated *time.Time `gorm:"type:timestamp"`
}

type MessageContext struct {
	UserDid         string `gorm:"type:string;primaryKey;not null"`
	TokenUUID       string `gorm:"type:string;primaryKey;not null"`
	LastMessageId   string `gorm:"type:string;not null"`
	TimelineContext string `gorm:"type:string;not null"`
}

// Analytics seems cool, and me liek numbrers.

type AnalyticData struct {
	DataType             string    `gorm:"type:string;not null"`
	IPAddress            string    `gorm:"type:string;"`
	Language             string    `gorm:"type:string;"`
	UserAgent            string    `gorm:"type:string;"`
	TwitterClient        string    `gorm:"type:string"`
	TwitterClientVersion string    `gorm:"type:string"`
	Timestamp            time.Time `gorm:"type:timestamp"`
}

// ShortLink represents the schema for the short_links table
type ShortLink struct {
	ShortCode   string `gorm:"type:string;primaryKey;not null"`
	OriginalURL string `gorm:"type:string;not null"`
}

var db *gorm.DB
var cfg config.Config

func InitDB(_cfg config.Config) {
	cfg = _cfg
	// Ensure the directory exists
	if cfg.DatabaseType == "sqlite" {
		dbDir := filepath.Dir(cfg.DatabasePath)
		if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
			panic("failed to create database directory")
		}
	}

	// Initialize the database connection
	var err error
	switch cfg.DatabaseType {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	case "mysql":
		db, err = gorm.Open(mysql.Open(cfg.DatabasePath), &gorm.Config{})
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.DatabasePath), &gorm.Config{})
	default:
		panic("unsupported database type")
	}

	if err != nil {
		panic("failed to connect database")
	}

	// Auto-migrate the schema
	db.AutoMigrate(&Token{})
	db.AutoMigrate(&MessageContext{})
	db.AutoMigrate(&TwitterIDs{})
	db.AutoMigrate(&AnalyticData{})
	db.AutoMigrate(&ShortLink{}) // Add this line
}

// StoreToken stores an encrypted access token and refresh token in the database.
// It returns the UUID of the stored token or an error if the operation fails.
//
// Parameters:
// - did: The decentralized identifier of the user.
// - accessToken: The access token to be encrypted and stored.
// - refreshToken: The refresh token to be encrypted and stored.
// - encryptionKey: The key used to encrypt the tokens.
// - accessExpiry: The expiry time of the access token.
// - refreshExpiry: The expiry time of the refresh token.
//
// Returns:
// - The UUID of the stored token.
// - An error if the operation fails.
func StoreToken(did string, pds string, accessToken string, refreshToken string, encryptionKey string, accessExpiry float64, refreshExpiry float64) (*string, error) {
	// Check if token exists for this DID.
	// Generate new UUID for new token
	uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	// Update or create token
	finalUUID, err := UpdateToken(uuid.String(), did, pds, accessToken, refreshToken, encryptionKey, accessExpiry, refreshExpiry)
	if err != nil {
		return nil, err
	}

	return finalUUID, nil
}

func UpdateToken(uuid string, did string, pds string, accessToken string, refreshToken string, encryptionKey string, accessExpiry float64, refreshExpiry float64) (*string, error) {
	encryptedAccess, err := authcrypt.Encrypt(accessToken, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %v", err)
	}

	encryptedRefresh, err := authcrypt.Encrypt(refreshToken, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %v", err)
	}

	token := Token{
		UserDid:               did,
		TokenUUID:             uuid,
		UserPDS:               pds,
		EncryptedAccessToken:  encryptedAccess,
		EncryptedRefreshToken: encryptedRefresh,
		AccessExpiry:          accessExpiry,
		RefreshExpiry:         refreshExpiry,
	}

	result := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_did"},
			{Name: "token_uuid"},
		},
		UpdateAll: true,
	}).Create(&token)

	if result.Error != nil {
		return nil, result.Error
	}

	return &token.TokenUUID, nil
}

// GetToken retrieves account data from the database
// @results: accessToken, refreshToken, accessExpiry, refreshExpiry, pds, error

func GetToken(did string, tokenUUID string, encryptionKey string) (*string, *string, *float64, *float64, *string, error) {
	var token Token
	if err := db.Where("user_did = ? AND token_uuid = ?", did, tokenUUID).First(&token).Error; err != nil {
		return nil, nil, nil, nil, nil, err
	}

	accessToken, err := authcrypt.Decrypt(token.EncryptedAccessToken, encryptionKey)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	refreshToken, err := authcrypt.Decrypt(token.EncryptedRefreshToken, encryptionKey)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return &accessToken, &refreshToken, &token.AccessExpiry, &token.RefreshExpiry, &token.UserPDS, nil
}

// Stores ID data in the database.
// @params: twitterID, blueskyID, dateCreated, reposterDid
// @results: error
func StoreTwitterIdInDatabase(twitterID *int64, blueskyId string, dateCreated *time.Time, reposterDid *string) error {
	if twitterID == nil {
		return fmt.Errorf("twitterID is nil")
	}

	storedData := TwitterIDs{
		TwitterID:   strconv.FormatInt(*twitterID, 10), // Convert *int64 to string
		BlueskyID:   blueskyId,
		DateCreated: dateCreated,
		ReposterDid: reposterDid,
	}

	result := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "twitter_id"},
		},
		UpdateAll: true,
	}).Create(&storedData)

	if result.Error != nil {
		// If there's an error, try updating the existing record
		fmt.Println("Error:", result.Error)
		panic(result.Error)
		//return db.Model(&TwitterIDs{}).Where("twitter_id = ?", strconv.FormatUint(twitterID, 10)).Updates(storedData).Error
	}

	return nil
}

// Gets a twitter id from the database
// @params: twitterID
// @results: blueskyID, dateCreated, reposterDid, error
func GetTwitterIDFromDatabase(twitterID *int64) (*string, *time.Time, *string, error) {
	if twitterID == nil {
		return nil, nil, nil, fmt.Errorf("twitterID is nil")
	}

	var blueskyID TwitterIDs
	if err := db.Where("twitter_id = ?", strconv.FormatInt(*twitterID, 10)).First(&blueskyID).Error; err != nil {
		return nil, nil, nil, err
	}

	return &blueskyID.BlueskyID, blueskyID.DateCreated, blueskyID.ReposterDid, nil
}

// Stores analytic data (if enabled)
// -- TYPES --
// 1. "login"
// 2. "tweets viewed"
// 3. "tweets posted"
func StoreAnalyticData(data AnalyticData) {
	if !cfg.TrackAnalytics {
		return
	}

	result := db.Create(&data)
	if result.Error != nil {
		fmt.Println("Failed to store analytic data:", result.Error)
	}
}

// StoreShortLink stores a short link in the database
func StoreShortLink(shortCode string, originalURL string) error {
	shortLink := ShortLink{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
	}

	result := db.Create(&shortLink)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// GetOriginalURL retrieves the original URL from the database using the short code
func GetOriginalURL(shortCode string) (string, error) {
	var shortLink ShortLink
	if err := db.Where("short_code = ?", shortCode).First(&shortLink).Error; err != nil {
		return "", err
	}

	return shortLink.OriginalURL, nil
}
