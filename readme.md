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

Creating a connection:

| Method             | Description                 |
| ------------------ | --------------------------- |
| `NewTCPConnection` | Create a new TCP connection |
| `NewUDPConnection` | Create a new UDP connection |

Connection methods:

| Method          | Description                     |
| --------------- | ------------------------------- |
| `WaitForStatus` | Wait for a specific status      |
| `Listen`        | Listen for incoming connections |
| `Connect`       | Connect to a remote server      |
| `Close`         | Close the connection            |
| `Write`         | Write data to the connection    |
| `Read`          | Read data from the connection   |

### Creating a Connection

To create a connection to a remote server, use the `NewTCPConnection` method and pass in a `NewConnectionArgs` struct.

Here is an example of a server and client that communicate with each other:

```go
ch := make(chan []byte)

server, err := nets.NewTCPConnection(nets.NewConnectionArgs{
  Addr:       "127.0.0.1:5200",
  BufferSize: 13,
  Timeout:    500 * time.Millisecond,
  Channel:    make(chan []byte),
  OnClose: func() {
  },
  OnStatus: func(current Status, previous Status) {
   log.Printf("server status: %s -> %s", previous, current)
  },
  OnConnection: func() {
   log.Printf("OnConnection: server received connection from client")
  },
  OnListen: func() {
   log.Printf("OnListen: server listening")
  },
  OnPacket: func(packet []byte) {
   log.Printf("OnPacket: server received packet from client: %s", string(packet))
  },
  OnError: func(err error, original error) {
   log.Printf("OnError: %s, %s", err.Error(), original.Error())
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
      log.Printf("OnPacket: received packet: %s", packet.Foo)
      // We cancel the context and stop the goroutine so things can finish outside of the goroutine.
      cancel()
      return
    }
  }
}()

// Create a client that will send a message to the server.
client, err := nets.NewTCPConnection(nets.NewConnectionArgs{
  Addr:       "127.0.0.1:5200",
  BufferSize: 13,
  Timeout:    500 * time.Millisecond,
  OnStatus: func(current Status, previous Status) {
    log.Printf("OnStatus: client status: %s -> %s", previous, current)
  },
})

// We connect the client to the server.
if err := client.Connect(); err != nil {
  log.Printf("error connecting: %v", err)
}

// Optionally, we can wait for the client to connect to the server.
if err := client.WaitForStatus(Connected, 1*time.Second); err != nil {
  log.Printf("error waiting for status: %v", err)
}

n, err := client.Write(&SimpleStruct{Foo: "bar"})
if err != nil {
  log.Printf("error writing: %v", err)
}

// Close the server and client.
server.Close()
client.Close()

// We wait for the goroutine to finish before checking the status.
<-ctx.Done()

log.Printf("server status: %s", server.GetStatus())
log.Printf("client status: %s", client.GetStatus())
```

Here is a minimal example of a server and client that communicate with each other:

```go
conn := nets.NewTCPConnection(nets.NewConnectionArgs{
  Addr:       "127.0.0.1:5200",
  BufferSize: 13,
  Timeout:    500 * time.Millisecond,
  Channel:    make(chan []byte),
})

if err := conn.Listen(); err != nil {
  log.Printf("error listening: %v", err)
}
```
