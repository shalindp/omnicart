package common

import (
	"context"
	"fmt"
	"sync"
	"time"

	"onion.api/infrastrucre/common/responses"
)

// StoredSession represents a retailer session persisted between process restarts.
type StoredSession struct {
	Token     string
	ExpiresAt *time.Time
}

// SessionStore is somewhere durable to keep a retailer session between process restarts.
type SessionStore interface {
	Load(context context.Context, chain responses.StoreChain) (*StoredSession, error)
	Save(context context.Context, chain responses.StoreChain, session StoredSession) error
	Clear(context context.Context, chain responses.StoreChain) error
}

// RetailSession is a retailer session owned by the throttled client:
// minted, refreshed and invalidated by the pump rather than by each caller.
type RetailSession interface {
	Prime() error
	MintRequest() RetailClientRequest
	TryAccept(response RetailClientResponse) bool
	Headers(currentTime time.Time) map[string][]string
	IsExpired(response RetailClientResponse) bool
	Invalidate()
}

// NoOpSession is a nil session that always returns nil headers.
type NoOpSession struct{}

func (NoOpSession) Prime() error                                { return nil }
func (NoOpSession) MintRequest() RetailClientRequest         { return RetailClientRequest{} }
func (NoOpSession) TryAccept(RetailClientResponse) bool      { return false }
func (NoOpSession) Headers(time.Time) map[string][]string        { return map[string][]string{} }
func (NoOpSession) IsExpired(RetailClientResponse) bool       { return false }
func (NoOpSession) Invalidate()                                  {}

// Compile-time check.
var _ RetailSession = NoOpSession{}

// InMemorySessionStore is a simple in-memory session store for testing.
type InMemorySessionStore struct {
	mutex    sync.Mutex
	sessions map[responses.StoreChain]StoredSession
}

// NewInMemorySessionStore creates a new in-memory session store.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{sessions: make(map[responses.StoreChain]StoredSession)}
}

// Load retrieves a session from the store.
func (store *InMemorySessionStore) Load(_ context.Context, chain responses.StoreChain) (*StoredSession, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	session, exists := store.sessions[chain]
	if !exists {
		return nil, nil
	}
	return &session, nil
}

// Save stores a session in the store.
func (store *InMemorySessionStore) Save(_ context.Context, chain responses.StoreChain, session StoredSession) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.sessions[chain] = session
	return nil
}

// Clear removes a session from the store.
func (store *InMemorySessionStore) Clear(_ context.Context, chain responses.StoreChain) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	delete(store.sessions, chain)
	return nil
}

// Compile-time check.
var _ SessionStore = (*InMemorySessionStore)(nil)

// UnimplementedSessionStore is a placeholder that always errors.
type UnimplementedSessionStore struct{}

// Load returns an error.
func (UnimplementedSessionStore) Load(context.Context, responses.StoreChain) (*StoredSession, error) {
	return nil, fmt.Errorf("session store not implemented")
}

// Save returns an error.
func (UnimplementedSessionStore) Save(context.Context, responses.StoreChain, StoredSession) error {
	return fmt.Errorf("session store not implemented")
}

// Clear returns an error.
func (UnimplementedSessionStore) Clear(context.Context, responses.StoreChain) error {
	return fmt.Errorf("session store not implemented")
}
