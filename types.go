package nets

import (
	"context"
	"net"
	"sync"
	"time"
)

type ConnectionType string

const (
	ConnectionTypeServer ConnectionType = "server"
	ConnectionTypeClient ConnectionType = "client"
)

type Status int

func (s Status) String() string {
	switch s {
	case Disconnected:
		return "disconnected"
	case Reconnecting:
		return "reconnecting"
	case Connecting:
		return "connecting"
	case Connected:
		return "connected"
	default:
		return "unknown"
	}
}

const (
	Disconnected Status = iota
	Reconnecting
	Connecting
	Connected
)

type Error string

const (
	NewConnectionError         Error = "new connection error"
	ConnectionFailedError      Error = "connection failed"
	ConnectionResetByPeerError Error = "connection reset by peer"
	ListenError                Error = "listen error"
	ListenAcceptError          Error = "listen accept error"
	WaitForStatusTimeoutError  Error = "timeout waiting for status"
	ContextCancelledError      Error = "context cancelled"
	ConnectionClosedError      Error = "connection closed"
	ResolveTCPAddrError        Error = "resolve tcp addr error"
	PacketWriteError           Error = "packet write error"
	PacketReadError            Error = "packet read error"
	PacketDecodeError          Error = "packet decode error"
	PacketEncodeError          Error = "packet encode error"
)

type NetConn interface {
	// Connect connects to a TCP server and returns an error if it fails.
	// It will retry until it succeeds or the context is cancelled.
	//
	// Arguments:
	//   - address string: The address of the server to connect to.
	//
	// Returns:
	//   - Error: An error if the connection fails.
	Connect(string) error
	Write([]byte) (int, error)
	Close() error
	RemoteAddr() net.Addr
	Listen(context.Context, string, func(packet []byte)) error
}

// Error implements the error interface for the Error type.
func (e Error) Error() string {
	return string(e)
}

// Connection represents both a client and server connection instance.
// The type T is the type of the packet functions.
type Connection struct {
	// Customizable vars:
	Addr       string
	Timeout    time.Duration
	BufferSize int
	Channel    chan []byte
	// Callbacks:
	OnConnection func()
	OnDisconnect func()
	OnClose      func()
	OnListen     func()
	OnStatus     func(current Status, previous Status)
	OnPacket     func(packet []byte)
	OnError      func(err error, original error)
	// External vars:
	Stats *ConnectionStats
	// Internal vars:
	conn    NetConn
	status  Status
	context context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
}

// NewConnectionArgs is the arguments for the NewConnection function.
// ID, Addr, and Timeout are required.
// OnConnect, OnDisconnect, OnClose, OnStatus, OnPacket, and OnError are optional.
// If not set, they will be no-op functions.
type NewConnectionArgs struct {
	Addr         string
	Timeout      time.Duration
	BufferSize   int
	Channel      chan []byte
	Stats        bool
	OnConnection func()
	OnDisconnect func()
	OnListen     func()
	OnClose      func()
	OnStatus     func(current Status, previous Status)
	OnPacket     func(packet []byte)
	OnError      func(err error, original error)
}
