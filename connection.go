package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mateothegreat/go-multilog/multilog"
)

type ConnectionType string

const (
	ConnectionTypeServer ConnectionType = "server"
	ConnectionTypeClient ConnectionType = "client"
)

type Status int

const (
	Disconnected Status = iota
	Reconnecting
	Connecting
	Connected
)

type Connection[T PacketFuncs] struct {
	Type    ConnectionType
	Conn    *net.TCPConn
	ID      string
	Addr    string
	Timeout time.Duration
	Status  Status
	Changed chan Status
	Context context.Context
	Cancel  context.CancelFunc
	Ch      chan *T
	Mu      sync.Mutex
	Cond    *sync.Cond
	closed  bool
}

// NewConnection creates a new connection with the given ID and address.
// Callers should call Connect() or Listen() on the returned connection
// to initiate the connection process.
//
// Arguments:
//   - id: The ID of the connection.
//   - addr: The address of the connection.
//
// Returns:
//   - A new connection with the given ID and address.
func NewConnection[T PacketFuncs](id, addr string, timeout time.Duration) *Connection[T] {
	ctx, cancel := context.WithCancel(context.Background())
	ret := &Connection[T]{
		ID:      id,
		Addr:    addr,
		Status:  Disconnected,
		Timeout: timeout,
		Changed: make(chan Status),
		Context: ctx,
		Cancel:  cancel,
		Ch:      make(chan *T),
		Mu:      sync.Mutex{},
		Cond:    sync.NewCond(&sync.Mutex{}),
	}
	return ret
}

// Connect connects to a TCP server and returns an error if it fails.
// It will retry until it succeeds or the context is cancelled.
func (c *Connection[T]) Connect() error {
	ch := make(chan *T)
	defer close(ch)

	addr, err := net.ResolveTCPAddr("tcp", c.Addr)
	if err != nil {
		multilog.Error(c.ID, "resolve addr error", map[string]any{
			"addr":  c.Addr,
			"error": err,
		})
		return err
	}
	go func() {
		for {
			if c.GetStatus() == Disconnected {
				conn, err := net.DialTCP("tcp", nil, addr)
				if err != nil {
					c.SetStatus(Disconnected)
					multilog.Error(c.ID, "connect error", map[string]any{
						"addr":  c.Addr,
						"error": err,
					})
					time.Sleep(c.Timeout)
					continue
				}
				c.Conn = conn
				c.SetStatus(Connected)
				multilog.Debug(c.ID, "connected", map[string]any{
					"addr": c.Conn.RemoteAddr(),
				})
				return
			}
		}
	}()
	return nil
}

// Listen listens for incoming connections and handles them.
// It will retry every 500ms until it succeeds.
func (c *Connection[T]) Listen() error {
	if c.GetStatus() != Disconnected {
		return fmt.Errorf("connection is already in use")
	}

	addr, err := net.ResolveTCPAddr("tcp", c.Addr)
	if err != nil {
		multilog.Error(c.ID, "listen resolve addr error", map[string]any{
			"addr":  c.Addr,
			"error": err,
		})
		return err
	}

	go func() {
		for {
			select {
			case <-c.Context.Done():
				return
			default:
				if c.GetStatus() == Disconnected {
					multilog.Debug(c.ID, "listening", map[string]any{
						"addr":   c.Addr,
						"status": c.GetStatus(),
					})
					listener, err := net.ListenTCP("tcp", addr)
					if err != nil {
						multilog.Error(c.ID, "listen error", map[string]any{
							"addr":  c.Addr,
							"error": err,
						})
						c.SetStatus(Disconnected)
						c.Close()
						time.Sleep(c.Timeout)
						continue
					}

					c.SetStatus(Connected)
					multilog.Debug(c.ID, "listening", map[string]any{
						"addr": c.Addr,
					})

					for {
						conn, err := listener.AcceptTCP()
						if err != nil {
							multilog.Error(c.ID, "accept error", map[string]any{
								"addr":  c.Addr,
								"error": err,
							})
							continue
						}
						c.Conn = conn
						go c.handleConnection(conn)
					}
				}
			}
		}
	}()
	return nil
}

// handleConnection handles a new TCP connection for reading and writing packets.
func (c *Connection[T]) handleConnection(conn *net.TCPConn) {
	multilog.Debug(c.ID, "handling new connection", map[string]any{
		"addr": conn.RemoteAddr(),
	})

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			conn.Close()
			return
		}
		packet := c.NewPacket()
		err = packet.Decode(buf[:n])
		if err != nil {
			multilog.Error(c.ID, "error decoding packet", map[string]any{
				"addr":  conn.RemoteAddr(),
				"error": err,
			})
			continue
		}
		c.Ch <- &packet // Directly use packet as it is of type *T
	}
}

func (c *Connection[T]) Write(p *T) (int, error) {
	if c.GetStatus() == Connected {
		data, err := (*p).Encode()
		if err != nil {
			return 0, err
		}
		n, err := c.Conn.Write(data)
		if err != nil {
			multilog.Error(c.ID, "error writing to connection", map[string]any{
				"addr":  c.Addr,
				"error": err,
			})
			c.SetStatus(Disconnected)
			c.Close()
			go c.Connect()
			return 0, err
		}
		return n, err
	}
	return 0, fmt.Errorf("connection not connected")
}

func (c *Connection[T]) Read(b []byte) (int, *T, error) {
	if c.GetStatus() == Connected {
		n, err := c.Conn.Read(b)
		if err != nil {
			multilog.Error(c.ID, "error reading from connection", map[string]any{
				"addr":  c.Addr,
				"error": err,
			})
			c.SetStatus(Disconnected)
			c.Close()
			go c.Connect()
			return 0, nil, err
		}
		packet := c.NewPacket()
		err = packet.Decode(b[:n])
		if err != nil {
			return 0, nil, err
		}
		return n, &packet, nil
	}
	return 0, nil, nil
}

func (c *Connection[T]) Close() error {
	if c.GetStatus() == Disconnected {
		return nil
	}
	multilog.Debug(c.ID, "closing connection", map[string]any{
		"addr": c.Addr,
	})
	if c.Conn != nil {
		c.Conn.Close()
		c.Conn = nil
	}
	c.closed = true
	c.SetStatus(Disconnected)
	return nil
}

func (c *Connection[T]) WaitForStatus(status Status) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	for c.GetStatus() != status {
		c.Cond.Wait()
	}
}

func (c *Connection[T]) GetStatus() Status {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return c.Status
}

func (c *Connection[T]) SetStatus(status Status) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if c.Status == status {
		println("status already set", status)
		return
	}
	c.Status = status
}

func (c *Connection[T]) NewPacket() T {
	var packet T
	return packet
}
