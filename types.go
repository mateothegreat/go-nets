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

// Error implements the error interface for the Error type.
func (e Error) Error() string {
	return string(e)
}

// Connection represents both a client and server connection instance.
// The type T is the type of the packet functions.
type Connection struct {
	// Customizable vars:
	ID         string
	Addr       string
	Timeout    time.Duration
	BufferSize int
	Packets    chan []byte
	// Callbacks:
	OnConnection func(net.Addr)
	OnDisconnect func(net.Addr)
	OnClose      func()
	OnListen     func()
	OnStatus     func(status Status)
	OnPacket     func(packet []byte)
	OnError      func(err error, original error)
	// Internal vars:
	conn    *net.TCPConn
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
	ID           string
	Addr         string
	Timeout      time.Duration
	BufferSize   int
	Messages     chan []byte
	OnConnection func(net.Addr)
	OnDisconnect func(net.Addr)
	OnListen     func()
	OnClose      func()
	OnStatus     func(status Status)
	OnPacket     func(packet []byte)
	OnError      func(err error, original error)
}
