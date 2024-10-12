// This stores the private sessions and also public sessions for later use.
package data

import "errors"

var (
	// SessionStore stores the session in the own map for simplicity.
	// NOTE: How the session will clear? Use the cron job to restart the server daily.
	SessionStore map[string]string = make(map[string]string, 5)

	// SessionCount keeps track of the number of sessions.
	// Also acts as an id number for sessions.
	SessionCount = 0

	ErrInvalidSession = errors.New("Invalid session.")
)
