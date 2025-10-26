package tui

import "time"

// Ticker intervals for data updates
const (
	// DataTickerInterval defines how often to refresh widget data
	DataTickerInterval = 25 * time.Second

	// ProfileTickerInterval defines how often to update profile information
	ProfileTickerInterval = 5 * time.Minute

	// ThrottleInterval defines the minimum time between SetListData calls for the same widget
	ThrottleInterval = 20 * time.Second
)
