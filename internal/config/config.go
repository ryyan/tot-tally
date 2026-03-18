// config.go defines application configuration, settings, and static validation maps.
package config

import "time"

// Config holds all application settings.
type Config struct {
	Port           string
	NumShards      int
	TotDirectory   string
	LimitDirectory string
	MaxTallies     int
	MaxTotsPerIP   int
	TimeFormat     string
	CleanupAge     time.Duration
}

// NewDefaultConfig returns a standard configuration for the application.
func NewDefaultConfig() *Config {
	return &Config{
		Port:           ":5000",
		NumShards:      4096,
		TotDirectory:   "tots",
		LimitDirectory: "limits",
		MaxTallies:     100,
		MaxTotsPerIP:   10,
		TimeFormat:     "02 Jan 03:04PM",
		CleanupAge:     180 * 24 * time.Hour,
	}
}

var (
	// TallyKindMap defines the relationship between form IDs and emoji storage strings.
	TallyKindMap = map[int64]string{
		1: "🍼1", 2: "🍼2", 3: "🍼3", 4: "🍼4", 5: "🍼5", 6: "🍼6", 7: "🍼7", 8: "🍼8",
		9: "🍎", 10: "🍲", 11: "🚽", 12: "💩", 13: "🚽💩", 14: "🛁", 15: "🦷", 16: "🤱L", 17: "🤱R",
	}

	// Validation sets using the empty-struct idiom for zero-byte memory footprint.
	AllowedAvatars = map[string]struct{}{
		"👶": {}, "🧒": {}, "👦": {}, "👧": {}, "🐥": {}, "🧸": {}, "🦖": {}, "🐰": {},
		"🎀": {}, "👑": {}, "🚂": {}, "🦄": {}, "⭐": {}, "🚀": {}, "🌻": {}, "🐞": {},
		"🐘": {}, "🦒": {}, "🐼": {}, "🐝": {},
	}

	AllowedMilkSettings = map[string]struct{}{
		"bottle": {}, "nursing": {}, "both": {},
	}

	AllowedTimezones = map[string]struct{}{
		"Pacific/Honolulu": {}, "America/Anchorage": {}, "America/Los_Angeles": {},
		"America/Boise": {}, "America/Denver": {}, "America/Phoenix": {},
		"America/Chicago": {}, "America/Detroit": {}, "America/New_York": {},
	}

	// FlashMessages maps cookie keys to user-visible toast notifications.
	FlashMessages = map[string]string{
		"tally":            "Tally Added!",
		"undo":             "Tally Undone",
		"updated":          "Settings Updated",
		"deleted":          "Tot Deleted",
		"error_limit":      "Error: Too many requests!",
		"error_limit_ip":   "Error: Tot limit reached for this IP!",
		"error_not_found":  "Error: Tot not found!",
		"error_unexpected": "Error: Unexpected error!",
	}
)
