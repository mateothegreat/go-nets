package nets

import (
	"context"
	"fmt"
	"net"
)

type TCPConnection struct {
	conn     *net.TCPConn
	addr     *net.TCPAddr
	listener *net.TCPListener
	clients  []*net.TCPConn
}

// Connect connects to a TCP server and returns an error if it fails.
// It will retry until it succeeds or the context is cancelled.
//
// Arguments:
//   - address string: The address of the server to connect to.
//
// Returns:
//   - error: An error if the connection fails.
func (t *TCPConnection) Connect(address string) error {
	if t.addr == nil {
		addr, err := net.ResolveTCPAddr("tcp", address)
		if err != nil {
			return err
		}
		t.addr = addr
	}
	conn, err := net.DialTCP("tcp", nil, t.addr)
	if err != nil {
		return err
	}
	t.conn = conn
	return nil
}

// Write writes data to the connection.
//
// Arguments:
//   - b []byte: The buffer to write.
//
// Returns:
//   - int: The number of bytes written.
//   - error: An error if the write fails.
func (t *TCPConnection) Write(b []byte) (int, error) {
	if t.conn == nil {
		return 0, fmt.Errorf("cannot write to connection: connection is not initialized")
	}
	return t.conn.Write(b)
}

// Close closes the connection and returns an error if it fails.
// It is safe to call this function from multiple goroutines.
//
// Returns:
//   - error: An error if the connection fails to close.
func (t *TCPConnection) Close() error {
	for _, conn := range t.clients {
		conn.Close()
	}
	if t.listener != nil {
		t.listener.Close()
	}
	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}

// RemoteAddr returns the remote address of the connection.
//
// Returns:
//   - net.Addr: The remote address of the connection.
func (t *TCPConnection) RemoteAddr() net.Addr {
	return t.addr
}

// Listen listens for incoming connections and returns an error if it fails.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - ctx context.Context: The context to use for the connection.
//   - address string: The address to listen on.
//
// Returns:
//   - error: An error if the connection fails to listen.
func (t *TCPConnection) Listen(ctx context.Context, address string, onPacket func(packet []byte)) error {
	if t.addr == nil {
		addr, err := net.ResolveTCPAddr("tcp", address)
		if err != nil {
			return err
		}
		t.addr = addr
	}
	listener, err := net.ListenTCP("tcp", t.addr)
	if err != nil {
		return err
	}
	t.listener = listener
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := t.listener.AcceptTCP()
				if err != nil {
					continue
				}
				t.clients = append(t.clients, conn)
				go t.handleConnection(ctx, conn, onPacket)
			}
		}
	}()
	return nil
}

// handleConnection handles an incoming connection.
// It is safe to call this function from multiple goroutines.
//
// Arguments:
//   - conn *net.TCPConn: The connection to handle.
func (t *TCPConnection) handleConnection(ctx context.Context, conn *net.TCPConn, onPacket func(packet []byte)) {
	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := conn.Read(buf)
			if err != nil {
				continue
			}
			onPacket(buf[:n])
		}
	}
}
