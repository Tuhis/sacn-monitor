package stats

import (
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	tracker := NewTracker()

	if tracker == nil {
		t.Fatal("NewTracker() returned nil")
	}

	if len(tracker.GetAllUniverseIDs()) != 0 {
		t.Errorf("GetAllUniverseIDs() = %v, want empty", tracker.GetAllUniverseIDs())
	}
}

func TestTracker_RecordPacket(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	tracker.RecordPacket(1, cid, "test-source", 0)

	stats := tracker.GetUniverseStats(1)
	if stats == nil {
		t.Fatal("GetUniverseStats(1) returned nil")
	}

	if stats.PacketCount != 1 {
		t.Errorf("PacketCount = %d, want 1", stats.PacketCount)
	}

	sources := tracker.GetSources(1)
	if len(sources) != 1 {
		t.Fatalf("len(GetSources(1)) = %d, want 1", len(sources))
	}

	if sources[0].Name != "test-source" {
		t.Errorf("Source.Name = %q, want %q", sources[0].Name, "test-source")
	}

	if sources[0].PacketCount != 1 {
		t.Errorf("Source.PacketCount = %d, want 1", sources[0].PacketCount)
	}
}

func TestTracker_PacketLossDetection_SimpleGap(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	// Send sequence 0, then skip to 5 (lost 1, 2, 3, 4)
	tracker.RecordPacket(1, cid, "test", 0)
	tracker.RecordPacket(1, cid, "test", 5)

	stats := tracker.GetUniverseStats(1)
	if stats.LostPackets != 4 {
		t.Errorf("LostPackets = %d, want 4", stats.LostPackets)
	}

	sources := tracker.GetSources(1)
	if sources[0].LostPackets != 4 {
		t.Errorf("Source.LostPackets = %d, want 4", sources[0].LostPackets)
	}
}

func TestTracker_PacketLossDetection_Wraparound(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	// Send sequence 254, then 1 (lost 255, 0)
	tracker.RecordPacket(1, cid, "test", 254)
	tracker.RecordPacket(1, cid, "test", 1)

	stats := tracker.GetUniverseStats(1)
	if stats.LostPackets != 2 {
		t.Errorf("LostPackets = %d, want 2 (255 and 0)", stats.LostPackets)
	}
}

func TestTracker_PacketLossDetection_NoLoss(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	// Sequential packets - no loss
	for i := 0; i < 10; i++ {
		tracker.RecordPacket(1, cid, "test", uint8(i))
	}

	stats := tracker.GetUniverseStats(1)
	if stats.LostPackets != 0 {
		t.Errorf("LostPackets = %d, want 0", stats.LostPackets)
	}

	if stats.PacketCount != 10 {
		t.Errorf("PacketCount = %d, want 10", stats.PacketCount)
	}
}

func TestTracker_GetLossPercentage(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	// 10 packets received, 2 lost (seq 0, then 3 - lost 1 and 2)
	tracker.RecordPacket(1, cid, "test", 0)
	tracker.RecordPacket(1, cid, "test", 3) // Lost 1, 2

	loss := tracker.GetLossPercentage(1)
	// 2 lost out of 4 total expected (2 received + 2 lost) = 50%
	expectedLoss := 50.0
	if loss != expectedLoss {
		t.Errorf("GetLossPercentage(1) = %.2f%%, want %.2f%%", loss, expectedLoss)
	}
}

func TestTracker_GetLossPercentage_NoPackets(t *testing.T) {
	tracker := NewTracker()

	loss := tracker.GetLossPercentage(999)
	if loss != 0 {
		t.Errorf("GetLossPercentage(999) = %.2f, want 0 (no packets)", loss)
	}
}

func TestTracker_GetPacketRate(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	// Record multiple packets quickly
	for i := 0; i < 50; i++ {
		tracker.RecordPacket(1, cid, "test", uint8(i%256))
	}

	rate := tracker.GetPacketRate(1)

	// Rate should be at least 50 (all packets within 1 second window)
	if rate < 50 {
		t.Errorf("GetPacketRate(1) = %.2f, want >= 50", rate)
	}
}

func TestTracker_GetPacketRate_NoPackets(t *testing.T) {
	tracker := NewTracker()

	rate := tracker.GetPacketRate(999)
	if rate != 0 {
		t.Errorf("GetPacketRate(999) = %.2f, want 0", rate)
	}
}

func TestTracker_MultipleSources(t *testing.T) {
	tracker := NewTracker()
	cid1 := [16]byte{1, 0, 0, 0}
	cid2 := [16]byte{2, 0, 0, 0}

	tracker.RecordPacket(1, cid1, "source-1", 0)
	tracker.RecordPacket(1, cid2, "source-2", 0)
	tracker.RecordPacket(1, cid1, "source-1", 1)

	sources := tracker.GetSources(1)
	if len(sources) != 2 {
		t.Fatalf("len(GetSources(1)) = %d, want 2", len(sources))
	}

	stats := tracker.GetUniverseStats(1)
	if stats.PacketCount != 3 {
		t.Errorf("PacketCount = %d, want 3", stats.PacketCount)
	}
}

func TestTracker_MultipleUniverses(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	tracker.RecordPacket(1, cid, "test", 0)
	tracker.RecordPacket(2, cid, "test", 0)
	tracker.RecordPacket(3, cid, "test", 0)

	ids := tracker.GetAllUniverseIDs()
	if len(ids) != 3 {
		t.Errorf("len(GetAllUniverseIDs()) = %d, want 3", len(ids))
	}
}

