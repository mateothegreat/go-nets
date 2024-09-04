package nets

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
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
}

func TestSuiteTest(t *testing.T) {
	suite.Run(t, new(TestSourceSuite))
}

func (suite *TestSourceSuite) TestConnection() {
	// We create a context and cancel function to control the lifetime of the
	// test which will be used to stop the goroutines like listening for packets.
	ctx, cancel := context.WithCancel(context.Background())

	// First, we create the server.
	server, err := NewConnection(NewConnectionArgs[*SimpleStruct]{
		ID:       "receiver",
		Addr:     "127.0.0.1:5200",
		Timeout:  500 * time.Millisecond,
		Messages: make(chan *SimpleStruct),
		OnClose: func() {
		},
		OnStatus: func(status Status) {
			if status == Connected {
				suite.T().Log("server listening")
			}
		},
		OnConnection: func(addr net.Addr) {
			suite.T().Log("server connection", addr)
		},
		OnPacket: func(packet *SimpleStruct) {
			suite.T().Log("server packet", packet)
		},
		OnError: func(err error, original error) {
			suite.T().Log("OnError", err)
		},
	})

	suite.NoError(err)

	// We start the server.
	suite.NoError(server.Listen())

	// We wait for the server to connect to the client.
	suite.NoError(server.WaitForStatus(Connected, 1*time.Second))

	// Next, we create the client.
	client, err := NewConnection(NewConnectionArgs[*SimpleStruct]{
		ID:      "sender",
		Addr:    "127.0.0.1:5200",
		Timeout: 500 * time.Millisecond,
	})
	suite.NoError(err)

	// We connect the client to the server.
	suite.NoError(client.Connect())

	// We wait for the client to connect to the server.
	suite.NoError(client.WaitForStatus(Connected, 1*time.Second))

	// Now we start a goroutine to listen for packets from the server.
	go func() {
		for {
			select {
			// If the context is done, we should exit the loop and goroutine.
			case <-ctx.Done():
				return
			// If we receive a packet, we should check the value and cancel the
			// context to stop the goroutine.
			case packet := <-server.Messages:
				suite.Equal("bar", (*packet).Foo)
				// We cancel the context and stop the goroutine so the test can finish.
				cancel()
				return
			}
		}
	}()

	// We send a packet to the server.
	n, err := client.Write(&SimpleStruct{Foo: "bar"})
	suite.NoError(err)
	suite.Equal(13, n)

	// We close the server and client.
	suite.NoError(server.Close())
	suite.NoError(client.Close())

	// We wait for the goroutine to finish before checking the status.
	<-ctx.Done()

	// We check the status of the server and client.
	suite.Equal(Disconnected, server.GetStatus())
	suite.Equal(Disconnected, client.GetStatus())
}
