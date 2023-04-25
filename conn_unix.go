//go:build darwin || netbsd || freebsd || openbsd || dragonfly || linux
// +build darwin netbsd freebsd openbsd dragonfly linux

package gevent

import (
	"github.com/go-gevent/gevent/internal/utils"
	"github.com/go-gevent/gevent/ringbuffer"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Conn tcp连接
type Conn struct {
	fd          int                    // file descriptor
	lnidx       int                    // listener index in the server lns list
	outBuffer   *ringbuffer.RingBuffer // write buffer
	inBuffer    *ringbuffer.RingBuffer
	sa          syscall.Sockaddr // remote socket address
	opened      bool             // connection opened event fired
	toClose     utils.AtomicBool
	action      Action      // next user action
	ctx         interface{} // user-defined context
	addrIndex   int         // index of listening address
	localAddr   net.Addr    // local addre
	remoteAddr  net.Addr    // remote addr
	activeTime  int64       // Last received message time
	loop        *loop       // connected loop
	pendingFunc []func()
	mu          sync.RWMutex
}

// Send 发送
// 供非 loop 协程调用
func (c *Conn) Send(buf []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.pendingFunc) == 0 {
		c.pendingFunc = append(c.pendingFunc, func() {
			c.send(buf)
		})
		c.Wake()
	} else {
		c.pendingFunc = append(c.pendingFunc, func() {
			c.send(buf)
		})
	}
}

func (c *Conn) send(buf []byte) {
	if c.outBuffer.Length() > 0 {
		_, _ = c.outBuffer.Write(buf)
		return
	}

	n, err := syscall.Write(c.fd, buf)
	if err != nil {
		_, _ = c.outBuffer.Write(buf)
		return
	}

	if n < len(buf) {
		_, _ = c.outBuffer.Write(buf[n:])
	}
}

// Context 获取 Context
func (c *Conn) Context() interface{} {
	c.mu.RLock()
	ctx := c.ctx
	c.mu.RUnlock()
	return ctx
}

// SetContext 设置 Context
func (c *Conn) SetContext(ctx interface{}) { c.ctx = ctx }

// AddrIndex AddrIndex
func (c *Conn) AddrIndex() int { return c.addrIndex }

// LocalAddr LocalAddr
func (c *Conn) LocalAddr() net.Addr { return c.localAddr }

// RemoteAddr RemoteAddr
func (c *Conn) RemoteAddr() net.Addr      { return c.remoteAddr }
func (c *Conn) setActiveTime(t time.Time) { atomic.SwapInt64(&c.activeTime, t.Unix()) }
func (c *Conn) getActiveTime() time.Time  { return time.Unix(atomic.LoadInt64(&c.activeTime), 0) }

// Wake 唤醒 loop
func (c *Conn) Wake() {
	if c.loop != nil {
		_ = c.loop.poll.Trigger(c)
	}
}
