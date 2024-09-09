package nets

import (
	"context"
	"fmt"
	"net"
)

type UDPConnection struct {
	conn *net.UDPConn
	addr *net.UDPAddr
}

func (u *UDPConnection) Connect(address string) error {
	if u.addr == nil {
		addr, err := net.ResolveUDPAddr("udp", address)
		if err != nil {
			return err
		}
		u.addr = addr
	}
	conn, err := net.DialUDP("udp", nil, u.addr)
	if err != nil {
		return err
	}
	u.conn = conn
	return nil
}

func (u *UDPConnection) Write(b []byte) (int, error) {
	if u.conn == nil {
		return 0, fmt.Errorf("cannot write to connection: connection is not initialized")
	}
	return u.conn.Write(b)
}

func (u *UDPConnection) Close() error {
	if u.conn == nil {
		return nil
	}
	return u.conn.Close()
}

func (u *UDPConnection) RemoteAddr() net.Addr {
	return u.addr
}

func (u *UDPConnection) Listen(ctx context.Context, address string, onPacket func(packet []byte)) error {
	if u.addr == nil {
		addr, err := net.ResolveUDPAddr("udp", address)
		if err != nil {
			return err
		}
		u.addr = addr
	}
	listener, err := net.ListenUDP("udp", u.addr)
	if err != nil {
		return err
	}
	u.conn = listener
	go func() {
		buf := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := u.conn.Read(buf)
				if err != nil {
					continue
				}
				onPacket(buf[:n])
			}
		}
	}()
	return nil
}
