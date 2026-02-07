package stats

import (
	"sync"
	"time"
)

// Constants for loss tracking
const (
	// lossWindowDuration is the time window for recent loss calculation
	lossWindowDuration = time.Minute
	// sourceRestartThreshold is the sequence gap above which we assume source restart
	sourceRestartThreshold = 200
)

// PacketEvent records a packet reception event for sliding window tracking
type PacketEvent struct {
	Timestamp time.Time
	Received  uint64 // packets received in this event
	Lost      uint64 // packets lost detected in this event
}

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
	packetsInWindow []time.Time   // For rate calculation
	lossWindow      []PacketEvent // For sliding window loss calculation
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
	source, sourceExists := stats.Sources[sourceCID]
	if !sourceExists {
		source = &Source{
			CID:  sourceCID,
			Name: sourceName,
		}
		stats.Sources[sourceCID] = source
	}

	// Check for packet loss (sequence gap)
	var lostThisPacket uint64
	if sourceExists && source.PacketCount > 0 {
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
			// If gap is too large, assume source restart rather than massive loss
			if lost < sourceRestartThreshold {
				lostThisPacket = uint64(lost)
				source.LostPackets += lostThisPacket
				stats.LostPackets += lostThisPacket
			}
			// If lost >= sourceRestartThreshold, we treat it as a restart
			// and don't count any loss
		}
	}

	// Record event for sliding window loss tracking
	stats.lossWindow = append(stats.lossWindow, PacketEvent{
		Timestamp: now,
		Received:  1,
		Lost:      lostThisPacket,
	})

	// Clean old events from loss window
	lossCutoff := now.Add(-lossWindowDuration)
	newLossWindow := stats.lossWindow[:0]
	for _, evt := range stats.lossWindow {
		if evt.Timestamp.After(lossCutoff) {
			newLossWindow = append(newLossWindow, evt)
		}
	}
	stats.lossWindow = newLossWindow

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

// GetLossPercentage returns cumulative packet loss percentage for a universe
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

// GetRecentLossPercentage returns packet loss percentage for the last minute
func (t *Tracker) GetRecentLossPercentage(universeID uint16) float64 {
	t.mu.RLock()
	stats := t.universes[universeID]
	t.mu.RUnlock()

	if stats == nil {
		return 0
	}

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	// Sum up received and lost from the sliding window
	now := time.Now()
	cutoff := now.Add(-lossWindowDuration)

	var totalReceived, totalLost uint64
	for _, evt := range stats.lossWindow {
		if evt.Timestamp.After(cutoff) {
			totalReceived += evt.Received
			totalLost += evt.Lost
		}
	}

	totalExpected := totalReceived + totalLost
	if totalExpected == 0 {
		return 0
	}

	return float64(totalLost) / float64(totalExpected) * 100
}

// GetSourceLossPercentage returns packet loss percentage for a specific source
func (t *Tracker) GetSourceLossPercentage(universeID uint16, sourceCID [16]byte) float64 {
	t.mu.RLock()
	stats := t.universes[universeID]
	t.mu.RUnlock()

	if stats == nil {
		return 0
	}

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	source, exists := stats.Sources[sourceCID]
	if !exists {
		return 0
	}

	totalExpected := source.PacketCount + source.LostPackets
	if totalExpected == 0 {
		return 0
	}

	return float64(source.LostPackets) / float64(totalExpected) * 100
}

// ResetUniverseStats clears all statistics for a specific universe
func (t *Tracker) ResetUniverseStats(universeID uint16) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if stats, exists := t.universes[universeID]; exists {
		stats.mu.Lock()
		stats.PacketCount = 0
		stats.LostPackets = 0
		stats.packetsInWindow = nil
		stats.lossWindow = nil
		for _, source := range stats.Sources {
			source.PacketCount = 0
			source.LostPackets = 0
		}
		stats.mu.Unlock()
	}
}

// ResetAllStats clears all tracked data
func (t *Tracker) ResetAllStats() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.universes = make(map[uint16]*UniverseStats)
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
