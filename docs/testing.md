# Testing Strategy

## Overview

sacn-monitor uses **Test-Driven Development (TDD)** with comprehensive unit tests for all core components.

## Test Summary

| Package | Tests | Coverage Focus |
|---------|-------|----------------|
| `internal/sacn` | 12 | Packet parsing, validation |
| `internal/universe` | 14 | Channel state, staleness |
| `internal/stats` | 13 | Rate, loss detection |
| **Total** | **39** | |

---

## Running Tests

```powershell
# All tests
go test ./...

# Verbose output
go test ./... -v

# With coverage
go test ./... -cover

# Specific package
go test ./internal/sacn/... -v
```

---

## Test Categories

### Parser Tests (`internal/sacn/parser_test.go`)

Tests E1.31 packet parsing:

| Test | Purpose |
|------|---------|
| `TestParse_ValidPacket` | Correct field extraction |
| `TestParse_MaxChannels` | 512 channel handling |
| `TestParse_EmptyChannelData` | Zero-length DMX data |
| `TestParse_PacketTooShort` | Reject truncated packets |
| `TestParse_Invalid*` | Reject malformed headers |

**Helper**: `buildValidPacket()` constructs test packets.

### Universe Tests (`internal/universe/universe_test.go`)

Tests state management:

| Test | Purpose |
|------|---------|
| `TestUniverse_Update` | Channel value updates |
| `TestUniverse_ActiveChannelCount` | Active tracking |
| `TestUniverse_IsStale` | Timeout detection |
| `TestManager_GetAll_Sorted` | Sorted universe list |
| `TestManager_PruneStale` | Cleanup old universes |

### Stats Tests (`internal/stats/tracker_test.go`)

Tests statistics calculation:

| Test | Purpose |
|------|---------|
| `TestTracker_PacketLossDetection_SimpleGap` | Sequence gaps |
| `TestTracker_PacketLossDetection_Wraparound` | 255→0 wrap |
| `TestTracker_GetPacketRate` | Rate calculation |
| `TestTracker_MultipleSources` | Multi-source tracking |

---

## Writing New Tests

### Template

```go
func TestFeature_Scenario(t *testing.T) {
    // Arrange
    tracker := stats.NewTracker()
    
    // Act
    tracker.RecordPacket(1, cid, "test", 0)
    
    // Assert
    if got := tracker.GetPacketRate(1); got != expected {
        t.Errorf("GetPacketRate() = %v, want %v", got, expected)
    }
}
```

### Table-Driven Tests

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    []byte
        expected int
    }{
        {"empty", []byte{}, 0},
        {"one", []byte{255}, 1},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

---

## Integration Testing

Manual integration testing with tlight-commander-go:

1. Start tlight-commander-go (sends E1.31 on universe 1)
2. Run sacn-monitor
3. Verify:
   - Universe 1 appears
   - Channel values display correctly
   - Rate shows ~44-50 pps
   - Source name shows "tlight-commander-go"

---

## Future Improvements

- [ ] Add integration tests with mock UDP sender
- [ ] Add benchmarks for parser performance
- [ ] Add fuzzing for packet parsing
