package widgets

import (
	"fmt"
	"vastix/internal/database"
	log "vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
	vast_client "github.com/vast-data/go-vast-client"
	"go.uber.org/zap"
)

// getRecordKeys returns all keys from a record for debugging
func getRecordKeys(record vast_client.Record) []string {
	keys := make([]string, 0, len(record))
	for k := range record {
		keys = append(keys, k)
	}
	return keys
}

// createUserAccessKeyAfterCreateCallback creates a callback function that stores
// newly created user access keys in the local database
// The isNonLocal parameter indicates whether this callback is for non-local users
func createUserAccessKeyAfterCreateCallback(db *database.Service, isNonLocal bool) common.AfterCreateFunc {
	return func(record vast_client.Record, parentRowData common.RowData) (tea.Msg, error) {
		keyType := "local"
		if isNonLocal {
			keyType = "non-local"
		}

		// Get the active profile
		activeProfile, err := db.GetActiveProfile()
		if err != nil {
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("failed to get active profile: %w", err),
			}, nil
		}
		if activeProfile == nil {
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("no active profile found"),
			}, nil
		}

		// Extract user information from the record
		// The record should contain the newly created access key details
		var accessKey, secretKey string
		var userId int64
		var username string
		var userUid *int64

		// Extract access_key
		if ak, exists := record["access_key"]; exists {
			accessKey = fmt.Sprintf("%v", ak)
		}

		// Extract secret_key
		if sk, exists := record["secret_key"]; exists {
			secretKey = fmt.Sprintf("%v", sk)
		}

		// Extract user_id from the record
		// The API response uses "id" field, not "user_id"
		// For non-local users, this field might not exist (userId will remain 0)
		if uid, exists := record["id"]; exists {
			switch v := uid.(type) {
			case int64:
				userId = v
			case float64:
				userId = int64(v)
			case int:
				userId = int64(v)
			}
			log.GetAuxLogger().Printf("[UserKeyCallback] Found id: %d", userId)
		} else if uid, exists := record["user_id"]; exists {
			// Fallback to user_id if id doesn't exist
			switch v := uid.(type) {
			case int64:
				userId = v
			case float64:
				userId = int64(v)
			case int:
				userId = int64(v)
			}
		}

		// Extract username from parent row data (users widget has "name" column)
		// The API response doesn't include username, so we get it from the selected user row
		if parentRowData.Len() > 0 {
			if name := parentRowData.Get("name"); name != nil {
				username = fmt.Sprintf("%v", name)
				log.GetAuxLogger().Printf("[UserKeyCallback] Found username from parent row: %s", username)
			} else {
				log.GetAuxLogger().Printf("[UserKeyCallback] No 'name' field in parent row data")
			}
		} else {
			// Fallback: try to get from API response (though it usually doesn't include it)
			if name, exists := record["username"]; exists {
				username = fmt.Sprintf("%v", name)
			} else if name, exists := record["name"]; exists {
				username = fmt.Sprintf("%v", name)
			}
		}

		// Extract user UID if available
		if uidVal, exists := record["uid"]; exists {
			switch v := uidVal.(type) {
			case int64:
				userUid = &v
			case float64:
				val := int64(v)
				userUid = &val
			case int:
				val := int64(v)
				userUid = &val
			}
		}

		// Store the keys in the database if we have the required information
		// For non-local users, userId might be 0, but that's okay
		if accessKey != "" && secretKey != "" {
			uidValue := int64(0)
			if userUid != nil {
				uidValue = *userUid
			}

			// Use the isNonLocal parameter passed to the callback factory
			// This is more reliable than trying to infer from the response data
			log.GetAuxLogger().Printf("[UserKeyCallback] Attempting to store key: userID=%d, username=%s, uid=%d, isNonLocal=%v",
				userId, username, uidValue, isNonLocal)

			// Use the generic CreateUserKey method that handles both local and non-local users
			_, err = db.CreateUserKey(activeProfile.ID, userId, username, uidValue, accessKey, secretKey, isNonLocal)
			if err != nil {
				log.GetAuxLogger().Printf("[UserKeyCallback] Failed to store %s user key: %v", keyType, err)
				return msg_types.ErrorMsg{
					Err: fmt.Errorf("failed to store %s user key in database: %w", keyType, err),
				}, nil
			}

			// Log success
			log.Info("Successfully stored user key in database",
				zap.String("userType", keyType),
				zap.Int64("userId", userId),
				zap.String("username", username))
		} else {
			// Return error if we don't have enough information
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("insufficient information to store user key: accessKey=%v, secretKey=%v",
					accessKey != "", secretKey != ""),
			}, nil
		}

		return msg_types.SetDataMsg{UseSpinner: false}, nil
	}
}
