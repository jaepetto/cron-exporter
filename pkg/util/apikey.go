package util

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

// GenerateAPIKey generates a cryptographically secure random API key
// The key is 32 bytes (256 bits) of entropy, encoded as base32 for readability
func GenerateAPIKey() (string, error) {
	// Generate 32 bytes of random data (256 bits of entropy)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode as base32 and remove padding for cleaner keys
	key := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)

	// Convert to lowercase for consistency
	key = strings.ToLower(key)

	// Add a prefix to identify these as cronmetrics API keys
	return fmt.Sprintf("cm_%s", key), nil
}

// ValidateAPIKeyFormat checks if an API key has the expected format
func ValidateAPIKeyFormat(apiKey string) bool {
	if apiKey == "" {
		return false
	}

	// Check for our prefix
	if !strings.HasPrefix(apiKey, "cm_") {
		return false
	}

	// Remove prefix and check the remaining part
	keyPart := strings.TrimPrefix(apiKey, "cm_")

	// Should be 52 characters (32 bytes * 8 bits / 5 bits per base32 char)
	if len(keyPart) != 52 {
		return false
	}

	// Check if it's valid base32
	_, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(keyPart))
	return err == nil
}
