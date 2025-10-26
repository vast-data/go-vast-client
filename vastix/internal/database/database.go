package database

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	vastixlog "vastix/internal/logging"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Service struct {
	db         *gorm.DB
	profileMux sync.Mutex // Protects profile activation operations
}

var (
	dbInstance *Service
)

func New() *Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed to get user home directory: %v", err)
	}

	// Create .vastix directory in user home
	vastixDir := filepath.Join(homeDir, ".vastix")
	if err := os.MkdirAll(vastixDir, 0755); err != nil {
		log.Fatalf("failed to create .vastix directory: %v", err)
	}

	// Database file path
	dbPath := filepath.Join(vastixDir, "store.sqlite")

	conn := sqlite.Open(dbPath)
	db, err := gorm.Open(conn, &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Auto-migrate models
	if err := db.AutoMigrate(&Profile{}, &ResourceHistory{}, &UserKey{}, &ApiToken{}, &SshConnection{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Create default "local" SSH connection if it doesn't exist
	if err := createDefaultLocalSshConnection(db); err != nil {
		log.Fatalf("failed to create default local SSH connection: %v", err)
	}

	dbInstance = &Service{
		db: db,
	}
	return dbInstance
}

// createDefaultLocalSshConnection creates the default "local [pseudo ssh]" SSH connection if it doesn't exist
func createDefaultLocalSshConnection(db *gorm.DB) error {
	var count int64
	if err := db.Model(&SshConnection{}).Where("name = ?", "local [pseudo ssh]").Count(&count).Error; err != nil {
		return err
	}

	// If "local [pseudo ssh]" connection doesn't exist, create it
	if count == 0 {
		localConn := &SshConnection{
			Name:        "local [pseudo ssh]",
			SshHost:     "-",
			SshUserName: "-",
			SshPassword: "-",
			SshKey:      "-",
			SshPort:     22, // Keep default port structure
		}
		return db.Create(localConn).Error
	}
	return nil
}

// GetDB returns the database instance
func (s *Service) GetDB() *gorm.DB {
	return s.db
}

// Close closes the database connection
func (s *Service) Close() error {
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// Profile operations

// CreateProfile creates a new profile in the database
func (s *Service) CreateProfile(profile *Profile) error {
	vastixlog.Debug("Creating profile in database",
		zap.String("endpoint", profile.Endpoint),
		zap.Bool("ssl_verify_before_create", profile.SSLVerify))

	err := s.db.Create(profile).Error

	vastixlog.Debug("Profile created in database",
		zap.Uint("id", profile.ID),
		zap.Bool("ssl_verify_after_create", profile.SSLVerify),
		zap.Error(err))

	return err
}

// CreateProfileAsActive creates a new profile and sets it as active, deactivating all others
func (s *Service) CreateProfileAsActive(profile *Profile) error {
	// Lock to prevent concurrent profile activation
	s.profileMux.Lock()
	defer s.profileMux.Unlock()

	// Start a transaction to ensure consistency
	tx := s.db.Begin()

	// First, deactivate all existing profiles
	if err := tx.Model(&Profile{}).Where("active = ?", true).Update("active", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Set the new profile as active
	profile.Active = true

	// Create the new profile
	if err := tx.Create(profile).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	return tx.Commit().Error
}

// GetProfile retrieves a profile by ID
func (s *Service) GetProfile(id uint64) (*Profile, error) {
	var profile Profile
	err := s.db.First(&profile, id).Error
	return &profile, err
}

// GetAllProfiles retrieves all profiles from the database
func (s *Service) GetAllProfiles() ([]Profile, error) {
	var profiles []Profile
	err := s.db.Find(&profiles).Error
	return profiles, err
}

// UpdateProfile updates an existing profile
func (s *Service) UpdateProfile(profile *Profile) error {
	return s.db.Save(profile).Error
}

// DeleteProfile deletes a profile by ID
func (s *Service) DeleteProfile(id uint) error {
	return s.db.Delete(&Profile{}, id).Error
}

// GetProfileByEndpoint retrieves a profile by endpoint
func (s *Service) GetProfileByEndpoint(endpoint string) (*Profile, error) {
	var profile Profile
	err := s.db.Where("endpoint = ?", endpoint).First(&profile).Error
	return &profile, err
}

// Active Profile operations

// SetActiveProfile sets a profile as active and deactivates all others
func (s *Service) SetActiveProfile(id uint) error {
	// Lock to prevent concurrent profile activation
	s.profileMux.Lock()
	defer s.profileMux.Unlock()

	// Start a transaction to ensure consistency
	tx := s.db.Begin()

	// First, deactivate all profiles
	if err := tx.Model(&Profile{}).Where("active = ?", true).Update("active", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Then, activate the specified profile
	if err := tx.Model(&Profile{}).Where("id = ?", id).Update("active", true).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	return tx.Commit().Error
}

// GetActiveProfile retrieves the currently active profile
func (s *Service) GetActiveProfile() (*Profile, error) {
	var profile Profile
	err := s.db.Where("active = ?", true).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No active profile found - return nil profile with no error
			return nil, nil
		}
		// Other database error - return nil profile with error
		return nil, err
	}
	return &profile, nil
}

// DeactivateAllProfiles deactivates all profiles
func (s *Service) DeactivateAllProfiles() error {
	return s.db.Model(&Profile{}).Where("active = ?", true).Update("active", false).Error
}

// EnsureSingleActiveProfile fixes data corruption by ensuring only one profile is active
// If multiple profiles are active, it keeps the most recently updated one
func (s *Service) EnsureSingleActiveProfile() error {
	var activeProfiles []Profile
	err := s.db.Where("active = ?", true).Order("updated_at DESC").Find(&activeProfiles).Error
	if err != nil {
		return fmt.Errorf("failed to query active profiles: %w", err)
	}

	if len(activeProfiles) <= 1 {
		// 0 or 1 active profiles is fine
		return nil
	}

	// Multiple active profiles found - this is the bug we're fixing
	vastixlog.Warn("Multiple active profiles detected, fixing data corruption",
		zap.Int("count", len(activeProfiles)))

	// Start a transaction to ensure consistency
	tx := s.db.Begin()

	// Deactivate all profiles first
	if err := tx.Model(&Profile{}).Where("active = ?", true).Update("active", false).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to deactivate all profiles: %w", err)
	}

	// Activate only the most recently updated one (first in our DESC ordered list)
	if len(activeProfiles) > 0 {
		if err := tx.Model(&Profile{}).Where("id = ?", activeProfiles[0].ID).Update("active", true).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to reactivate primary profile: %w", err)
		}

		vastixlog.Info("Fixed multiple active profiles, kept most recent",
			zap.Uint("kept_profile_id", activeProfiles[0].ID),
			zap.String("endpoint", activeProfiles[0].Endpoint),
			zap.Int("deactivated_count", len(activeProfiles)-1))
	}

	return tx.Commit().Error
}

// HasActiveProfile checks if there is an active profile
func (s *Service) HasActiveProfile() (bool, error) {
	var count int64
	err := s.db.Model(&Profile{}).Where("active = ?", true).Count(&count).Error
	return count > 0, err
}

// Resource History operations

// GetCurrentResourceHistory returns the single resource history record
func (s *Service) GetCurrentResourceHistory() (*ResourceHistory, error) {
	var history ResourceHistory
	err := s.db.First(&history).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No history record found
		}
		return nil, err
	}
	return &history, nil
}

