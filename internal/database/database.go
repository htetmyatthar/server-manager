// This stores the private sessions and also public sessions for later use.
package data

import (
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/htetmyatthar/server-manager/internal/config"
)

var (
	ErrInvalidSession  = errors.New("Invalid session.")
	ErrSessionNotFound = errors.New("Session not found.")
	ErrSessionExpired  = errors.New("Session expired.")
)

type Session struct {
	Data      string
	ExpiresAt time.Time
}

// SessionStore defines the methods required for session management.
type SessionStore interface {
	// CreateSession creates a new session with the given ID and data that valid through the config.SessionDuration.
	// If the data string is ""(empty), the default user id is added using the sessionCount value of the session store.
	CreateSession(id string, data string) error

	// GetSession retrieves the session data for the given ID.
	GetSession(id string) (Session, error)

	// DeleteSession removes the session with the given ID.
	DeleteSession(id string) error

	// CleanupExpiredSessions removes all sessions that have expired.
	CleanupExpiredSessions() error
}

// MemSessionStore is an in-memory implementation of SessionShop
type MemSessionStore struct {
	sessions     map[string]Session
	mu           sync.RWMutex
	sessionCount int
}

// NewMemSessionStore initializes a new InMemorySessionStore.
func NewMemSessionStore() *MemSessionStore {
	store := &MemSessionStore{
		sessions:     make(map[string]Session),
		sessionCount: 0,
	}

	// Start the cleanup goroutine
	go store.periodicCleanup()

	return store
}

// CreateSession creates a new session. If the data string is ""(empty), the default
// user id is added using the sessionCount value of the session store.
func (store *MemSessionStore) CreateSession(id string, data string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	// NOTE: always overwrite the existing session.
	if data != "" {
		store.sessions[id] = Session{
			Data:      strconv.Itoa(store.sessionCount),
			ExpiresAt: time.Now().Add(time.Duration(*config.SessionDuration) * time.Minute),
		}
		store.sessionCount++
	} else {
		store.sessions[id] = Session{
			Data:      data,
			ExpiresAt: time.Now().Add(time.Duration(*config.SessionDuration) * time.Minute),
		}
	}

	return nil
}

// GetSession retrieves a session by ID returning error if the session is invalid or expired.
func (store *MemSessionStore) GetSession(id string) (Session, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	session, exists := store.sessions[id]
	if !exists {
		return Session{}, ErrSessionNotFound
	}

	// Check if session has expired
	if time.Now().After(session.ExpiresAt) {
		// Session expired, delete it.
		store.mu.Lock()
		delete(store.sessions, id)
		store.mu.Unlock()
		return Session{}, ErrSessionExpired
	}

	return session, nil
}

// DeleteSession removes a session by ID.
func (store *MemSessionStore) DeleteSession(id string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if _, exists := store.sessions[id]; !exists {
		return ErrSessionNotFound
	}

	delete(store.sessions, id)
	return nil
}

// CleanupExpiredSessions removes all expired sessions.
func (store *MemSessionStore) CleanupExpiredSessions() error {
	store.mu.Lock()
	defer store.mu.Unlock()

	now := time.Now()
	for id, session := range store.sessions {
		if now.After(session.ExpiresAt) {
			delete(store.sessions, id)
		}
	}

	return nil
}

// periodicCleanup runs CleanupExpiredSessions at regular intervals.
func (store *MemSessionStore) periodicCleanup() {
	// Interval is fourth of the configured session duration.
	var ticker *time.Ticker
	if *config.SessionDuration < 4 {
		ticker = time.NewTicker(time.Duration(*config.SessionDuration) * time.Minute)
	} else {
		ticker = time.NewTicker((time.Duration(*config.SessionDuration) / 4) * time.Minute)
	}
	defer ticker.Stop()

	for {
		<-ticker.C
		if err := store.CleanupExpiredSessions(); err != nil {
			log.Println("ERROR: Cleaning expired session gone wrong.")
			continue
		}
	}
}
