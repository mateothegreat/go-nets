# `nets` 🆒

Simple library for managing network connections in Go.

## Features

- Easily create a tcp server or client in one line.
- Automagic reconnect logic under the hood ♻️.
- Send and receive packets with any encoding you want.
- Handles connection status changes automatically.
- Configurable event handlers 🦸‍♂️.
- **No dependencies.**

## Installation

```bash
go get github.com/mateothegreat/go-nets@latest
```

## Usage

### Creating a Connection

To create a connection to a remote server, use the `NewConnection` method and pass in a `NewConnectionArgs` struct.

First, let's define a simple struct that we will use to send and receive data and an encoder/decoder for that struct
so the connection manager knows how to encode and decode the data:

```go
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
```

Here is an example of a server and client that communicate with each other:

```go
ch := make(chan *SimpleStruct)

server, err := nets.NewConnection(nets.NewConnectionArgs[*SimpleStruct]{
  Addr:     "127.0.0.1:5200",
  Timeout:  500 * time.Millisecond,
  Messages: ch,
  OnClose: func() {
   // do something when the connection is closed
  },
  OnStatus: func(status Status) {
   if status == Connected {
    // do something when the status changes to connected
   } else if status == Disconnected {
    // do something when the status changes to disconnected
   }
  },
  OnConnection: func(addr net.Addr) {
   // do something when the connection is established
  },
  OnPacket: func(packet *SimpleStruct) {
   // do something when a packet is received
  },
  OnError: func(err error, original error) {
   // do something when an error occurs
  },
})

if err := server.Listen(); err != nil {
  log.Printf("error listening: %v", err)
}

ctx, cancel := context.WithCancel(context.Background())

go func() {
  for {
    select {
    // If the context is done, we should exit the loop and goroutine.
    case <-ctx.Done():
    return
    // If we receive a packet, we should check the value and cancel the
    // context to stop the goroutine.
    case packet := <-ch:
    suite.Equal("bar", (*packet).Foo)
    // We cancel the context and stop the goroutine so things can finish outside of the goroutine.
    cancel()
    return
    }
  }
}()

// Create a client that will send a message to the server.
client, err := nets.NewConnection(nets.NewConnectionArgs[*SimpleStruct]{
  ID:      "sender",
  Addr:    "127.0.0.1:5200",
  Timeout: 500 * time.Millisecond,
  Messages: ch,
})

// We connect the client to the server.
if err := client.Connect(); err != nil {
  log.Printf("error connecting: %v", err)
}

// We wait for the client to connect to the server.
if err := client.WaitForStatus(nets.Connected, 1*time.Second); err != nil {
  log.Printf("error waiting for status: %v", err)
}

n, err := client.Write(&SimpleStruct{Foo: "bar"})
if err != nil {
  log.Printf("error writing: %v", err)
}
log.Printf("client sent %d bytes", n)

// We close the server and client.
server.Close()
client.Close()

// We wait for the goroutine to finish before checking the status.
<-ctx.Done()

log.Printf("server status: %s", server.GetStatus())
log.Printf("client status: %s", client.GetStatus())
```

Here is a minimal example of a server and client that communicate with each other:

```go
conn := nets.NewConnection(nets.NewConnectionArgs[*SimpleStruct]{
  Addr:     "127.0.0.1:5200",
  Timeout:  500 * time.Millisecond,
  Messages: make(chan *SimpleStruct),
})

if err := conn.Listen(); err != nil {
  log.Printf("error listening: %v", err)
}
```
