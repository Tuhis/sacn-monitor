package sacn

import (
	"net"
	"time"
)

// E131 protocol constants
const (
	E131Port          = 5568
	E131MaxChannels   = 512
	E131HeaderSize    = 126
	E131RootVector    = 0x00000004
	E131FramingVector = 0x00000002
	E131DMPVector     = 0x02
	E131MulticastBase = "239.255."
)

// ACNPacketIdentifier is the magic bytes for E1.31 packets
var ACNPacketIdentifier = []byte{0x41, 0x53, 0x43, 0x2d, 0x45, 0x31, 0x2e, 0x31, 0x37, 0x00, 0x00, 0x00}

// Packet represents a parsed E1.31 packet
type Packet struct {
	// Root layer
	CID [16]byte // Component Identifier (UUID)

	// Framing layer
	SourceName string
	Priority   uint8
	Sequence   uint8
	Universe   uint16

	// DMP layer
	StartCode   uint8
	ChannelData []byte // DMX channel values (up to 512)

	// Metadata
	SourceAddr net.Addr
	ReceivedAt time.Time
}

// ChannelCount returns the number of channels in this packet
func (p *Packet) ChannelCount() int {
	return len(p.ChannelData)
}

// ParseError represents an error during packet parsing
type ParseError struct {
	Message string
	Offset  int
}

func (e *ParseError) Error() string {
	return e.Message
}

// NewParseError creates a new ParseError
func NewParseError(message string, offset int) *ParseError {
	return &ParseError{Message: message, Offset: offset}
}
