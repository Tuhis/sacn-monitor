package sacn

import (
	"testing"
)

// buildValidPacket creates a valid E1.31 packet for testing
func buildValidPacket(universe uint16, sequence uint8, sourceName string, channels []byte) []byte {
	channelCount := len(channels)
	if channelCount > E131MaxChannels {
		channelCount = E131MaxChannels
	}

	packetSize := E131HeaderSize + channelCount
	packet := make([]byte, packetSize)

	// === Root Layer ===
	// Preamble Size (offset 0-1): 0x0010
	packet[0] = 0x00
	packet[1] = 0x10

	// ACN Packet Identifier (offset 4-15)
	copy(packet[4:16], ACNPacketIdentifier)

	// Root Flags & Length (offset 16-17)
	rootLength := uint16(packetSize - 16)
	packet[16] = 0x70 | byte(rootLength>>8)
	packet[17] = byte(rootLength)

	// Root Vector (offset 18-21): 0x00000004
	packet[18] = 0x00
	packet[19] = 0x00
	packet[20] = 0x00
	packet[21] = 0x04

	// CID (offset 22-37): test UUID
	copy(packet[22:38], []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0,
		0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0})

	// === Framing Layer ===
	// Framing Flags & Length (offset 38-39)
	framingLength := uint16(packetSize - 38)
	packet[38] = 0x70 | byte(framingLength>>8)
	packet[39] = byte(framingLength)

	// Framing Vector (offset 40-43): 0x00000002
	packet[40] = 0x00
	packet[41] = 0x00
	packet[42] = 0x00
	packet[43] = 0x02

	// Source Name (offset 44-107): 64 bytes
	copy(packet[44:108], []byte(sourceName))

	// Priority (offset 108)
	packet[108] = 100

	// Sequence (offset 111)
	packet[111] = sequence

	// Universe (offset 113-114)
	packet[113] = byte(universe >> 8)
	packet[114] = byte(universe)

	// === DMP Layer ===
	// DMP Flags & Length (offset 115-116)
	dmpLength := uint16(packetSize - 115)
	packet[115] = 0x70 | byte(dmpLength>>8)
	packet[116] = byte(dmpLength)

	// DMP Vector (offset 117): 0x02
	packet[117] = 0x02

	// Address Type & Data Type (offset 118): 0xa1
	packet[118] = 0xa1

	// Address Increment (offset 121-122): 0x0001
	packet[121] = 0x00
	packet[122] = 0x01

	// Property Value Count (offset 123-124)
	propValCount := uint16(1 + channelCount)
	packet[123] = byte(propValCount >> 8)
	packet[124] = byte(propValCount)

	// DMX Start Code (offset 125): 0x00
	packet[125] = 0x00

	// Channel data (offset 126+)
	copy(packet[E131HeaderSize:], channels)

	return packet
}

func TestParse_ValidPacket(t *testing.T) {
	channels := []byte{255, 128, 64, 0, 100, 200}
	packet := buildValidPacket(1, 42, "test-source", channels)

	result, err := Parse(packet)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if result.Universe != 1 {
		t.Errorf("Universe = %d, want 1", result.Universe)
	}

	if result.Sequence != 42 {
		t.Errorf("Sequence = %d, want 42", result.Sequence)
	}

	if result.SourceName != "test-source" {
		t.Errorf("SourceName = %q, want %q", result.SourceName, "test-source")
	}

	if result.Priority != 100 {
		t.Errorf("Priority = %d, want 100", result.Priority)
	}

	if result.StartCode != 0 {
		t.Errorf("StartCode = %d, want 0", result.StartCode)
	}

	if len(result.ChannelData) != len(channels) {
		t.Errorf("ChannelData length = %d, want %d", len(result.ChannelData), len(channels))
	}

	for i, expected := range channels {
		if result.ChannelData[i] != expected {
			t.Errorf("ChannelData[%d] = %d, want %d", i, result.ChannelData[i], expected)
		}
	}
}

func TestParse_MaxChannels(t *testing.T) {
	channels := make([]byte, E131MaxChannels)
	for i := range channels {
		channels[i] = byte(i % 256)
	}

	packet := buildValidPacket(100, 1, "full-universe", channels)

	result, err := Parse(packet)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if result.ChannelCount() != E131MaxChannels {
		t.Errorf("ChannelCount() = %d, want %d", result.ChannelCount(), E131MaxChannels)
	}
}

