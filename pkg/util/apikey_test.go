package util

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	// Test basic generation
	key1, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Test that key has correct format
	if !strings.HasPrefix(key1, "cm_") {
		t.Errorf("API key should start with 'cm_', got: %s", key1)
	}

	// Test expected length (3 char prefix + 52 char base32)
	expectedLength := 3 + 52 // "cm_" + 52 chars
	if len(key1) != expectedLength {
		t.Errorf("API key should be %d characters long, got %d: %s", expectedLength, len(key1), key1)
	}

	// Test uniqueness - generate multiple keys
	key2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate second API key: %v", err)
	}

	if key1 == key2 {
		t.Errorf("Generated keys should be unique, but got identical keys: %s", key1)
	}

	// Test lowercase
	if key1 != strings.ToLower(key1) {
		t.Errorf("API key should be lowercase, got: %s", key1)
	}
}

func TestValidateAPIKeyFormat(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{
			name:     "valid generated key",
			apiKey:   "",
			expected: true,
		},
		{
			name:     "empty string",
			apiKey:   "",
			expected: false,
		},
		{
			name:     "wrong prefix",
			apiKey:   "xyz_abcdefghijklmnopqrstuvwxyz234567abcdefghijklmnopqr",
			expected: false,
		},
		{
			name:     "no prefix",
			apiKey:   "abcdefghijklmnopqrstuvwxyz234567abcdefghijklmnopqr",
			expected: false,
		},
		{
			name:     "too short",
			apiKey:   "cm_short",
			expected: false,
		},
		{
			name:     "too long",
			apiKey:   "cm_abcdefghijklmnopqrstuvwxyz234567abcdefghijklmnopqrtoolong",
			expected: false,
		},
		{
			name:     "invalid characters",
			apiKey:   "cm_abcdefghijklmnopqrstuvwxyz234567abcdefghijklmno@#$",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the "valid generated key" test, generate a real key
			if tt.name == "valid generated key" {
				key, err := GenerateAPIKey()
				if err != nil {
					t.Fatalf("Failed to generate API key for test: %v", err)
				}
				tt.apiKey = key
			}

			result := ValidateAPIKeyFormat(tt.apiKey)
			if result != tt.expected {
				t.Errorf("ValidateAPIKeyFormat(%q) = %v, expected %v", tt.apiKey, result, tt.expected)
			}
		})
	}
}

func TestAPIKeyUniqueness(t *testing.T) {
	// Generate multiple keys and ensure they're all unique
	const numKeys = 100
	keys := make(map[string]bool)

	for i := 0; i < numKeys; i++ {
		key, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("Failed to generate API key %d: %v", i, err)
		}

		if keys[key] {
			t.Errorf("Duplicate API key generated: %s", key)
		}
		keys[key] = true
	}

	if len(keys) != numKeys {
		t.Errorf("Expected %d unique keys, got %d", numKeys, len(keys))
	}
}
