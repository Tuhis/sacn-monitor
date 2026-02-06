package universe

import (
	"sync"
	"time"
)

// Channel represents the state of a single DMX channel
type Channel struct {
	Value      uint8     // Current value (0-255)
	Active     bool      // True if channel is included in received packets
	LastUpdate time.Time // When the channel was last updated
}

// Universe represents the state of a single sACN universe
type Universe struct {
	ID           uint16
	Channels     [512]Channel
	SourceName   string
	SourceCID    [16]byte
	Priority     uint8
	LastSequence uint8
	LastPacket   time.Time
	PacketCount  uint64
	mu           sync.RWMutex
}

// NewUniverse creates a new universe with the given ID
func NewUniverse(id uint16) *Universe {
	return &Universe{
		ID: id,
	}
}

// Update updates the universe with new channel data from a packet
func (u *Universe) Update(channelData []byte, sourceName string, sourceCID [16]byte, priority uint8, sequence uint8) {
	u.mu.Lock()
	defer u.mu.Unlock()

	now := time.Now()

	// Update metadata
	u.SourceName = sourceName
	u.SourceCID = sourceCID
	u.Priority = priority
	u.LastSequence = sequence
	u.LastPacket = now
	u.PacketCount++

	// Update channels that are in the packet
	for i := 0; i < len(channelData) && i < 512; i++ {
		u.Channels[i].Value = channelData[i]
		u.Channels[i].Active = true
		u.Channels[i].LastUpdate = now
	}
}

// GetChannel returns a copy of the channel at the given index (0-511)
func (u *Universe) GetChannel(index int) Channel {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if index < 0 || index >= 512 {
		return Channel{}
	}
	return u.Channels[index]
}

// GetAllChannels returns a copy of all channels
func (u *Universe) GetAllChannels() [512]Channel {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.Channels
}

// ActiveChannelCount returns the number of channels that are receiving data
func (u *Universe) ActiveChannelCount() int {
	u.mu.RLock()
	defer u.mu.RUnlock()

	count := 0
	for _, ch := range u.Channels {
		if ch.Active {
			count++
		}
	}
	return count
}

// IsStale returns true if the universe hasn't received data for the given duration
func (u *Universe) IsStale(timeout time.Duration) bool {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if u.LastPacket.IsZero() {
		return true
	}
	return time.Since(u.LastPacket) > timeout
}

// GetInfo returns a snapshot of the universe metadata
func (u *Universe) GetInfo() UniverseInfo {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return UniverseInfo{
		ID:           u.ID,
		SourceName:   u.SourceName,
		SourceCID:    u.SourceCID,
		Priority:     u.Priority,
		LastSequence: u.LastSequence,
		LastPacket:   u.LastPacket,
		PacketCount:  u.PacketCount,
	}
}

// UniverseInfo is a snapshot of universe metadata (no mutex needed)
type UniverseInfo struct {
	ID           uint16
	SourceName   string
	SourceCID    [16]byte
	Priority     uint8
	LastSequence uint8
	LastPacket   time.Time
	PacketCount  uint64
}