func TestParse_EmptyChannelData(t *testing.T) {
	packet := buildValidPacket(1, 1, "empty", []byte{})

	result, err := Parse(packet)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if result.ChannelCount() != 0 {
		t.Errorf("ChannelCount() = %d, want 0", result.ChannelCount())
	}
}

func TestParse_PacketTooShort(t *testing.T) {
	shortPacket := make([]byte, 50) // Less than E131HeaderSize

	_, err := Parse(shortPacket)
	if err == nil {
		t.Fatal("Parse() expected error for short packet, got nil")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}

	if parseErr.Message != "packet too short" {
		t.Errorf("ParseError.Message = %q, want %q", parseErr.Message, "packet too short")
	}
}

func TestParse_InvalidPreamble(t *testing.T) {
	packet := buildValidPacket(1, 1, "test", []byte{0})
	packet[0] = 0xFF // Invalid preamble

	_, err := Parse(packet)
	if err == nil {
		t.Fatal("Parse() expected error for invalid preamble, got nil")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}

	if parseErr.Message != "invalid preamble size" {
		t.Errorf("ParseError.Message = %q, want %q", parseErr.Message, "invalid preamble size")
	}
}

func TestParse_InvalidACNIdentifier(t *testing.T) {
	packet := buildValidPacket(1, 1, "test", []byte{0})
	packet[4] = 0xFF // Corrupt ACN identifier

	_, err := Parse(packet)
	if err == nil {
		t.Fatal("Parse() expected error for invalid ACN identifier, got nil")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}

	if parseErr.Message != "invalid ACN packet identifier" {
		t.Errorf("ParseError.Message = %q, want %q", parseErr.Message, "invalid ACN packet identifier")
	}
}

func TestParse_InvalidRootVector(t *testing.T) {
	packet := buildValidPacket(1, 1, "test", []byte{0})
	packet[21] = 0xFF // Invalid root vector

	_, err := Parse(packet)
	if err == nil {
		t.Fatal("Parse() expected error for invalid root vector, got nil")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}

	if parseErr.Message != "invalid root vector" {
		t.Errorf("ParseError.Message = %q, want %q", parseErr.Message, "invalid root vector")
	}
}

func TestParse_InvalidFramingVector(t *testing.T) {
	packet := buildValidPacket(1, 1, "test", []byte{0})
	packet[43] = 0xFF // Invalid framing vector

	_, err := Parse(packet)
	if err == nil {
		t.Fatal("Parse() expected error for invalid framing vector, got nil")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}

	if parseErr.Message != "invalid framing vector" {
		t.Errorf("ParseError.Message = %q, want %q", parseErr.Message, "invalid framing vector")
	}
}

func TestParse_InvalidDMPVector(t *testing.T) {
	packet := buildValidPacket(1, 1, "test", []byte{0})
	packet[117] = 0xFF // Invalid DMP vector

	_, err := Parse(packet)
	if err == nil {
		t.Fatal("Parse() expected error for invalid DMP vector, got nil")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}

	if parseErr.Message != "invalid DMP vector" {
		t.Errorf("ParseError.Message = %q, want %q", parseErr.Message, "invalid DMP vector")
	}
}

func TestParse_LargeUniverse(t *testing.T) {
	packet := buildValidPacket(63999, 1, "test", []byte{255})

	result, err := Parse(packet)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if result.Universe != 63999 {
		t.Errorf("Universe = %d, want 63999", result.Universe)
	}
}

func TestParse_CIDExtraction(t *testing.T) {
	packet := buildValidPacket(1, 1, "test", []byte{0})

	result, err := Parse(packet)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	expectedCID := [16]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0,
		0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}

	if result.CID != expectedCID {
		t.Errorf("CID = %v, want %v", result.CID, expectedCID)
	}
}

func TestPacket_ChannelCount(t *testing.T) {
	tests := []struct {
		name     string
		channels []byte
		want     int
	}{
		{"empty", []byte{}, 0},
		{"one channel", []byte{255}, 1},
		{"three channels", []byte{255, 128, 64}, 3},
		{"full universe", make([]byte, 512), 512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Packet{ChannelData: tt.channels}
			if got := p.ChannelCount(); got != tt.want {
				t.Errorf("ChannelCount() = %d, want %d", got, tt.want)
			}
		})
	}
}
