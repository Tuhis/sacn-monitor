package stats

import (
	"sync"
	"time"
)

// Source represents a unique sACN source
type Source struct {
	CID          [16]byte
	Name         string
	LastSequence uint8
	LastSeen     time.Time
	PacketCount  uint64
	LostPackets  uint64
}

// UniverseStats tracks statistics for a single universe
type UniverseStats struct {
	UniverseID      uint16
	Sources         map[[16]byte]*Source
	PacketCount     uint64
	LostPackets     uint64
	LastPacket      time.Time
	packetsInWindow []time.Time // For rate calculation
	mu              sync.RWMutex
}

// Tracker tracks packet statistics for all universes
type Tracker struct {
	universes  map[uint16]*UniverseStats
	rateWindow time.Duration
	mu         sync.RWMutex
}

// NewTracker creates a new stats tracker
func NewTracker() *Tracker {
	return &Tracker{
		universes:  make(map[uint16]*UniverseStats),
		rateWindow: time.Second, // Calculate rate over 1 second window
	}
}

// RecordPacket records a packet for statistics tracking
func (t *Tracker) RecordPacket(universeID uint16, sourceCID [16]byte, sourceName string, sequence uint8) {
	t.mu.Lock()
	stats, exists := t.universes[universeID]
	if !exists {
		stats = &UniverseStats{
			UniverseID: universeID,
			Sources:    make(map[[16]byte]*Source),
		}
		t.universes[universeID] = stats
	}
	t.mu.Unlock()

	stats.mu.Lock()
	defer stats.mu.Unlock()

	now := time.Now()
	stats.PacketCount++
	stats.LastPacket = now

	// Add to rate window
	stats.packetsInWindow = append(stats.packetsInWindow, now)

	// Clean old packets from window
	cutoff := now.Add(-t.rateWindow)
	newWindow := stats.packetsInWindow[:0]
	for _, pt := range stats.packetsInWindow {
		if pt.After(cutoff) {
			newWindow = append(newWindow, pt)
		}
	}
	stats.packetsInWindow = newWindow

	// Track source
	source, exists := stats.Sources[sourceCID]
	if !exists {
		source = &Source{
			CID:  sourceCID,
			Name: sourceName,
		}
		stats.Sources[sourceCID] = source
	}

	// Check for packet loss (sequence gap)
	if exists && source.PacketCount > 0 {
		expectedSeq := uint8((int(source.LastSequence) + 1) % 256)
		if sequence != expectedSeq {
			// Calculate how many packets were lost
			var lost int
			if sequence > expectedSeq {
				lost = int(sequence) - int(expectedSeq)
			} else {
				// Wrapped around
				lost = 256 - int(expectedSeq) + int(sequence)
			}
			source.LostPackets += uint64(lost)
			stats.LostPackets += uint64(lost)
		}
	}

	source.LastSequence = sequence
	source.LastSeen = now
	source.PacketCount++
	source.Name = sourceName // Update name in case it changed
}

// GetUniverseStats returns stats for a specific universe
func (t *Tracker) GetUniverseStats(universeID uint16) *UniverseStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.universes[universeID]
}

// GetPacketRate returns packets per second for a universe
func (t *Tracker) GetPacketRate(universeID uint16) float64 {
	t.mu.RLock()
	stats := t.universes[universeID]
	t.mu.RUnlock()

	if stats == nil {
		return 0
	}

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	// Clean old packets and count
	now := time.Now()
	cutoff := now.Add(-t.rateWindow)
	count := 0
	for _, pt := range stats.packetsInWindow {
		if pt.After(cutoff) {
			count++
		}
	}

	return float64(count) / t.rateWindow.Seconds()
}

// GetLossPercentage returns packet loss percentage for a universe
func (t *Tracker) GetLossPercentage(universeID uint16) float64 {
	t.mu.RLock()
	stats := t.universes[universeID]
	t.mu.RUnlock()

	if stats == nil {
		return 0
	}

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	totalExpected := stats.PacketCount + stats.LostPackets
	if totalExpected == 0 {
		return 0
	}

	return float64(stats.LostPackets) / float64(totalExpected) * 100
}

// GetSources returns all sources for a universe
func (t *Tracker) GetSources(universeID uint16) []Source {
	t.mu.RLock()
	stats := t.universes[universeID]
	t.mu.RUnlock()

	if stats == nil {
		return nil
	}

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	sources := make([]Source, 0, len(stats.Sources))
	for _, s := range stats.Sources {
		sources = append(sources, *s)
	}
	return sources
}

// GetAllUniverseIDs returns all tracked universe IDs
func (t *Tracker) GetAllUniverseIDs() []uint16 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ids := make([]uint16, 0, len(t.universes))
	for id := range t.universes {
		ids = append(ids, id)
	}
	return ids
}
