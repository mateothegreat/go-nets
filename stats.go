package nets

import (
	"sync"
)

// ConnectionPacketStatus is a type that represents the status of a packet.
type ConnectionPacketStatus string

const (
	// ConnectionPacketStatusOK is the status of a packet that was sent or received successfully.
	ConnectionPacketStatusOK ConnectionPacketStatus = "ok"
	// ConnectionPacketStatusFailed is the status of a packet that failed to be sent or received.
	ConnectionPacketStatusFailed ConnectionPacketStatus = "failed"
)

// ConnectionStats is a struct that contains the stats for
// sent and received packets.
type ConnectionStats struct {
	mu       sync.Mutex
	sent     map[ConnectionPacketStatus]int
	received map[ConnectionPacketStatus]int
}

// Add adds a packet to the stats.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - ConnectionPacketStatus: The status of the packet.
func (c *ConnectionStats) Add(status ConnectionPacketStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent[status]++
}

// AddReceived adds a received packet to the stats.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - ConnectionPacketStatus: The status of the packet.
func (c *ConnectionStats) AddReceived(status ConnectionPacketStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.received[status]++
}

// Get returns the number of packets with the given status.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - ConnectionPacketStatus: The status of the packet.
//
// Returns:
//   - int: The number of packets with the given status.
func (c *ConnectionStats) Get(status ConnectionPacketStatus) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if status == ConnectionPacketStatusOK {
		return c.sent[status]
	}
	return c.received[status]
}

// NewConnectionStats creates a new ConnectionStats struct.
//
// Returns:
//   - *ConnectionStats: A new ConnectionStats struct.
func NewConnectionStats() *ConnectionStats {
	return &ConnectionStats{
		sent:     make(map[ConnectionPacketStatus]int),
		received: make(map[ConnectionPacketStatus]int),
	}
}
