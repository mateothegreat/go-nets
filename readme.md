# Network Connection Management

## Features

- Easily create a connection to a remote server.
- Automatic reconnect logic.
- Send and receive packets.
- Handle connection status changes.
- Built-in support for binary.BigEndian encoding.

## Usage

### Creating a Connection

To create a connection to a remote server, use the `NewConnection` function:

```go
conn := network.NewConnection(conf.Config.Camera, fmt.Sprintf("%s:%d", sink.Addr, sink.Port))
```

### Handling Connection Status

You can handle connection status changes using the `Changed` channel:

```go
ctx, cancel := context.WithCancel(context.Background())
go func() {
  for {
    select {
    case <-ctx.Done():
      return
    case <-conn.Changed:
      multilog.Debug("main", "connection changed", map[string]interface{}{
        "status": conn.Status,
      })
      switch conn.Status {
      case network.Connected:
        multilog.Debug("main", "connection changed", map[string]interface{}{
          "status": conn.Status,
        })
        observability.ConnConnects.WithLabelValues(conf.Config.Camera).Inc()
        observability.ConnStatus.WithLabelValues(conf.Config.Camera).Set(float64(conn.Status))
      case network.Disconnected:
        multilog.Debug("main", "connection changed", map[string]interface{}{
          "status": conn.Status,
        })
        observability.ConnStatus.WithLabelValues(conf.Config.Camera).Set(float64(conn.Status))
      }
    }
  }
}()
```

### Sending and Receiving Packets

To send a packet, use the `Write` method:

```go
pkt := &network.Packet{
  ID:   "cm0ejck9q0001w4hiylut1tht",
  Type: int(types.SourceMediaTypeVideo),
  Data: []byte("{\"foo\": \"bar\"}"),
}

n, err := conn.Write(pkt)
if err != nil {
  multilog.Error("handlevideoh265", "failed to write to conn", map[string]interface{}{
    "error": err,
    "len":   len(pkt.Data),
    "addr":  conn.Addr,
  })
}
```

To receive a packet, use the `Read` method:

```go
pkt := &network.Packet{}
n, err := conn.Read(pkt)
if err != nil {
  multilog.Error("handlevideoh265", "failed to read from conn", map[string]interface{}{
    "error": err,
    "len":   n,
    "data":  conn.Addr,
  })
}
```

### Closing a Connection

To close a connection, use the `Close` method:

```go
conn.Close()
```