func TestTracker_SourceNameUpdate(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	tracker.RecordPacket(1, cid, "old-name", 0)
	tracker.RecordPacket(1, cid, "new-name", 1)

	sources := tracker.GetSources(1)
	if len(sources) != 1 {
		t.Fatalf("len(sources) = %d, want 1", len(sources))
	}

	if sources[0].Name != "new-name" {
		t.Errorf("Source.Name = %q, want %q", sources[0].Name, "new-name")
	}
}

func TestSource_LastSeen(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	before := time.Now()
	tracker.RecordPacket(1, cid, "test", 0)
	after := time.Now()

	sources := tracker.GetSources(1)
	if sources[0].LastSeen.Before(before) || sources[0].LastSeen.After(after) {
		t.Errorf("Source.LastSeen not within expected range")
	}
}

func TestTracker_GetRecentLossPercentage(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	// Send 2 packets: 0, then 3 (lost 1, 2) - 50% loss
	tracker.RecordPacket(1, cid, "test", 0)
	tracker.RecordPacket(1, cid, "test", 3) // Lost 1, 2

	loss := tracker.GetRecentLossPercentage(1)
	// 2 lost out of 4 total expected (2 received + 2 lost) = 50%
	expectedLoss := 50.0
	if loss != expectedLoss {
		t.Errorf("GetRecentLossPercentage(1) = %.2f%%, want %.2f%%", loss, expectedLoss)
	}
}

func TestTracker_GetRecentLossPercentage_NoPackets(t *testing.T) {
	tracker := NewTracker()

	loss := tracker.GetRecentLossPercentage(999)
	if loss != 0 {
		t.Errorf("GetRecentLossPercentage(999) = %.2f, want 0 (no packets)", loss)
	}
}

func TestTracker_SourceRestartDetection(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	// Send sequence 100, then jump to 50 (gap of 206 - should be treated as restart)
	tracker.RecordPacket(1, cid, "test", 100)
	tracker.RecordPacket(1, cid, "test", 50) // Gap is 256-100+50 = 206, exceeds threshold

	stats := tracker.GetUniverseStats(1)
	if stats.LostPackets != 0 {
		t.Errorf("LostPackets = %d, want 0 (should detect restart, not loss)", stats.LostPackets)
	}

	sources := tracker.GetSources(1)
	if sources[0].LostPackets != 0 {
		t.Errorf("Source.LostPackets = %d, want 0 (should detect restart)", sources[0].LostPackets)
	}
}

func TestTracker_SourceRestartDetection_SmallGapStillCounted(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	// Send sequence 0, then 100 (gap of 99 - should still count as loss)
	tracker.RecordPacket(1, cid, "test", 0)
	tracker.RecordPacket(1, cid, "test", 100) // Lost 1-99 = 99 packets

	stats := tracker.GetUniverseStats(1)
	if stats.LostPackets != 99 {
		t.Errorf("LostPackets = %d, want 99", stats.LostPackets)
	}
}

func TestTracker_GetSourceLossPercentage(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	// Send 2 packets: 0, then 3 (lost 1, 2)
	tracker.RecordPacket(1, cid, "test", 0)
	tracker.RecordPacket(1, cid, "test", 3)

	loss := tracker.GetSourceLossPercentage(1, cid)
	expectedLoss := 50.0
	if loss != expectedLoss {
		t.Errorf("GetSourceLossPercentage(1, cid) = %.2f%%, want %.2f%%", loss, expectedLoss)
	}
}

func TestTracker_GetSourceLossPercentage_UnknownSource(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}
	unknownCid := [16]byte{9, 9, 9, 9}

	tracker.RecordPacket(1, cid, "test", 0)

	loss := tracker.GetSourceLossPercentage(1, unknownCid)
	if loss != 0 {
		t.Errorf("GetSourceLossPercentage(1, unknownCid) = %.2f, want 0", loss)
	}
}

func TestTracker_ResetUniverseStats(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	// Record some packets with loss
	tracker.RecordPacket(1, cid, "test", 0)
	tracker.RecordPacket(1, cid, "test", 5) // Lost 4 packets

	// Verify stats exist
	stats := tracker.GetUniverseStats(1)
	if stats.PacketCount != 2 || stats.LostPackets != 4 {
		t.Fatalf("Initial stats not as expected")
	}

	// Reset
	tracker.ResetUniverseStats(1)

	// Verify counters are cleared
	stats = tracker.GetUniverseStats(1)
	if stats.PacketCount != 0 {
		t.Errorf("PacketCount = %d, want 0 after reset", stats.PacketCount)
	}
	if stats.LostPackets != 0 {
		t.Errorf("LostPackets = %d, want 0 after reset", stats.LostPackets)
	}

	// Sources should still exist but with zeroed counters
	sources := tracker.GetSources(1)
	if len(sources) != 1 {
		t.Fatalf("Expected 1 source after reset, got %d", len(sources))
	}
	if sources[0].PacketCount != 0 {
		t.Errorf("Source.PacketCount = %d, want 0 after reset", sources[0].PacketCount)
	}
}

func TestTracker_ResetAllStats(t *testing.T) {
	tracker := NewTracker()
	cid := [16]byte{1, 2, 3, 4}

	// Record packets on multiple universes
	tracker.RecordPacket(1, cid, "test", 0)
	tracker.RecordPacket(2, cid, "test", 0)
	tracker.RecordPacket(3, cid, "test", 0)

	// Verify we have 3 universes
	ids := tracker.GetAllUniverseIDs()
	if len(ids) != 3 {
		t.Fatalf("Expected 3 universes, got %d", len(ids))
	}

	// Reset all
	tracker.ResetAllStats()

	// Verify all universe data is cleared
	ids = tracker.GetAllUniverseIDs()
	if len(ids) != 0 {
		t.Errorf("Expected 0 universes after ResetAllStats, got %d", len(ids))
	}
}
