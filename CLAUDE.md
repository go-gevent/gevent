# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **go-gevent**, a high-performance network library for Go based on epoll event loops. It's designed for building scalable network servers with TCP and UDP support, inspired by event-driven architectures.

## Development Commands

### Building and Testing
```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run specific test file
go test -v ./gevent_test.go

# Run benchmarks
go test -bench=. ./...

# Build the library
go build ./...
```

### Running Examples
```bash
# Run the echo TCP server example
go run example/echo_tcp/main.go -port 5007 -loops 4

# Run the HTTP server example
go run example/http/main.go
```

### Module Management
```bash
# Update dependencies
go mod tidy

# Download dependencies
go mod download
```

## Architecture Overview

### Core Components

1. **Event Loop System (`gevent.go`, `loop_unix.go`)**
   - Main event handling with epoll-based I/O multiplexing
   - Multi-loop support for leveraging multiple CPU cores
   - Load balancing strategies: Random, RoundRobin, LeastConnections

2. **Connection Management (`conn_unix.go`)**
   - Connection abstraction with context support
   - Non-blocking I/O operations
   - Connection lifecycle management

3. **Ring Buffer (`ringbuffer/ringbuffer.go`)**
   - High-performance circular buffer for data buffering
   - Auto-expanding buffer with virtual read/write pointers
   - Zero-copy operations where possible

4. **Timing Wheel (`internal/timingwheel/`)**
   - Hierarchical timing wheel for efficient timer management
   - Used for connection timeouts and scheduled operations
   - Delay queue implementation for timer expiration

5. **Platform-specific Code (`internal/internal_*.go`)**
   - OS-specific implementations for Linux, Darwin, BSD, OpenBSD
   - System call wrappers and platform optimizations

### Key Design Patterns

- **Event-driven Architecture**: All I/O operations are non-blocking and event-driven
- **Pool Pattern**: RingBuffer uses object pooling for memory efficiency
- **Multi-loop Pattern**: Multiple event loops can run concurrently for better CPU utilization
- **Load Balancing**: Connections are distributed across loops using configurable strategies

### Network Protocol Support

- **TCP**: Full-duplex TCP connections with keep-alive support
- **UDP**: Packet-based UDP communication
- **Unix Domain Sockets**: Local inter-process communication
- **Reuseport**: SO_REUSEPORT support for better load distribution

## Testing Strategy

The test suite includes:
- Connection lifecycle tests with multiple clients
- Load balancing verification across different strategies
- Unix domain socket testing
- Reuseport functionality testing
- Timer and tick mechanism testing
- Concurrent connection handling

## Important Implementation Details

### Connection Context
- Each connection can store custom context data via `SetContext()`
- Context is validated in connection lifecycle callbacks

### Buffer Management
- Use `RingBuffer` for efficient data buffering
- `PeekAll()` and `RetrieveAll()` for zero-copy operations
- Virtual read operations for lookahead without consumption

### Error Handling
- Connection errors are propagated through the `Closed` event callback
- Graceful shutdown via `Action.Shutdown` return value
- Connection-specific cleanup via `Action.Close`

### Performance Considerations
- Minimize memory allocations in hot paths
- Use buffer pools for frequently allocated objects
- Prefer stack allocation for small, short-lived objects
- Consider NUMA topology when using multiple loops

## Common Development Patterns

### Basic Server Setup
```go
var events gevent.Events
events.NumLoops = runtime.NumCPU()
events.LoadBalance = gevent.LeastConnections
events.Serving = func(srv gevent.Server) gevent.Action { /* ... */ }
events.Data = func(c *gevent.Conn, in *ringbuffer.RingBuffer) ([]byte, gevent.Action) { /* ... */ }
err := gevent.Serve(events, timeout, "tcp://localhost:8080")
```

### Connection State Management
- Use `Conn.SetContext()` to associate state with connections
- Access state in callbacks via `Conn.Context()`
- Clean up state in the `Closed` callback

### Data Processing
- Use `RingBuffer.PeekAll()` to examine data without consuming
- Call `RingBuffer.RetrieveAll()` to consume processed data
- Return response data from the `Data` callback