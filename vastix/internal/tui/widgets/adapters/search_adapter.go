package adapters

import (
	vast_client "github.com/vast-data/go-vast-client"
	"vastix/internal/database"
	log "vastix/internal/logging"
	"vastix/internal/tui/widgets/common"
)

// SearchAdapter manages search functionality for widgets
type SearchAdapter struct {
	resourceType string // Type of resource this adapter represents

	// Search functionality
	fuzzyListSearch      string // Local fuzzy search string for list
	fuzzyDetailsSearch   string // Local fuzzy search string for details
	serverSearchParamStr string // Server-side search parameters as string

	// Database connection
	db *database.Service
}

// NewSearchAdapter creates a new search adapter
func NewSearchAdapter(db *database.Service, resourceType string) *SearchAdapter {
	log.Debug("SearchAdapter initializing")

	adapter := &SearchAdapter{
		resourceType:         resourceType,
		fuzzyListSearch:      "",
		fuzzyDetailsSearch:   "",
		serverSearchParamStr: "",
		db:                   db,
	}

	log.Debug("SearchAdapter initialized successfully")
	return adapter
}

// SetFuzzyListSearchString sets the fuzzy search query for list filtering
func (sa *SearchAdapter) SetFuzzyListSearchString(query string) {
	sa.fuzzyListSearch = query
}

// GetFuzzyListSearchString returns the current fuzzy list search query
func (sa *SearchAdapter) GetFuzzyListSearchString() string {
	return sa.fuzzyListSearch
}

// SetFuzzyDetailsSearchString sets the fuzzy search query for details filtering
func (sa *SearchAdapter) SetFuzzyDetailsSearchString(query string) {
	sa.fuzzyDetailsSearch = query
}

// GetFuzzyDetailsSearchString returns the current fuzzy details search query
func (sa *SearchAdapter) GetFuzzyDetailsSearchString() string {
	return sa.fuzzyDetailsSearch
}

// SetServerSearchParams sets the server-side search parameters string
func (sa *SearchAdapter) SetServerSearchParams(paramStr string) {
	sa.serverSearchParamStr = paramStr
}

// GetServerSearchParams returns the current server-side search parameters as string
func (sa *SearchAdapter) GetServerSearchParams() string {
	return sa.serverSearchParamStr
}

// ClearFilters removes any active filters
func (sa *SearchAdapter) ClearFilters() {
	sa.ClearServerSearchParams()
	sa.ClearFuzzyListSearch()
	sa.ClearFuzzyDetailsSearch()
}

// ClearFuzzyListSearch clears the local fuzzy list search string
func (sa *SearchAdapter) ClearFuzzyListSearch() {
	sa.fuzzyListSearch = ""
}

func (sa *SearchAdapter) ClearFuzzyDetailsSearch() {
	sa.fuzzyDetailsSearch = ""
}

func (sa *SearchAdapter) ClearServerSearchParams() {
	sa.serverSearchParamStr = ""
}

func (sa *SearchAdapter) GetServerParams() *vast_client.Params {
	if sa.serverSearchParamStr == "" {
		return nil
	}
	params, _ := common.ConvertServerParamsToVastParams(sa.serverSearchParamStr)
	return &params
}
