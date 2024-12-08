package db_controller

import (
	"os"
	"path/filepath"

	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
	UserDID               string  `gorm:"column:user_did"`
	TokenUUID             string  `gorm:"column:token_uuid"`
	EncryptedAccessToken  string  `gorm:"column:encrypted_access_token"`
	EncryptedRefreshToken string  `gorm:"column:encrypted_refresh_token"`
	AccessExpiry          float64 `gorm:"column:access_expiry"`
	RefreshExpiry         float64 `gorm:"column:refresh_expiry"`
}

type MessageContext struct {
	UserDID         string `gorm:"column:user_did"`
	TokenUUID       string `gorm:"column:token_uuid"`
	LastMessageId   string `gorm:"column:message_id"`
	TimelineContext string `gorm:"column:timeline_context"`
}

var db *gorm.DB

func InitDB() {
	// Ensure the directory exists
	dbPath := "./db/twitterbridge.db"
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
		panic("failed to create database directory")
	}

	// Initialize the database connection
	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Auto-migrate the schema
	db.AutoMigrate(&Token{})
	db.AutoMigrate(&MessageContext{})
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
func StoreToken(did string, accessToken string, refreshToken string, encryptionKey string, accessExpiry float64, refreshExpiry float64) (*string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	tokenUUID, err := UpdateToken(uuid.String(), did, accessToken, refreshToken, encryptionKey, accessExpiry, refreshExpiry)
	if err != nil {
		return nil, err
	}

	return tokenUUID, nil
}

func UpdateToken(uuid string, did string, accessToken string, refreshToken string, encryptionKey string, accessExpiry float64, refreshExpiry float64) (*string, error) {
	token := Token{
		UserDID:   did,
		TokenUUID: uuid,
		EncryptedAccessToken: func() string {
			encryptedToken, err := bridge.Encrypt(accessToken, encryptionKey)
			if err != nil {
				panic("failed to encrypt access token")
			}
			return encryptedToken
		}(),
		EncryptedRefreshToken: func() string {
			encryptedToken, err := bridge.Encrypt(refreshToken, encryptionKey)
			if err != nil {
				panic("failed to encrypt refresh token")
			}
			return encryptedToken
		}(),
		AccessExpiry:  accessExpiry,
		RefreshExpiry: refreshExpiry,
	}

	if err := db.Where("user_did = ? AND token_uuid = ?", did, uuid).Assign(&token).FirstOrCreate(&token).Error; err != nil {
		return nil, err
	}

	return &token.TokenUUID, nil
}

func GetToken(did string, tokenUUID string, encryptionKey string) (*string, *string, *float64, *float64, error) {
	var token Token
	if err := db.Where("user_did = ? AND token_uuid = ?", did, tokenUUID).First(&token).Error; err != nil {
		return nil, nil, nil, nil, err
	}

	accessToken, err := bridge.Decrypt(token.EncryptedAccessToken, encryptionKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	refreshToken, err := bridge.Decrypt(token.EncryptedRefreshToken, encryptionKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return &accessToken, &refreshToken, &token.AccessExpiry, &token.RefreshExpiry, nil
}

// SetMessageContext stores or updates the message context in the database.
// Parameters:
// - did: The decentralized identifier of the user.
// - tokenUUID: The UUID of the token.
// - lastMessageId: The ID of the last message.
// - timelineContext: The context of the timeline.
// - encryptionKey: The key used to encrypt the context.
func SetMessageContext(did string, tokenUUID string, lastMessageId string, timelineContext string, encryptionKey string) error {
	encryptedLastMessageId, err := bridge.Encrypt(lastMessageId, encryptionKey)
	if err != nil {
		return err
	}

	encryptedTimelineContext, err := bridge.Encrypt(timelineContext, encryptionKey)
	if err != nil {
		return err
	}

	messageContext := MessageContext{
		UserDID:         did,
		TokenUUID:       tokenUUID,
		LastMessageId:   encryptedLastMessageId,
		TimelineContext: encryptedTimelineContext,
	}

	if err := db.Where("user_did = ? AND token_uuid = ?", did, tokenUUID).Assign(&messageContext).FirstOrCreate(&messageContext).Error; err != nil {
		return err
	}

	return nil
}

// GetMessageContext retrieves the message context from the database.
// Parameters:
// - did: The decentralized identifier of the user.
// - tokenUUID: The UUID of the token.
// - encryptionKey: The key used to decrypt the context.
// Returns:
// - The last message ID.
// - The timeline context.
// - An error if the operation fails.
func GetMessageContext(did string, tokenUUID string, encryptionKey string) (*string, *string, error) {
	var messageContext MessageContext
	if err := db.Where("user_did = ? AND token_uuid = ?", did, tokenUUID).First(&messageContext).Error; err != nil {
		return nil, nil, err
	}

	lastMessageId, err := bridge.Decrypt(messageContext.LastMessageId, encryptionKey)
	if err != nil {
		return nil, nil, err
	}

	timelineContext, err := bridge.Decrypt(messageContext.TimelineContext, encryptionKey)
	if err != nil {
		return nil, nil, err
	}

	return &lastMessageId, &timelineContext, nil
}
