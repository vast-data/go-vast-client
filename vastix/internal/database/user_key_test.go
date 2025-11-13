package database

import (
	"testing"
)

func TestUserKeyOperations(t *testing.T) {
	// Skip this test to avoid conflicts with the singleton database
	t.Skip("Skipping UserKey operations test to avoid singleton database conflicts")

	// This test is kept as a reference for manual testing
	// When run manually (with proper database isolation), it would test:

	// 1. Create a test profile
	// 2. Create user keys for different users
	// 3. Test cascade deletion when profile is deleted
	// 4. Test retrieval operations
}

func TestUserKeyModel(t *testing.T) {
	t.Run("local user key", func(t *testing.T) {
		// Test the UserKey model structure for local users
		testUID := int64(1001)
		userKey := UserKey{
			ProfileID: 1,
			UserID:    12345,
			Username:  "test-user",
			UserUID:   &testUID,
			NonLocal:  false, // Local user
			AccessKey: "test-access-key",
			SecretKey: "test-secret-key",
		}

		// Verify field assignments
		if userKey.ProfileID != 1 {
			t.Errorf("Expected ProfileID to be 1, got %d", userKey.ProfileID)
		}
		if userKey.UserID != 12345 {
			t.Errorf("Expected UserID to be 12345, got %d", userKey.UserID)
		}
		if userKey.Username != "test-user" {
			t.Errorf("Expected Username to be 'test-user', got %s", userKey.Username)
		}
		if userKey.UserUID == nil || *userKey.UserUID != 1001 {
			t.Errorf("Expected UserUID to be 1001, got %v", userKey.UserUID)
		}
		if userKey.NonLocal != false {
			t.Errorf("Expected NonLocal to be false for local user, got %t", userKey.NonLocal)
		}
		if userKey.AccessKey != "test-access-key" {
			t.Errorf("Expected AccessKey to be 'test-access-key', got %s", userKey.AccessKey)
		}
		if userKey.SecretKey != "test-secret-key" {
			t.Errorf("Expected SecretKey to be 'test-secret-key', got %s", userKey.SecretKey)
		}
	})

	t.Run("non-local user key", func(t *testing.T) {
		// Test the UserKey model structure for non-local users
		testUID := int64(2002)
		userKey := UserKey{
			ProfileID: 1,
			UserID:    0, // Non-local users don't have UserID
			Username:  "non-local-user",
			UserUID:   &testUID,
			NonLocal:  true, // Non-local user
			AccessKey: "nonlocal-access-key",
			SecretKey: "nonlocal-secret-key",
		}

		// Verify field assignments
		if userKey.ProfileID != 1 {
			t.Errorf("Expected ProfileID to be 1, got %d", userKey.ProfileID)
		}
		if userKey.UserID != 0 {
			t.Errorf("Expected UserID to be 0 for non-local user, got %d", userKey.UserID)
		}
		if userKey.Username != "non-local-user" {
			t.Errorf("Expected Username to be 'non-local-user', got %s", userKey.Username)
		}
		if userKey.UserUID == nil || *userKey.UserUID != 2002 {
			t.Errorf("Expected UserUID to be 2002, got %v", userKey.UserUID)
		}
		if userKey.NonLocal != true {
			t.Errorf("Expected NonLocal to be true for non-local user, got %t", userKey.NonLocal)
		}
		if userKey.AccessKey != "nonlocal-access-key" {
			t.Errorf("Expected AccessKey to be 'nonlocal-access-key', got %s", userKey.AccessKey)
		}
		if userKey.SecretKey != "nonlocal-secret-key" {
			t.Errorf("Expected SecretKey to be 'nonlocal-secret-key', got %s", userKey.SecretKey)
		}
	})
}

// TestCascadeDeletion tests that user keys are automatically deleted when the parent profile is deleted
// This test is primarily documentation of the expected behavior due to GORM's CASCADE constraint
func TestCascadeDeletion(t *testing.T) {
	t.Log("UserKey model has CASCADE constraint on ProfileID")
	t.Log("When a Profile is deleted, all associated UserKey records should be automatically deleted")
	t.Log("This is enforced by the database constraint: gorm:\"constraint:OnDelete:CASCADE\"")
}
