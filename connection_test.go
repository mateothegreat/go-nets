package nets

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type SimpleStruct struct {
	Foo string `json:"foo"`
}

func (s *SimpleStruct) Encode() ([]byte, error) {
	return json.Marshal(s)
}

func (s *SimpleStruct) Decode(data []byte) error {
	err := json.Unmarshal(data, s)
	if err != nil {
		fmt.Printf("error decoding: %v\n", err)
		return err
	}
	return nil
}

type TestSourceSuite struct {
	suite.Suite
	addr string
}

func TestSuiteTest(t *testing.T) {
	suite.Run(t, new(TestSourceSuite))
}

func (suite *TestSourceSuite) SetupTest() {
	suite.addr = "127.0.0.1:5200"
}

func (suite *TestSourceSuite) TestTCPConnection() {
	// Create a context and cancel function to control the lifetime of the
	// test which will be used to stop the goroutines like listening for packets.
	ctx, cancel := context.WithCancel(context.Background())

	// Create the server.
	server, err := NewTCPConnection(NewConnectionArgs{
		Addr:       suite.addr,
		BufferSize: 13,
		Timeout:    500 * time.Millisecond,
		Channel:    make(chan []byte),
		Stats:      true,
		OnClose: func() {
		},
		OnStatus: func(current Status, previous Status) {
			suite.T().Logf("server status: %s -> %s", previous, current)
		},
		OnConnection: func() {
			suite.T().Log("server received connection from client")
		},
		OnListen: func() {
			suite.T().Log("server listening")
		},
		OnPacket: func(packet []byte) {
			suite.T().Logf("server received packet from client: %s", string(packet))
		},
		OnError: func(err error, original error) {
			suite.T().Logf("OnError: %s, %s", err.Error(), original.Error())
		},
	})
	suite.NoError(err)

	// Start the server.
	suite.NoError(server.Listen())

	// Wait for the server to connect to the client.
	suite.NoError(server.WaitForStatus(Connected, 1*time.Second))

	// Next, we create the client.
	client, err := NewTCPConnection(NewConnectionArgs{
		Addr:       suite.addr,
		BufferSize: 13,
		Timeout:    500 * time.Millisecond,
		Stats:      true,
		OnStatus: func(current Status, previous Status) {
			suite.T().Logf("OnStatus: client status: %s -> %s", previous, current)
		},
	})
	suite.NoError(err)

	// Connect the client to the server.
	suite.NoError(client.Connect())

	// Wait for the client to connect to the server.
	suite.NoError(client.WaitForStatus(Connected, 1*time.Second))

	// Now we start a goroutine to listen for packets from the server.
	go func() {
		timeout := time.After(5 * time.Second) // Set the timeout duration
		for {
			select {
			// If the context is done, we should exit the loop and goroutine.
			case <-ctx.Done():
				return
			// If the timeout occurs, we should cancel the context and stop the goroutine.
			case <-timeout:
				suite.T().Error("timeout waiting for packet")
				cancel()
				return
			// If we receive a packet, we should check the value and cancel the
			// context to stop the goroutine.
			case packet := <-server.Channel:
				suite.T().Logf("client received packet from server: %s", string(packet))
				suite.Equal("bar", string(packet))
				// We cancel the context and stop the goroutine so the test can finish.
				cancel()
				return
			}
		}
	}()

	// We send a packet to the server.
	n, err := client.Write([]byte("bar"))
	suite.NoError(err)
	suite.Equal(3, n)

	// We wait for the goroutine to finish before checking the status.
	<-ctx.Done()

	// We close the server and client.
	suite.NoError(server.Close())
	suite.NoError(client.Close())

	// Check the stats.
	suite.Equal(1, client.Stats.Sent.OK.Packets)
	suite.Equal(3, client.Stats.Sent.OK.Bytes)
	suite.Equal(1, server.Stats.Received.OK.Packets)
	suite.Equal(3, server.Stats.Received.OK.Bytes)

	// We check the status of the server and client.
	suite.Equal(Disconnected, server.GetStatus())
	suite.Equal(Disconnected, client.GetStatus())
}

func (suite *TestSourceSuite) TestUDPConnection() {
	// Create a context and cancel function to control the lifetime of the
	// test which will be used to stop the goroutines like listening for packets.
	ctx, cancel := context.WithCancel(context.Background())

	// Create the server.
	server, err := NewUDPConnection(NewConnectionArgs{
		Addr:       suite.addr,
		BufferSize: 13,
		Timeout:    500 * time.Millisecond,
		Stats:      true,
		Channel:    make(chan []byte),
		OnClose: func() {
		},
		OnStatus: func(current Status, previous Status) {
			suite.T().Logf("OnStatus: server status: %s -> %s", previous, current)
		},
		OnConnection: func() {
			suite.T().Log("OnConnection: server received connection from client")
		},
		OnListen: func() {
			suite.T().Log("OnListen: server listening")
		},
		OnPacket: func(packet []byte) {
			suite.T().Logf("OnPacket: server received packet from client: %s", string(packet))
		},
		OnError: func(err error, original error) {
			suite.T().Logf("OnError: %s, %s", err.Error(), original.Error())
		},
	})
	suite.NoError(err)

	// Start the server.
	suite.NoError(server.Listen())

	// Wait for the server to connect to the client.
	suite.NoError(server.WaitForStatus(Connected, 1*time.Second))

	// Next, we create the client.
	client, err := NewUDPConnection(NewConnectionArgs{
		Addr:       suite.addr,
		BufferSize: 13,
		Timeout:    500 * time.Millisecond,
		Stats:      true,
		OnStatus: func(current Status, previous Status) {
			suite.T().Logf("OnStatus: client status: %s -> %s", previous, current)
		},
	})
	suite.NoError(err)

	// Connect the client to the server.
	suite.NoError(client.Connect())

	// Wait for the client to connect to the server.
	suite.NoError(client.WaitForStatus(Connected, 1*time.Second))

	// Now we start a goroutine to listen for packets from the server.
	go func() {
		timeout := time.After(1 * time.Second) // Set the timeout duration
		for {
			select {
			// If the context is done, we should exit the loop and goroutine.
			case <-ctx.Done():
				return
			// If the timeout occurs, we should cancel the context and stop the goroutine.
			case <-timeout:
				suite.T().Error("timeout waiting for packet")
				cancel()
				return
			// If we receive a packet, we should check the value and cancel the
			// context to stop the goroutine.
			case packet := <-server.Channel:
				suite.T().Logf("client received packet from server: %s", string(packet))
				suite.Equal("bar", string(packet))
				// We cancel the context and stop the goroutine so the test can finish.
				cancel()
				return
			}
		}
	}()

	// We send a packet to the server.
	n, err := client.Write([]byte("bar"))
	suite.NoError(err)
	suite.Equal(3, n)

	// We wait for the goroutine to finish before checking the status.
	<-ctx.Done()

	// Check the stats.
	suite.Equal(1, client.Stats.Sent.OK.Packets)
	suite.Equal(3, client.Stats.Sent.OK.Bytes)
	suite.Equal(1, server.Stats.Received.OK.Packets)
	suite.Equal(3, server.Stats.Received.OK.Bytes)

	// We close the server and client.
	suite.NoError(server.Close())
	suite.NoError(client.Close())

	// We check the status of the server and client.
	suite.Equal(Disconnected, server.GetStatus())
	suite.Equal(Disconnected, client.GetStatus())
}
