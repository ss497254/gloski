package system

import (
	"sync"
	"time"
)

// Store holds the latest system stats with thread-safe access.
type Store struct {
	mu        sync.RWMutex
	stats     *Stats
	updatedAt time.Time
}

// NewStore creates a new stats store.
func NewStore() *Store {
	return &Store{}
}

// Update stores new stats.
func (s *Store) Update(stats *Stats) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats = stats
	s.updatedAt = time.Now()
}

// Get returns the latest stats (nil if not yet collected).
func (s *Store) Get() *Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// UpdatedAt returns when stats were last updated.
func (s *Store) UpdatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.updatedAt
}
