package nets

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// NewTCPConnection creates a new connection with the given ID and address.
// Callers should call Connect() or Listen() on the returned connection
// to initiate the connection process.
//
// Arguments:
//   - NewConnectionArgs: The arguments to pass to the connection.
//
// Returns:
//   - *Connection
func NewTCPConnection(args NewConnectionArgs) (*Connection, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		Addr:         args.Addr,
		Timeout:      args.Timeout,
		BufferSize:   args.BufferSize,
		Channel:      args.Channel,
		status:       Disconnected,
		context:      ctx,
		cancel:       cancel,
		mu:           sync.Mutex{},
		conn:         &TCPConnection{},
		OnConnection: args.OnConnection,
		OnDisconnect: args.OnDisconnect,
		OnClose:      args.OnClose,
		OnStatus:     args.OnStatus,
		OnPacket:     args.OnPacket,
		OnError:      args.OnError,
	}, nil
}

// NewTCPConnection creates a new connection with the given ID and address.
// Callers should call Connect() or Listen() on the returned connection
// to initiate the connection process.
//
// Arguments:
//   - NewConnectionArgs: The arguments to pass to the connection.
//
// Returns:
//   - *Connection
func NewUDPConnection(args NewConnectionArgs) (*Connection, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		Addr:         args.Addr,
		Timeout:      args.Timeout,
		BufferSize:   args.BufferSize,
		Channel:      args.Channel,
		status:       Disconnected,
		context:      ctx,
		cancel:       cancel,
		mu:           sync.Mutex{},
		conn:         &UDPConnection{},
		OnConnection: args.OnConnection,
		OnDisconnect: args.OnDisconnect,
		OnClose:      args.OnClose,
		OnStatus:     args.OnStatus,
		OnPacket:     args.OnPacket,
		OnError:      args.OnError,
	}, nil
}

// Connect connects to a TCP or UDP server and returns an error if it fails.
// It will retry until it succeeds or the context is cancelled.
//
// Returns:
//   - Error: An error if the connection fails.
func (c *Connection) Connect() error {
	ch := make(chan []byte)
	defer close(ch)
	go func() {
		for {
			if c.GetStatus() == Disconnected {
				err := c.conn.Connect(c.Addr)
				if err != nil {
					c.SetStatus(Disconnected)
					if c.OnError != nil {
						c.OnError(NewConnectionError, err)
					}
					time.Sleep(c.Timeout)
					continue
				}
				c.SetStatus(Connected)
				if c.OnConnection != nil {
					c.OnConnection()
				}
				return
			}
		}
	}()
	return nil
}

// Listen listens for incoming TCP connections and handles them.
func (c *Connection) Listen() error {
	if c.GetStatus() != Disconnected {
		return fmt.Errorf("connection is already in a connected state")
	}
	go func() {
		for {
			select {
			case <-c.context.Done():
				return
			default:
				if c.GetStatus() == Disconnected {
					err := c.conn.Listen(c.context, c.Addr, func(packet []byte) {
						if c.OnPacket != nil {
							c.OnPacket(packet)
						}
						if c.Channel != nil {
							c.Channel <- packet
						}
					})
					if err != nil {
						c.error(ListenError, err)
						c.SetStatus(Disconnected)
						c.Close()
						if c.OnClose != nil {
							c.OnClose()
						}
						time.Sleep(c.Timeout)
						continue
					}
					if c.OnListen != nil {
						c.OnListen()
					}
					c.SetStatus(Connected)
				}
			}
		}
	}()
	return nil
}

// Write writes a packet to the connection.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - p: The packet to write to the connection.
//
// Returns:
//   - int: The number of bytes written.
//   - error: An error if the write fails.
func (c *Connection) Write(p []byte) (int, error) {
	if c.GetStatus() == Connected {
		if len(p) > c.BufferSize {
			return 0, c.error(PacketWriteError, fmt.Errorf("packet is too large for buffer size %d", c.BufferSize))
		}
		n, err := c.conn.Write(p)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "write: broken pipe" {
				c.Close()
				c.Connect()
				return 0, c.error(ConnectionResetByPeerError, err)
			}
			return 0, c.error(PacketWriteError, err)
		}
		return n, nil
	}
	return 0, c.error(ConnectionClosedError, fmt.Errorf("cannot write to connection: connection is not initialized"))
}

// WaitForStatus waits for the connection to reach the given status.
// It is recommended to use OnStatus instead of this function.
//
// Arguments:
//   - Status: The status to wait for.
//   - timeout: The maximum time to wait for the status.
func (c *Connection) WaitForStatus(status Status, timeout time.Duration) error {
	if c.GetStatus() == status {
		return nil
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return c.error(WaitForStatusTimeoutError, fmt.Errorf("timeout waiting for status %d within %d", status, timeout))
		case <-c.context.Done():
			return ContextCancelledError
		default:
			if c.GetStatus() == status {
				return nil
			}
		}
	}
}

// GetStatus returns the current status of the connection.
// It is safe to call this function from multiple goroutines.
//
// Returns:
//   - Status: The current status of the connection.
func (c *Connection) GetStatus() Status {
	if c.mu.TryLock() {
		defer c.mu.Unlock()
		return c.status
	}
	return c.status
}

// SetStatus sets the status of the connection and notifies any waiters
// that the status has changed if the status is different and the status
// channel arg is not nil.
//
// Arguments:
//   - Status: The status to set the connection to.
func (c *Connection) SetStatus(status Status) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.status == status {
		return
	}
	if c.OnStatus != nil {
		c.OnStatus(status, c.status)
	}
	c.status = status
}

// Close closes the connection and sets the status to Disconnected.
// It is safe to call this function from multiple goroutines.
//
// Returns:
//   - Error: An error if the connection fails to close.
func (c *Connection) Close() error {
	if c.GetStatus() == Disconnected {
		return nil
	}
	if c.conn != nil {
		c.cancel()
		if err := c.conn.Close(); err != nil {
			return c.error(ConnectionClosedError, err)
		}
		c.conn = nil
	}
	c.SetStatus(Disconnected)
	if c.OnClose != nil {
		c.OnClose()
	}
	return nil
}

// error is a helper function to call the OnError callback if it is set.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - err: The error to pass to the OnError callback.
//   - original: The original error that caused the error.
//
// Returns:
func (c *Connection) error(err error, original error) error {
	if c.OnError != nil {
		c.OnError(err, original)
	}
	return err
}
