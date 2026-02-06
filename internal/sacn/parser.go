package sacn

import (
	"bytes"
	"encoding/binary"
	"time"
)

// Parse parses a raw E1.31 packet and returns a Packet struct
func Parse(data []byte) (*Packet, error) {
	if len(data) < E131HeaderSize {
		return nil, NewParseError("packet too short", 0)
	}

	// Validate preamble size (offset 0-1): must be 0x0010
	if data[0] != 0x00 || data[1] != 0x10 {
		return nil, NewParseError("invalid preamble size", 0)
	}

	// Validate ACN Packet Identifier (offset 4-15)
	if !bytes.Equal(data[4:16], ACNPacketIdentifier) {
		return nil, NewParseError("invalid ACN packet identifier", 4)
	}

	// Validate Root Vector (offset 18-21): must be 0x00000004
	rootVector := binary.BigEndian.Uint32(data[18:22])
	if rootVector != E131RootVector {
		return nil, NewParseError("invalid root vector", 18)
	}

	// Validate Framing Vector (offset 40-43): must be 0x00000002
	framingVector := binary.BigEndian.Uint32(data[40:44])
	if framingVector != E131FramingVector {
		return nil, NewParseError("invalid framing vector", 40)
	}

	// Validate DMP Vector (offset 117): must be 0x02
	if data[117] != E131DMPVector {
		return nil, NewParseError("invalid DMP vector", 117)
	}

	packet := &Packet{
		ReceivedAt: time.Now(),
	}

	// Extract CID (offset 22-37)
	copy(packet.CID[:], data[22:38])

	// Extract Source Name (offset 44-107): 64 bytes, null-terminated
	sourceNameBytes := data[44:108]
	nullIdx := bytes.IndexByte(sourceNameBytes, 0)
	if nullIdx > 0 {
		packet.SourceName = string(sourceNameBytes[:nullIdx])
	} else if nullIdx < 0 {
		packet.SourceName = string(sourceNameBytes)
	}

	// Extract Priority (offset 108)
	packet.Priority = data[108]

	// Extract Sequence (offset 111)
	packet.Sequence = data[111]

	// Extract Universe (offset 113-114)
	packet.Universe = binary.BigEndian.Uint16(data[113:115])

	// Extract Start Code (offset 125)
	packet.StartCode = data[125]

	// Extract channel data (offset 126+)
	if len(data) > E131HeaderSize {
		channelCount := len(data) - E131HeaderSize
		if channelCount > E131MaxChannels {
			channelCount = E131MaxChannels
		}
		packet.ChannelData = make([]byte, channelCount)
		copy(packet.ChannelData, data[E131HeaderSize:E131HeaderSize+channelCount])
	}

	return packet, nil
}