// SetResourceHistory updates the single resource history record (upsert pattern)
func (s *Service) SetResourceHistory(currentResource, previousResource string) error {
	vastixlog.Debug("Setting resource history",
		zap.String("current", currentResource),
		zap.String("previous", previousResource))

	// Try to get existing record
	existingHistory, err := s.GetCurrentResourceHistory()
	if err != nil {
		return err
	}

	if existingHistory == nil {
		// No record exists, create new one
		history := &ResourceHistory{
			CurrentResource:  currentResource,
			PreviousResource: previousResource,
		}

		err := s.db.Create(history).Error
		if err != nil {
			vastixlog.Debug("Failed to create resource history", zap.Error(err))
			return err
		}

		vastixlog.Debug("Resource history created",
			zap.String("current", currentResource),
			zap.String("previous", previousResource))
	} else {
		// Record exists, update it
		err := s.db.Model(existingHistory).Updates(ResourceHistory{
			CurrentResource:  currentResource,
			PreviousResource: previousResource,
		}).Error

		if err != nil {
			vastixlog.Debug("Failed to update resource history", zap.Error(err))
			return err
		}

		vastixlog.Debug("Resource history updated",
			zap.String("current", currentResource),
			zap.String("previous", previousResource))
	}

	return nil
}

// GetCurrentResource returns just the current active resource type
func (s *Service) GetCurrentResource() (string, error) {
	history, err := s.GetCurrentResourceHistory()
	if err != nil {
		return "", err
	}
	if history == nil {
		return "views", nil // Default resource if no history exists
	}
	return history.CurrentResource, nil
}

