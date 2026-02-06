package universe

import (
	"sort"
	"sync"
	"time"
)

// Manager manages all discovered universes
type Manager struct {
	universes map[uint16]*Universe
	mu        sync.RWMutex
}

// NewManager creates a new universe manager
func NewManager() *Manager {
	return &Manager{
		universes: make(map[uint16]*Universe),
	}
}

// GetOrCreate returns the universe with the given ID, creating it if it doesn't exist
func (m *Manager) GetOrCreate(id uint16) *Universe {
	m.mu.Lock()
	defer m.mu.Unlock()

	if u, exists := m.universes[id]; exists {
		return u
	}

	u := NewUniverse(id)
	m.universes[id] = u
	return u
}

// Get returns the universe with the given ID, or nil if it doesn't exist
func (m *Manager) Get(id uint16) *Universe {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.universes[id]
}

// GetAll returns all universes sorted by ID
func (m *Manager) GetAll() []*Universe {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Universe, 0, len(m.universes))
	for _, u := range m.universes {
		result = append(result, u)
	}

	// Sort by universe ID
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result
}

// GetActiveUniverses returns all universes that have received data within the timeout
func (m *Manager) GetActiveUniverses(timeout time.Duration) []*Universe {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Universe, 0, len(m.universes))
	for _, u := range m.universes {
		if !u.IsStale(timeout) {
			result = append(result, u)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result
}

// Count returns the number of known universes
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.universes)
}

// Remove removes a universe by ID
func (m *Manager) Remove(id uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.universes, id)
}

// PruneStale removes all universes that haven't received data within the timeout
func (m *Manager) PruneStale(timeout time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	pruned := 0
	for id, u := range m.universes {
		if u.IsStale(timeout) {
			delete(m.universes, id)
			pruned++
		}
	}
	return pruned
}
