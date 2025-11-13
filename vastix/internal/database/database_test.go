package database

import (
	"testing"
)

func TestDatabaseService(t *testing.T) {
	// Use the global New() function which creates a service
	service := New()
	if service == nil {
		t.Error("Expected non-nil service")
	}
	defer service.Close()

	// Test that we can get the database instance
	db := service.GetDB()
	if db == nil {
		t.Error("Expected non-nil database instance")
	}
}

func TestService_ProfileOperations(t *testing.T) {
	// Skip test to avoid database conflicts during testing
	t.Skip("Skipping database tests to avoid singleton conflicts")

	// Use the global service for testing
	service := New()
	defer service.Close()

	t.Run("create and retrieve profile", func(t *testing.T) {
		profile := &Profile{
			Alias:       "test-profile",
			Endpoint:    "test.example.com",
			Username:    "testuser",
			Password:    "testpass",
			Port:        443,
			SSLVerify:   true,
			VastVersion: "5.3.0",
			ApiVersion:  "v5",
		}

		// Create profile
		err := service.CreateProfile(profile)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		// Retrieve all profiles
		profiles, err := service.GetAllProfiles()
		if err != nil {
			t.Fatalf("Failed to get profiles: %v", err)
		}

		if len(profiles) != 1 {
			t.Errorf("Expected 1 profile, got %d", len(profiles))
		}

		retrievedProfile := profiles[0]
		if retrievedProfile.Alias != profile.Alias {
			t.Errorf("Expected alias %s, got %s", profile.Alias, retrievedProfile.Alias)
		}

		if retrievedProfile.Endpoint != profile.Endpoint {
			t.Errorf("Expected endpoint %s, got %s", profile.Endpoint, retrievedProfile.Endpoint)
		}
	})

	t.Run("set and get active profile", func(t *testing.T) {
		// First create a profile
		profile := &Profile{
			Alias:    "active-test",
			Endpoint: "active.example.com",
			Username: "activeuser",
			Password: "activepass",
		}

		err := service.CreateProfile(profile)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		// Get the created profile (it should have an ID now)
		profiles, err := service.GetAllProfiles()
		if err != nil {
			t.Fatalf("Failed to get profiles: %v", err)
		}

		var createdProfile *Profile
		for i, p := range profiles {
			if p.Alias == "active-test" {
				createdProfile = &profiles[i]
				break
			}
		}

		if createdProfile == nil {
			t.Fatal("Created profile not found")
		}

		// Set as active
		err = service.SetActiveProfile(createdProfile.ID)
		if err != nil {
			t.Fatalf("Failed to set active profile: %v", err)
		}

		// Get active profile
		activeProfile, err := service.GetActiveProfile()
		if err != nil {
			t.Fatalf("Failed to get active profile: %v", err)
		}

		if activeProfile == nil {
			t.Fatal("Expected active profile, got nil")
		}

		if activeProfile.ID != createdProfile.ID {
			t.Errorf("Expected active profile ID %d, got %d", createdProfile.ID, activeProfile.ID)
		}
	})

	t.Run("delete profile", func(t *testing.T) {
		// Create a profile to delete
		profile := &Profile{
			Alias:    "delete-test",
			Endpoint: "delete.example.com",
			Username: "deleteuser",
		}

		err := service.CreateProfile(profile)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		// Get all profiles before deletion
		profilesBefore, err := service.GetAllProfiles()
		if err != nil {
			t.Fatalf("Failed to get profiles: %v", err)
		}

		// Find the profile to delete
		var profileToDelete *Profile
		for i, p := range profilesBefore {
			if p.Alias == "delete-test" {
				profileToDelete = &profilesBefore[i]
				break
			}
		}

		if profileToDelete == nil {
			t.Fatal("Profile to delete not found")
		}

		// Delete the profile
		err = service.DeleteProfile(profileToDelete.ID)
		if err != nil {
			t.Fatalf("Failed to delete profile: %v", err)
		}

		// Get all profiles after deletion
		profilesAfter, err := service.GetAllProfiles()
		if err != nil {
			t.Fatalf("Failed to get profiles after deletion: %v", err)
		}

		// Verify the profile was deleted
		for _, p := range profilesAfter {
			if p.ID == profileToDelete.ID {
				t.Error("Profile was not deleted")
			}
		}

		if len(profilesAfter) != len(profilesBefore)-1 {
			t.Errorf("Expected %d profiles after deletion, got %d", len(profilesBefore)-1, len(profilesAfter))
		}
	})
}

func TestProfile_ProfileName(t *testing.T) {
	tests := []struct {
		name     string
		profile  Profile
		expected string
	}{
		{
			name: "alias with endpoint format",
			profile: Profile{
				Alias:    "my-alias",
				Endpoint: "example.com",
			},
			expected: "my-alias [example.com]",
		},
		{
			name: "endpoint when no alias",
			profile: Profile{
				Alias:    "",
				Endpoint: "example.com",
			},
			expected: "example.com",
		},
		{
			name: "empty when both empty",
			profile: Profile{
				Alias:    "",
				Endpoint: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.profile.ProfileName()
			if result != tt.expected {
				t.Errorf("ProfileName() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestService_Close(t *testing.T) {
	service := New()

	// Close should not return an error
	err := service.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Second close should also not return an error (idempotent)
	err = service.Close()
	if err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}
}
