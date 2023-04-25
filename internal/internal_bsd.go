//go:build darwin || netbsd || freebsd || openbsd || dragonfly
// +build darwin netbsd freebsd openbsd dragonfly

package internal

import (
	"syscall"
)

type Poll struct {
	fd      int
	changes []syscall.Kevent_t
	notes   noteQueue
}

func OpenPoll() *Poll {
	l := new(Poll)
	p, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}
	l.fd = p
	_, err = syscall.Kevent(l.fd, []syscall.Kevent_t{{
		Ident:  0,
		Filter: syscall.EVFILT_USER,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
	}}, nil, nil)
	if err != nil {
		panic(err)
	}

	return l
}

func (p *Poll) Close() error {
	return syscall.Close(p.fd)
}

func (p *Poll) Trigger(note interface{}) error {
	p.notes.Add(note)
	_, err := syscall.Kevent(p.fd, []syscall.Kevent_t{{
		Ident:  0,
		Filter: syscall.EVFILT_USER,
		Fflags: syscall.NOTE_TRIGGER,
	}}, nil, nil)
	return err
}

func (p *Poll) Wait(iter func(fd int, note interface{}) error) error {
	events := make([]syscall.Kevent_t, 128)
	for {
		n, err := syscall.Kevent(p.fd, p.changes, events, nil)
		if err != nil && err != syscall.EINTR {
			return err
		}
		p.changes = p.changes[:0]
		if err := p.notes.ForEach(func(note interface{}) error {
			return iter(0, note)
		}); err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if fd := int(events[i].Ident); fd != 0 {
				if err := iter(fd, nil); err != nil {
					return err
				}
			}
		}
	}
}

func (p *Poll) addChange(filter, flags int16, fd int) {
	p.changes = append(p.changes, syscall.Kevent_t{
		Ident: uint64(fd), Flags: uint16(flags), Filter: filter,
	})
}

func (p *Poll) AddRead(fd int) {
	p.addChange(syscall.EVFILT_READ, syscall.EV_ADD, fd)
}

func (p *Poll) AddReadWrite(fd int) {
	p.addChange(syscall.EVFILT_READ, syscall.EV_ADD, fd)
	p.addChange(syscall.EVFILT_WRITE, syscall.EV_ADD, fd)
}

func (p *Poll) ModRead(fd int) {
	p.addChange(syscall.EVFILT_WRITE, syscall.EV_DELETE, fd)
}

func (p *Poll) ModReadWrite(fd int) {
	p.addChange(syscall.EVFILT_WRITE, syscall.EV_ADD, fd)
}

func (p *Poll) ModDetach(fd int) {
	p.addChange(syscall.EVFILT_READ, syscall.EV_DELETE, fd)
	p.addChange(syscall.EVFILT_WRITE, syscall.EV_DELETE, fd)
}