// GetPreviousResource returns just the previous resource type
func (s *Service) GetPreviousResource() (string, error) {
	history, err := s.GetCurrentResourceHistory()
	if err != nil {
		return "", err
	}
	if history == nil {
		return "", nil // No previous resource if no history exists
	}
	return history.PreviousResource, nil
}

// InitializeResourceHistory creates initial resource history record if none exists
func (s *Service) InitializeResourceHistory(defaultResource string) error {
	history, err := s.GetCurrentResourceHistory()
	if err != nil {
		return err
	}

	// If no history exists, create initial record
	if history == nil {
		return s.SetResourceHistory(defaultResource, "")
	}

	return nil
}

// UserKey operations

// CreateUserKey creates a new user key in the database
func (s *Service) CreateUserKey(profileID uint, userID int64, username string, userUID int64, accessKey, secretKey string, nonLocal bool) (*UserKey, error) {
	userKey := &UserKey{
		ProfileID: profileID,
		UserID:    userID, // Will be 0 for non-local users
		Username:  username,
		UserUID:   &userUID,
		NonLocal:  nonLocal,
		AccessKey: accessKey,
		SecretKey: secretKey,
	}

	err := s.db.Create(userKey).Error
	if err != nil {
		return nil, err
	}

	return userKey, nil
}

// CreateNonLocalUserKey creates a new user key for a non-local user
// Non-local users don't have UserID, so it will be set to 0
func (s *Service) CreateNonLocalUserKey(profileID uint, username string, userUID int64, accessKey, secretKey string) (*UserKey, error) {
	return s.CreateUserKey(profileID, 0, username, userUID, accessKey, secretKey, true)
}

// CreateLocalUserKey creates a new user key for a local user (convenience function)
func (s *Service) CreateLocalUserKey(profileID uint, userID int64, username string, userUID int64, accessKey, secretKey string) (*UserKey, error) {
	return s.CreateUserKey(profileID, userID, username, userUID, accessKey, secretKey, false)
}

// GetUserKeysByProfile retrieves all user keys for a specific profile
func (s *Service) GetUserKeysByProfile(profileID uint) ([]UserKey, error) {
	var userKeys []UserKey
	err := s.db.Where("profile_id = ?", profileID).Find(&userKeys).Error
	return userKeys, err
}

// GetUserKeysByUser retrieves all user keys for a specific user across all profiles
func (s *Service) GetUserKeysByUser(userID int64) ([]UserKey, error) {
	var userKeys []UserKey
	err := s.db.Preload("Profile").Where("user_id = ?", userID).Find(&userKeys).Error
	return userKeys, err
}

// GetUserKey retrieves a specific user key by ID
func (s *Service) GetUserKey(id uint) (*UserKey, error) {
	var userKey UserKey
	err := s.db.Preload("Profile").First(&userKey, id).Error
	return &userKey, err
}

// DeleteUserKey deletes a user key by ID
func (s *Service) DeleteUserKey(id uint) error {
	return s.db.Delete(&UserKey{}, id).Error
}

// GetLocalUserKeysByProfile retrieves all local user keys for a specific profile
func (s *Service) GetLocalUserKeysByProfile(profileID uint) ([]UserKey, error) {
	var userKeys []UserKey
	err := s.db.Where("profile_id = ? AND non_local = ?", profileID, false).Find(&userKeys).Error
	return userKeys, err
}

