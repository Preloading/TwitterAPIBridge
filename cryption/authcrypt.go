package cryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// GenerateKey generates a new AES key and returns it as a base64 encoded string
func GenerateKey() (string, error) {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// Encrypt encrypts the given plaintext using AES-GCM with the provided base64 encoded key and returns the ciphertext as a base64 encoded string
func Encrypt(plaintext, base64Key string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts the given base64 encoded ciphertext using AES-GCM with the provided base64 encoded key and returns the plaintext
func Decrypt(base64Ciphertext, base64Key string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(base64Ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// it took me an embarrising long time to learn that the jwt tokens encode the expiration time inside of them
func GetJWTTokenExpirationUnix(token string) (*float64, error) {
	// Split the JWT token into its components
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT token")
	}

	// Decode the payload (2nd part of the JWT)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	// Parse the payload into a map
	var claims map[string]interface{}
	err = json.Unmarshal(payload, &claims)
	if err != nil {
		return nil, err
	}

	// Extract the `exp` claim
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("expiration time not found in token")
	}
	return &exp, nil
}

// Base64URLEncode encodes the given string using base64 and returns the result as a URL-safe string
func Base64URLEncode(input string) string {
	base64 := base64.StdEncoding.EncodeToString([]byte(input))

	base64 = strings.ReplaceAll(base64, "+", "-")
	base64 = strings.ReplaceAll(base64, "/", "_")
	base64 = strings.ReplaceAll(base64, "=", "") // remove padding

	return base64
}

// Base64URLDecode decodes the given URL-safe string using base64 and returns the result as a string
func Base64URLDecode(input string) (string, error) {
	input = strings.ReplaceAll(input, "-", "+")
	input = strings.ReplaceAll(input, "_", "/")

	// Add padding if necessary
	switch len(input) % 4 {
	case 2:
		input += "=="
	case 3:
		input += "="
	}

	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}
