package system

import (
	"sync"
	"time"
)

// Sample represents a single stats sample with timestamp.
type Sample struct {
	Stats     *Stats    `json:"stats"`
	Timestamp time.Time `json:"timestamp"`
}

// Store holds system stats history with thread-safe access using a ring buffer.
type Store struct {
	mu       sync.RWMutex
	samples  []Sample
	capacity int
	head     int // Next write position
	count    int // Current number of samples
}

// NewStore creates a new stats store with the given capacity.
// Capacity determines how many samples are retained.
// Example: capacity=300 with 2s interval = 10 minutes of history.
func NewStore(capacity int) *Store {
	if capacity <= 0 {
		capacity = 300 // Default: 10 minutes at 2s interval
	}
	return &Store{
		samples:  make([]Sample, capacity),
		capacity: capacity,
	}
}

// Push adds a new stats sample to the store.
func (s *Store) Push(stats *Stats) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.samples[s.head] = Sample{
		Stats:     stats,
		Timestamp: time.Now(),
	}

	s.head = (s.head + 1) % s.capacity
	if s.count < s.capacity {
		s.count++
	}
}

// Get returns the latest stats (nil if no samples yet).
func (s *Store) Get() *Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 {
		return nil
	}

	// Latest sample is at head-1 (wrapping around)
	idx := (s.head - 1 + s.capacity) % s.capacity
	return s.samples[idx].Stats
}

// GetLatest returns the latest sample with timestamp.
func (s *Store) GetLatest() *Sample {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 {
		return nil
	}

	idx := (s.head - 1 + s.capacity) % s.capacity
	sample := s.samples[idx]
	return &sample
}

// GetHistory returns samples from the last duration.
// Returns samples in chronological order (oldest first).
func (s *Store) GetHistory(duration time.Duration) []Sample {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 {
		return nil
	}

	cutoff := time.Now().Add(-duration)
	result := make([]Sample, 0, s.count)

	// Start from oldest sample
	start := 0
	if s.count == s.capacity {
		start = s.head // Oldest is at head when buffer is full
	}

	for i := 0; i < s.count; i++ {
		idx := (start + i) % s.capacity
		sample := s.samples[idx]
		if sample.Timestamp.After(cutoff) {
			result = append(result, sample)
		}
	}

	return result
}

// GetAll returns all stored samples in chronological order (oldest first).
func (s *Store) GetAll() []Sample {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 {
		return nil
	}

	result := make([]Sample, s.count)

	// Start from oldest sample
	start := 0
	if s.count == s.capacity {
		start = s.head
	}

	for i := 0; i < s.count; i++ {
		idx := (start + i) % s.capacity
		result[i] = s.samples[idx]
	}

	return result
}

// Count returns the current number of samples in the store.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

// Capacity returns the maximum number of samples the store can hold.
func (s *Store) Capacity() int {
	return s.capacity
}
