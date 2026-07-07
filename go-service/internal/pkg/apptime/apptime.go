// Package apptime centralizes the application's configured time zone.
package apptime

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	_ "time/tzdata"
)

const DefaultTimeZone = "Asia/Shanghai"

var (
	mu   sync.RWMutex
	name = DefaultTimeZone
	loc  = mustLoad(DefaultTimeZone)
)

// Configure loads the configured IANA time zone and makes it Go's local time.
func Configure(timeZone string) error {
	timeZone = strings.TrimSpace(timeZone)
	if timeZone == "" {
		timeZone = DefaultTimeZone
	}

	loaded, err := time.LoadLocation(timeZone)
	if err != nil {
		return fmt.Errorf("invalid app.timezone %q: %w", timeZone, err)
	}

	mu.Lock()
	name = timeZone
	loc = loaded
	time.Local = loaded
	_ = os.Setenv("TZ", timeZone)
	mu.Unlock()
	return nil
}

// Name returns the configured IANA time zone name.
func Name() string {
	mu.RLock()
	defer mu.RUnlock()
	return name
}

// Location returns the configured time.Location.
func Location() *time.Location {
	mu.RLock()
	defer mu.RUnlock()
	return loc
}

// Now returns the current time in the configured application time zone.
func Now() time.Time {
	return time.Now().In(Location())
}

// FormatRFC3339 formats a time in the configured application time zone.
func FormatRFC3339(t time.Time) string {
	return t.In(Location()).Format(time.RFC3339)
}

func mustLoad(timeZone string) *time.Location {
	loaded, err := time.LoadLocation(timeZone)
	if err != nil {
		return time.Local
	}
	return loaded
}
