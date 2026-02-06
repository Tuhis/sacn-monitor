package sacn

import (
	"context"
	"fmt"
	"net"
	"sync"

	"golang.org/x/net/ipv4"
)

// Receiver listens for sACN packets on multicast, unicast, and broadcast
type Receiver struct {
	packets chan *Packet
	conn    *ipv4.PacketConn
	rawConn net.PacketConn
	mu      sync.RWMutex
	started bool
}

// NewReceiver creates a new sACN receiver
func NewReceiver() *Receiver {
	return &Receiver{
		packets: make(chan *Packet, 1000),
	}
}

// Packets returns the channel of received packets
func (r *Receiver) Packets() <-chan *Packet {
	return r.packets
}

// Start begins listening for sACN packets
func (r *Receiver) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return fmt.Errorf("receiver already started")
	}
	r.started = true
	r.mu.Unlock()

	// Listen on UDP port 5568 on all interfaces
	conn, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", E131Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", E131Port, err)
	}
	r.rawConn = conn

	// Create ipv4 PacketConn for multicast control
	r.conn = ipv4.NewPacketConn(conn)

	// Enable receiving multicast packets
	if err := r.conn.SetControlMessage(ipv4.FlagDst, true); err != nil {
		// Non-fatal on some platforms
		fmt.Printf("Warning: could not set control message: %v\n", err)
	}

	// Join multicast groups for commonly used universes (1-63)
	// We'll dynamically join more if we see them
	r.joinMulticastGroups(1, 63)

	// Start packet reading goroutine
	go r.readPackets(ctx)

	return nil
}

// joinMulticastGroups joins multicast groups for the given universe range
func (r *Receiver) joinMulticastGroups(startUniverse, endUniverse uint16) {
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("Warning: could not get network interfaces: %v\n", err)
		return
	}

	for universe := startUniverse; universe <= endUniverse; universe++ {
		group := multicastAddressForUniverse(universe)
		groupIP := net.ParseIP(group)
		if groupIP == nil {
			continue
		}

		for _, iface := range interfaces {
			// Skip loopback and non-multicast interfaces
			if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagMulticast == 0 {
				continue
			}
			if iface.Flags&net.FlagUp == 0 {
				continue
			}

			if err := r.conn.JoinGroup(&iface, &net.UDPAddr{IP: groupIP}); err != nil {
				// Silently ignore - some interfaces may not support multicast
			}
		}
	}
}

// multicastAddressForUniverse returns the multicast address for a given universe
// sACN multicast addresses are 239.255.{high}.{low} where universe = high*256 + low
func multicastAddressForUniverse(universe uint16) string {
	high := (universe >> 8) & 0xFF
	low := universe & 0xFF
	return fmt.Sprintf("239.255.%d.%d", high, low)
}

// readPackets continuously reads packets from the UDP socket
func (r *Receiver) readPackets(ctx context.Context) {
	buf := make([]byte, 1500) // Max UDP packet size

	for {
		select {
		case <-ctx.Done():
			r.Stop()
			return
		default:
		}

		n, _, src, err := r.conn.ReadFrom(buf)
		if err != nil {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return
			default:
				// Only log if not shutting down
				continue
			}
		}

		// Parse the packet
		packet, err := Parse(buf[:n])
		if err != nil {
			// Silently drop invalid packets
			continue
		}

		packet.SourceAddr = src

		// Try to send packet, drop if channel is full
		select {
		case r.packets <- packet:
		default:
			// Channel full, drop packet
		}
	}
}

// Stop stops the receiver
func (r *Receiver) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.rawConn != nil {
		r.rawConn.Close()
		r.rawConn = nil
	}
	r.started = false
}
