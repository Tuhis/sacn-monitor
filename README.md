# sACN Monitor

A terminal UI (TUI) application for monitoring sACN/E1.31 traffic during lighting development.

## Features

- Monitor all sACN universes simultaneously
- Real-time 512-channel grid visualization per universe
- Distinguish between active channels (receiving data) and inactive channels
- Packet rate monitoring
- Source identification (CID, Source Name)
- Packet loss detection via sequence number gaps
- Support for multicast, unicast, and broadcast traffic

## Installation

```bash
go install sacn-monitor/cmd/sacn-monitor@latest
```

## Usage

```bash
sacn-monitor
```

### Keyboard Controls

- `Tab` / `Shift+Tab` - Navigate between universes
- `↑↓←→` - Scroll channel grid
- `q` - Quit

## Building from Source

```bash
go build -o sacn-monitor ./cmd/sacn-monitor
```

## License

MIT
