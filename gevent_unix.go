//go:build darwin || netbsd || freebsd || openbsd || dragonfly || linux
// +build darwin netbsd freebsd openbsd dragonfly linux

package gevent

import (
	"errors"
	"github.com/go-gevent/gevent/internal/timingwheel"
	"github.com/panjf2000/ants/v2"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/go-gevent/gevent/internal"
	reuseport "github.com/libp2p/go-reuseport"
)

var errClosing = errors.New("closing")

type server struct {
	events      Events
	loops       []*loop
	lns         []*listener
	wg          sync.WaitGroup
	cond        *sync.Cond
	balance     LoadBalance
	accepted    uintptr
	tch         time.Duration
	waitTimeout time.Duration
}

// waitForShutdown waits for a signal to shutdown
func (s *server) waitForShutdown() {
	s.cond.L.Lock()
	s.cond.Wait()
	s.cond.L.Unlock()
}

// signalShutdown signals a shutdown an begins server closing
func (s *server) signalShutdown() {
	s.cond.L.Lock()
	s.cond.Signal()
	s.cond.L.Unlock()
}

func serve(events Events, waitTimeout time.Duration, listeners []*listener) error {
	// figure out the correct number of loops/goroutines to use.
	numLoops := events.NumLoops
	if numLoops <= 0 {
		if numLoops == 0 {
			numLoops = 1
		} else {
			numLoops = runtime.NumCPU()
		}
	}

	s := &server{
		events:      events,
		lns:         listeners,
		cond:        sync.NewCond(&sync.Mutex{}),
		balance:     events.LoadBalance,
		tch:         time.Duration(0),
		waitTimeout: waitTimeout,
	}

	if s.events.Serving != nil {
		var svr Server
		svr.NumLoops = numLoops
		svr.Addrs = make([]net.Addr, len(listeners))
		for i, ln := range listeners {
			svr.Addrs[i] = ln.lnaddr
		}
		action := s.events.Serving(svr)
		switch action {
		case None:
		case Shutdown:
			return nil
		}
	}

	defer func() {
		ants.Release()
		s.waitForShutdown()

		for _, l := range s.loops {
			_ = l.poll.Trigger(errClosing)
		}

		s.wg.Wait()

		for _, l := range s.loops {
			for _, c := range l.fdconns {
				_ = l.loopCloseConn(s, c, nil)
			}
			_ = l.poll.Close()
			l.tw.Stop()
		}
	}()

	s.loops = make([]*loop, numLoops)

	for i := 0; i < numLoops; i++ {
		l := &loop{
			idx:     i,
			poll:    internal.OpenPoll(),
			packet:  make([]byte, 0xFFFF),
			fdconns: make(map[int]*Conn),
			tw:      timingwheel.NewTimingWheel(time.Second, 60),
			bufferPool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 64*1024) // 64KB 用于存储读取的数据包
				},
			},
		}
		l.tw.Start()
		for _, ln := range listeners {
			l.poll.AddRead(ln.fd)
		}
		s.loops[i] = l
	}
	for _, l := range s.loops {
		s.wg.Add(1)
		err := ants.Submit(func() {
			l.loopRun(s)
			s.wg.Done()
			s.signalShutdown()
		})
		if err != nil {
			log.Printf("submit task error: %v", err)
		}
	}
	return nil
}

func (ln *listener) close() {
	if ln.fd != 0 {
		_ = syscall.Close(ln.fd)
	}
	if ln.f != nil {
		_ = ln.f.Close()
	}
	if ln.ln != nil {
		_ = ln.ln.Close()
	}
	if ln.pconn != nil {
		_ = ln.pconn.Close()
	}
	if ln.network == "unix" {
		_ = os.RemoveAll(ln.addr)
	}
}

// system takes the net listener and detaches it from it's parent
// event loop, grabs the file descriptor, and makes it non-blocking.
func (ln *listener) system() error {
	var err error
	switch netln := ln.ln.(type) {
	case nil:
		switch pconn := ln.pconn.(type) {
		case *net.UDPConn:
			ln.f, err = pconn.File()
		}
	case *net.TCPListener:
		ln.f, err = netln.File()
	case *net.UnixListener:
		ln.f, err = netln.File()
	}
	if err != nil {
		ln.close()
		return err
	}
	ln.fd = int(ln.f.Fd())
	return syscall.SetNonblock(ln.fd, true)
}

func reuseportListenPacket(proto, addr string) (l net.PacketConn, err error) {
	return reuseport.ListenPacket(proto, addr)
}

func reuseportListen(proto, addr string) (l net.Listener, err error) {
	return reuseport.Listen(proto, addr)
}
