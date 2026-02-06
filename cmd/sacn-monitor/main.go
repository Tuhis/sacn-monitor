package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"sacn-monitor/internal/sacn"
	"sacn-monitor/internal/stats"
	"sacn-monitor/internal/tui"
	"sacn-monitor/internal/universe"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Create components
	universeManager := universe.NewManager()
	statsTracker := stats.NewTracker()
	receiver := sacn.NewReceiver()

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle system signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Start the receiver
	if err := receiver.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting receiver: %v\n", err)
		os.Exit(1)
	}

	// Process incoming packets
	go func() {
		for packet := range receiver.Packets() {
			// Update universe state
			u := universeManager.GetOrCreate(packet.Universe)
			u.Update(
				packet.ChannelData,
				packet.SourceName,
				packet.CID,
				packet.Priority,
				packet.Sequence,
			)

			// Update stats
			statsTracker.RecordPacket(
				packet.Universe,
				packet.CID,
				packet.SourceName,
				packet.Sequence,
			)
		}
	}()

	// Create and run TUI
	model := tui.NewModel(universeManager, statsTracker)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