// GetNonLocalUserKeysByProfile retrieves all non-local user keys for a specific profile
func (s *Service) GetNonLocalUserKeysByProfile(profileID uint) ([]UserKey, error) {
	var userKeys []UserKey
	err := s.db.Where("profile_id = ? AND non_local = ?", profileID, true).Find(&userKeys).Error
	return userKeys, err
}

// GetUserKeysForActiveProfile retrieves all user keys for the currently active profile
func (s *Service) GetUserKeysForActiveProfile() ([]UserKey, error) {
	// Get the active profile first
	activeProfile, err := s.GetActiveProfile()
	if err != nil || activeProfile == nil {
		return nil, err
	}

	return s.GetUserKeysByProfile(activeProfile.ID)
}

// GetLocalUserKeysForActiveProfile retrieves all local user keys for the currently active profile
func (s *Service) GetLocalUserKeysForActiveProfile() ([]UserKey, error) {
	activeProfile, err := s.GetActiveProfile()
	if err != nil || activeProfile == nil {
		return nil, err
	}

	return s.GetLocalUserKeysByProfile(activeProfile.ID)
}

// GetNonLocalUserKeysForActiveProfile retrieves all non-local user keys for the currently active profile
func (s *Service) GetNonLocalUserKeysForActiveProfile() ([]UserKey, error) {
	activeProfile, err := s.GetActiveProfile()
	if err != nil || activeProfile == nil {
		return nil, err
	}

	return s.GetNonLocalUserKeysByProfile(activeProfile.ID)
}

// ApiToken operations

// CreateApiToken creates a new API token record in the database
func (s *Service) CreateApiToken(apiToken *ApiToken) error {
	return s.db.Create(apiToken).Error
}

// GetApiTokensByProfile retrieves all API tokens for a specific profile
func (s *Service) GetApiTokensByProfile(profileID uint) ([]ApiToken, error) {
	var apiTokens []ApiToken
	err := s.db.Preload("Profile").Where("profile_id = ?", profileID).Find(&apiTokens).Error
	return apiTokens, err
}

// GetApiTokensForActiveProfile retrieves all API tokens for the currently active profile
func (s *Service) GetApiTokensForActiveProfile() ([]ApiToken, error) {
	// Get the active profile first
	activeProfile, err := s.GetActiveProfile()
	if err != nil || activeProfile == nil {
		return nil, err
	}

	return s.GetApiTokensByProfile(activeProfile.ID)
}

// GetApiToken retrieves a specific API token by ID
func (s *Service) GetApiToken(id uint) (*ApiToken, error) {
	var apiToken ApiToken
	err := s.db.Preload("Profile").First(&apiToken, id).Error
	return &apiToken, err
}

// GetApiTokenByTokenID retrieves an API token by its VAST token ID
func (s *Service) GetApiTokenByTokenID(tokenID string, profileID uint) (*ApiToken, error) {
	var apiToken ApiToken
	err := s.db.Preload("Profile").Where("token_id = ? AND profile_id = ?", tokenID, profileID).First(&apiToken).Error
	return &apiToken, err
}

// DeleteApiToken deletes an API token by ID
func (s *Service) DeleteApiToken(id uint) error {
	return s.db.Delete(&ApiToken{}, id).Error
}

// SSH Connection operations

// CreateSshConnection creates a new SSH connection in the database
func (s *Service) CreateSshConnection(sshConn *SshConnection) error {
	return s.db.Create(sshConn).Error
}

// GetAllSshConnections retrieves all SSH connections from the database
func (s *Service) GetAllSshConnections() ([]SshConnection, error) {
	var connections []SshConnection
	err := s.db.Find(&connections).Error
	return connections, err
}

// GetSshConnection retrieves a specific SSH connection by ID
func (s *Service) GetSshConnection(id uint) (*SshConnection, error) {
	var connection SshConnection
	err := s.db.First(&connection, id).Error
	if err != nil {
		return nil, err
	}
	return &connection, nil
}

// UpdateSshConnection updates an existing SSH connection
func (s *Service) UpdateSshConnection(sshConn *SshConnection) error {
	return s.db.Save(sshConn).Error
}

// DeleteSshConnection deletes an SSH connection by ID
func (s *Service) DeleteSshConnection(id uint) error {
	return s.db.Delete(&SshConnection{}, id).Error
}
