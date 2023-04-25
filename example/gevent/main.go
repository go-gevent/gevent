package main

import (
	"flag"
	"fmt"
	"github.com/go-gevent/gevent"
	"github.com/go-gevent/gevent/ringbuffer"
	"log"
	"strings"
	"time"
)

func main() {
	var port int
	var loops int
	var udp bool
	var trace bool
	var reuseport bool

	flag.IntVar(&port, "port", 5007, "server port")
	flag.BoolVar(&udp, "udp", false, "listen on udp")
	flag.BoolVar(&reuseport, "reuseport", false, "reuseport (SO_REUSEPORT)")
	flag.BoolVar(&trace, "trace", false, "print packets to console")
	flag.IntVar(&loops, "loops", 1, "num loops")
	flag.Parse()

	var events gevent.Events
	events.NumLoops = loops
	events.Serving = func(srv gevent.Server) (action gevent.Action) {
		log.Printf("echo server started on port %d (loops: %d)", port, srv.NumLoops)
		if reuseport {
			log.Printf("reuseport")
		}
		return
	}
	events.Data = func(c *gevent.Conn, in *ringbuffer.RingBuffer) (out []byte, action gevent.Action) {
		first, end := in.PeekAll()
		if trace {
			log.Printf("%s", strings.TrimSpace(string(first)+string(end)))
		}
		out = first
		if len(end) > 0 {
			out = append(out, end...)
		}
		in.RetrieveAll()
		return
	}
	scheme := "tcp"
	if udp {
		scheme = "udp"
	}
	log.Fatal(gevent.Serve(events, time.Second*10, fmt.Sprintf("%s://:%d?reuseport=%t", scheme, port, reuseport)))
}
