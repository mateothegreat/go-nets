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

// ConnectionStatuses is a struct that contains the stats for
// sent and received packets.
type ConnectionStatuses struct {
	OK     ConnectionStat
	Failed ConnectionStat
}

// ConnectionStat is a struct that contains the stats for
// sent and received packets.
type ConnectionStat struct {
	Packets int
	Bytes   int
}

// ConnectionStats is a struct that contains the stats for
// sent and received packets.
type ConnectionStats struct {
	mu       sync.Mutex
	Sent     ConnectionStatuses
	Received ConnectionStatuses
}

// Add adds a packet to the stats.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - ConnectionPacketStatus: The status of the packet.
//   - bytes int: The number of bytes in the packet.
func (c *ConnectionStats) AddSent(status ConnectionPacketStatus, bytes int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if status == ConnectionPacketStatusOK {
		c.Sent.OK.Packets++
		c.Sent.OK.Bytes += bytes
	} else {
		c.Sent.Failed.Packets++
		c.Sent.Failed.Bytes += bytes
	}
}

// AddReceived adds a received packet to the stats.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - ConnectionPacketStatus: The status of the packet.
//   - bytes int: The number of bytes in the packet.
func (c *ConnectionStats) AddReceived(status ConnectionPacketStatus, bytes int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if status == ConnectionPacketStatusOK {
		c.Received.OK.Packets++
		c.Received.OK.Bytes += bytes
	} else {
		c.Received.Failed.Packets++
		c.Received.Failed.Bytes += bytes
	}
}

// Get returns the number of packets with the given status.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - ConnectionPacketStatus: The status of the packet.
//
// Returns:
//   - int: The number of packets with the given status.
func (c *ConnectionStats) Get() *ConnectionStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c
}

// NewConnectionStats creates a new ConnectionStats.
func NewConnectionStats() *ConnectionStats {
	return &ConnectionStats{
		mu:       sync.Mutex{},
		Sent:     ConnectionStatuses{},
		Received: ConnectionStatuses{},
	}
}
