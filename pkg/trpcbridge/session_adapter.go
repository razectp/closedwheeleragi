package trpcbridge

import (
	"context"
	"fmt"
	"sync"

	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/session"
)

// SessionAdapter implements session.Service with an in-memory store.
// Only the methods required by runner.Runner are fully implemented;
// the rest are no-ops that satisfy the interface contract.
type SessionAdapter struct {
	sessions map[string]*session.Session
	mu       sync.RWMutex
}

// NewSessionAdapter creates a SessionAdapter backed by an in-memory map.
func NewSessionAdapter() *SessionAdapter {
	return &SessionAdapter{
		sessions: make(map[string]*session.Session),
	}
}

// sessionKey builds a unique key from the session.Key fields.
func sessionKey(key session.Key) string {
	return fmt.Sprintf("%s/%s/%s", key.AppName, key.UserID, key.SessionID)
}

// CreateSession creates a new session or returns an existing one.
func (sa *SessionAdapter) CreateSession(_ context.Context, key session.Key, state session.StateMap, _ ...session.Option) (*session.Session, error) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	k := sessionKey(key)
	if existing, ok := sa.sessions[k]; ok {
		return existing, nil
	}

	sess := session.NewSession(key.AppName, key.UserID, key.SessionID)
	if state != nil {
		for sk, sv := range state {
			sess.SetState(sk, sv)
		}
	}

	sa.sessions[k] = sess
	return sess, nil
}

// GetSession retrieves an existing session.
// Returns (nil, nil) when the session does not exist — the runner interprets
// this as "not found" and proceeds to call CreateSession.
func (sa *SessionAdapter) GetSession(_ context.Context, key session.Key, _ ...session.Option) (*session.Session, error) {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	k := sessionKey(key)
	sess, ok := sa.sessions[k]
	if !ok {
		return nil, nil
	}
	return sess, nil
}

// AppendEvent appends an event to the session.
func (sa *SessionAdapter) AppendEvent(_ context.Context, sess *session.Session, evt *event.Event, _ ...session.Option) error {
	if sess == nil || evt == nil {
		return nil
	}
	sess.UpdateUserSession(evt)
	return nil
}

// Close releases resources.
func (sa *SessionAdapter) Close() error {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.sessions = make(map[string]*session.Session)
	return nil
}

// --- Stub methods (satisfy session.Service, not needed for pipeline use) ---

// ListSessions returns an empty list.
func (sa *SessionAdapter) ListSessions(_ context.Context, _ session.UserKey, _ ...session.Option) ([]*session.Session, error) {
	return nil, nil
}

// DeleteSession is a no-op.
func (sa *SessionAdapter) DeleteSession(_ context.Context, key session.Key, _ ...session.Option) error {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	delete(sa.sessions, sessionKey(key))
	return nil
}

// UpdateAppState is a no-op.
func (sa *SessionAdapter) UpdateAppState(_ context.Context, _ string, _ session.StateMap) error {
	return nil
}

// DeleteAppState is a no-op.
func (sa *SessionAdapter) DeleteAppState(_ context.Context, _ string, _ string) error {
	return nil
}

// ListAppStates returns an empty map.
func (sa *SessionAdapter) ListAppStates(_ context.Context, _ string) (session.StateMap, error) {
	return nil, nil
}

// UpdateUserState is a no-op.
func (sa *SessionAdapter) UpdateUserState(_ context.Context, _ session.UserKey, _ session.StateMap) error {
	return nil
}

// ListUserStates returns an empty map.
func (sa *SessionAdapter) ListUserStates(_ context.Context, _ session.UserKey) (session.StateMap, error) {
	return nil, nil
}

// DeleteUserState is a no-op.
func (sa *SessionAdapter) DeleteUserState(_ context.Context, _ session.UserKey, _ string) error {
	return nil
}

// UpdateSessionState is a no-op.
func (sa *SessionAdapter) UpdateSessionState(_ context.Context, _ session.Key, _ session.StateMap) error {
	return nil
}

// CreateSessionSummary is a no-op.
func (sa *SessionAdapter) CreateSessionSummary(_ context.Context, _ *session.Session, _ string, _ bool) error {
	return nil
}

// EnqueueSummaryJob is a no-op.
func (sa *SessionAdapter) EnqueueSummaryJob(_ context.Context, _ *session.Session, _ string, _ bool) error {
	return nil
}

// GetSessionSummaryText returns empty.
func (sa *SessionAdapter) GetSessionSummaryText(_ context.Context, _ *session.Session, _ ...session.SummaryOption) (string, bool) {
	return "", false
}
