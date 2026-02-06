package universe

import (
	"testing"
	"time"
)

func TestNewUniverse(t *testing.T) {
	u := NewUniverse(42)

	if u.ID != 42 {
		t.Errorf("ID = %d, want 42", u.ID)
	}

	if u.PacketCount != 0 {
		t.Errorf("PacketCount = %d, want 0", u.PacketCount)
	}

	// All channels should be inactive initially
	for i := 0; i < 512; i++ {
		if u.Channels[i].Active {
			t.Errorf("Channel[%d].Active = true, want false", i)
		}
	}
}

func TestUniverse_Update(t *testing.T) {
	u := NewUniverse(1)
	channelData := []byte{255, 128, 64, 0}
	cid := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	u.Update(channelData, "test-source", cid, 100, 42)

	// Check metadata
	if u.SourceName != "test-source" {
		t.Errorf("SourceName = %q, want %q", u.SourceName, "test-source")
	}

	if u.SourceCID != cid {
		t.Errorf("SourceCID mismatch")
	}

	if u.Priority != 100 {
		t.Errorf("Priority = %d, want 100", u.Priority)
	}

	if u.LastSequence != 42 {
		t.Errorf("LastSequence = %d, want 42", u.LastSequence)
	}

	if u.PacketCount != 1 {
		t.Errorf("PacketCount = %d, want 1", u.PacketCount)
	}

	// Check channel values
	for i, expected := range channelData {
		ch := u.GetChannel(i)
		if ch.Value != expected {
			t.Errorf("Channel[%d].Value = %d, want %d", i, ch.Value, expected)
		}
		if !ch.Active {
			t.Errorf("Channel[%d].Active = false, want true", i)
		}
	}

	// Check that remaining channels are still inactive
	ch := u.GetChannel(4)
	if ch.Active {
		t.Error("Channel[4].Active = true, want false (not in packet)")
	}
}

func TestUniverse_ActiveChannelCount(t *testing.T) {
	u := NewUniverse(1)

	if count := u.ActiveChannelCount(); count != 0 {
		t.Errorf("ActiveChannelCount() = %d, want 0", count)
	}

	u.Update([]byte{255, 128, 64}, "test", [16]byte{}, 100, 1)

	if count := u.ActiveChannelCount(); count != 3 {
		t.Errorf("ActiveChannelCount() = %d, want 3", count)
	}
}

func TestUniverse_IsStale(t *testing.T) {
	u := NewUniverse(1)

	// Should be stale initially (no packets received)
	if !u.IsStale(time.Second) {
		t.Error("IsStale() = false, want true (no packets)")
	}

	u.Update([]byte{255}, "test", [16]byte{}, 100, 1)

	// Should not be stale immediately after update
	if u.IsStale(time.Second) {
		t.Error("IsStale() = true, want false (just received)")
	}
}

func TestUniverse_GetChannel_OutOfBounds(t *testing.T) {
	u := NewUniverse(1)

	// Negative index
	ch := u.GetChannel(-1)
	if ch.Active {
		t.Error("GetChannel(-1).Active = true, want false")
	}

	// Too high index
	ch = u.GetChannel(512)
	if ch.Active {
		t.Error("GetChannel(512).Active = true, want false")
	}
}

func TestUniverse_GetAllChannels(t *testing.T) {
	u := NewUniverse(1)
	u.Update([]byte{100, 200}, "test", [16]byte{}, 100, 1)

	channels := u.GetAllChannels()

	if channels[0].Value != 100 {
		t.Errorf("channels[0].Value = %d, want 100", channels[0].Value)
	}

	if channels[1].Value != 200 {
		t.Errorf("channels[1].Value = %d, want 200", channels[1].Value)
	}
}

func TestUniverse_GetInfo(t *testing.T) {
	u := NewUniverse(1)
	cid := [16]byte{1, 2, 3, 4}
	u.Update([]byte{255}, "source-name", cid, 50, 99)

	info := u.GetInfo()

	if info.ID != 1 {
		t.Errorf("info.ID = %d, want 1", info.ID)
	}

	if info.SourceName != "source-name" {
		t.Errorf("info.SourceName = %q, want %q", info.SourceName, "source-name")
	}

	if info.Priority != 50 {
		t.Errorf("info.Priority = %d, want 50", info.Priority)
	}

	if info.LastSequence != 99 {
		t.Errorf("info.LastSequence = %d, want 99", info.LastSequence)
	}

	if info.PacketCount != 1 {
		t.Errorf("info.PacketCount = %d, want 1", info.PacketCount)
	}
}

