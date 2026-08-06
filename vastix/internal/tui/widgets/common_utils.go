package widgets

import (
	"fmt"
	"vastix/internal/database"

	vast_client "github.com/vast-data/go-vast-client"
)

func getActiveRest(db *database.Service) (*vast_client.VMSRest, error) {
	profile, err := db.GetActiveProfile()
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, fmt.Errorf("no active profile found")
	}
	return profile.RestClientFromProfile()
}
