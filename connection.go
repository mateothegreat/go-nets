package nets

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// NewConnection creates a new connection with the given ID and address.
// Callers should call Connect() or Listen() on the returned connection
// to initiate the connection process.
//
// Arguments:
//   - NewConnectionArgs[T]: The arguments to pass to the connection.
//
// Returns:
//   - *Connection[T]
func NewConnection(args NewConnectionArgs) (*Connection, error) {
	ctx, cancel := context.WithCancel(context.Background())
	ret := &Connection{
		// Required args:
		Addr:       args.Addr,
		Timeout:    args.Timeout,
		BufferSize: args.BufferSize,
		// Internal vars:
		status:  Disconnected,
		context: ctx,
		cancel:  cancel,
		mu:      sync.Mutex{},
	}

	if args.Addr == "" {
		return nil, fmt.Errorf("addr argument is required")
	}
	if args.Timeout <= 1 {
		return nil, fmt.Errorf("timeout duration argument must be greater than 1")
	}
	if args.Messages != nil {
		ret.Packets = args.Messages
	}

	if args.BufferSize <= 0 {
		ret.BufferSize = 1024 * 1024
	}

	// Set optional callbacks
	if args.OnConnection != nil {
		ret.OnConnection = args.OnConnection
	}
	if args.OnDisconnect != nil {
		ret.OnDisconnect = args.OnDisconnect
	}
	if args.OnClose != nil {
		ret.OnClose = args.OnClose
	}
	if args.OnListen != nil {
		ret.OnListen = args.OnListen
	}
	if args.OnStatus != nil {
		ret.OnStatus = args.OnStatus
	}
	if args.OnPacket != nil {
		ret.OnPacket = args.OnPacket
	}
	if args.OnError != nil {
		ret.OnError = args.OnError
	}

	return ret, nil
}

// Connect connects to a TCP server and returns an error if it fails.
// It will retry until it succeeds or the context is cancelled.
//
// Returns:
//   - Error: An error if the connection fails.
func (c *Connection) Connect() error {
	ch := make(chan []byte)
	defer close(ch)

	addr, err := net.ResolveTCPAddr("tcp", c.Addr)
	if err != nil {
		return c.error(ResolveTCPAddrError, err)
	}
	go func() {
		for {
			if c.GetStatus() == Disconnected {
				conn, err := net.DialTCP("tcp", nil, addr)
				if err != nil {
					c.SetStatus(Disconnected)
					if c.OnError != nil {
						c.OnError(NewConnectionError, err)
					}
					// Continue to the next iteration of the loop for retry.
					time.Sleep(c.Timeout)
					continue
				}
				c.conn = conn
				c.SetStatus(Connected)
				if c.OnConnection != nil {
					c.OnConnection(conn.RemoteAddr())
				}
				return
			}
		}
	}()
	return nil
}

// Listen listens for incoming connections and handles them.
// It will retry every 500ms until it succeeds.
func (c *Connection) Listen() error {
	if c.GetStatus() != Disconnected {
		return fmt.Errorf("connection is already in a connected state")
	}

	addr, err := net.ResolveTCPAddr("tcp", c.Addr)
	if err != nil {
		return c.error(ResolveTCPAddrError, err)
	}

	go func() {
		for {
			select {
			case <-c.context.Done():
				return
			default:
				if c.GetStatus() == Disconnected {
					listener, err := net.ListenTCP("tcp", addr)
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

					for {
						select {
						case <-c.context.Done():
							return
						default:
							conn, err := listener.AcceptTCP()
							if err != nil {
								c.error(ListenAcceptError, err)
								continue
							}
							c.conn = conn
							if c.OnConnection != nil {
								c.OnConnection(conn.RemoteAddr())
							}
							go c.handleConnection(conn)
						}
					}
				}
			}
		}
	}()
	return nil
}

// handleConnection handles a new TCP connection for reading and writing packets.
// This is a blocking function that will continue until the connection is closed
// or an error occurs and should be run as a goroutine to not block the main thread.

// Arguments:
//   - *net.TCPConn: The TCP connection to handle.
func (c *Connection) handleConnection(conn *net.TCPConn) {
	buf := make([]byte, c.BufferSize)
	for {
		select {
		case <-c.context.Done():
			return
		default:
			n, err := conn.Read(buf)
			if err != nil {
				c.error(PacketReadError, err)
				return
			}
			if c.Packets != nil {
				c.Packets <- buf[:n]
			}
			if c.OnPacket != nil {
				c.OnPacket(buf[:n])
			}
		}
	}
}

// Write writes a packet to the connection.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - p: The packet to write to the connection.
//
// Returns:
//   - int: The number of bytes written.
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
	return 0, c.error(ConnectionClosedError, fmt.Errorf("connection is not connected"))
}

// Read reads a packet from the connection.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - b: The buffer to read the packet into.
//
// Returns:
//   - int: The number of bytes read.
func (c *Connection) Read(b []byte) (int, []byte, error) {
	if c.GetStatus() == Connected {
		if len(b) > c.BufferSize {
			return 0, nil, c.error(PacketReadError, fmt.Errorf("buffer size %d is too large for packet size %d", c.BufferSize, len(b)))
		}
		n, err := c.conn.Read(b)
		if err != nil {
			c.SetStatus(Disconnected)
			c.Close()
			go c.Connect()
			return 0, nil, c.error(ConnectionResetByPeerError, err)
		}
		return n, b[:n], nil
	}
	return 0, nil, nil
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
		c.OnStatus(status)
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

// NewPacket creates a new packet of type []byte.
// This is used when decoding a packet to create a new packet of the correct type.
//
// Returns:
//   - []byte: A new packet of type []byte.
func (c *Connection) NewPacket() []byte {
	var packet []byte
	return packet
}