// Manager tests

func TestNewManager(t *testing.T) {
	m := NewManager()

	if m.Count() != 0 {
		t.Errorf("Count() = %d, want 0", m.Count())
	}
}

func TestManager_GetOrCreate(t *testing.T) {
	m := NewManager()

	u1 := m.GetOrCreate(1)
	if u1 == nil {
		t.Fatal("GetOrCreate(1) returned nil")
	}

	if u1.ID != 1 {
		t.Errorf("ID = %d, want 1", u1.ID)
	}

	// Getting the same universe again should return the same instance
	u1Again := m.GetOrCreate(1)
	if u1Again != u1 {
		t.Error("GetOrCreate(1) returned different instance")
	}

	if m.Count() != 1 {
		t.Errorf("Count() = %d, want 1", m.Count())
	}
}

func TestManager_Get(t *testing.T) {
	m := NewManager()

	// Should return nil for non-existent universe
	if u := m.Get(1); u != nil {
		t.Error("Get(1) returned non-nil for non-existent universe")
	}

	m.GetOrCreate(1)

	// Now it should exist
	if u := m.Get(1); u == nil {
		t.Error("Get(1) returned nil for existing universe")
	}
}

func TestManager_GetAll_Sorted(t *testing.T) {
	m := NewManager()

	// Create universes in non-sorted order
	m.GetOrCreate(100)
	m.GetOrCreate(1)
	m.GetOrCreate(50)

	all := m.GetAll()

	if len(all) != 3 {
		t.Fatalf("len(GetAll()) = %d, want 3", len(all))
	}

	// Should be sorted by ID
	if all[0].ID != 1 {
		t.Errorf("all[0].ID = %d, want 1", all[0].ID)
	}
	if all[1].ID != 50 {
		t.Errorf("all[1].ID = %d, want 50", all[1].ID)
	}
	if all[2].ID != 100 {
		t.Errorf("all[2].ID = %d, want 100", all[2].ID)
	}
}

func TestManager_Remove(t *testing.T) {
	m := NewManager()

	m.GetOrCreate(1)
	m.GetOrCreate(2)

	if m.Count() != 2 {
		t.Fatalf("Count() = %d, want 2", m.Count())
	}

	m.Remove(1)

	if m.Count() != 1 {
		t.Errorf("Count() = %d, want 1", m.Count())
	}

	if m.Get(1) != nil {
		t.Error("Get(1) returned non-nil after Remove(1)")
	}

	if m.Get(2) == nil {
		t.Error("Get(2) returned nil, expected it to still exist")
	}
}

func TestManager_GetActiveUniverses(t *testing.T) {
	m := NewManager()

	u1 := m.GetOrCreate(1)
	u2 := m.GetOrCreate(2)

	// Update u1 to make it active
	u1.Update([]byte{255}, "active", [16]byte{}, 100, 1)

	// u2 has no updates, so it should be stale

	active := m.GetActiveUniverses(time.Second)

	if len(active) != 1 {
		t.Fatalf("len(GetActiveUniverses()) = %d, want 1", len(active))
	}

	if active[0].ID != 1 {
		t.Errorf("active[0].ID = %d, want 1", active[0].ID)
	}

	// Also update u2
	u2.Update([]byte{128}, "also-active", [16]byte{}, 100, 1)

	active = m.GetActiveUniverses(time.Second)

	if len(active) != 2 {
		t.Errorf("len(GetActiveUniverses()) = %d, want 2", len(active))
	}
}

func TestManager_PruneStale(t *testing.T) {
	m := NewManager()

	u1 := m.GetOrCreate(1)
	m.GetOrCreate(2) // No updates, will be stale

	u1.Update([]byte{255}, "active", [16]byte{}, 100, 1)

	// Prune with a very short timeout - both should survive initially
	pruned := m.PruneStale(time.Hour)

	// u2 has never received a packet, so LastPacket is zero, making it stale
	if pruned != 1 {
		t.Errorf("PruneStale() pruned %d, want 1", pruned)
	}

	if m.Count() != 1 {
		t.Errorf("Count() = %d, want 1 after prune", m.Count())
	}
}
