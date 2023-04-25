package internal

import (
	"syscall"
)

// Poll 结构包含了 epoll 和 eventfd 文件描述符
type Poll struct {
	fd    int       // epoll fd
	wfd   int       // wake fd
	notes noteQueue // 通知队列
}

// sysEventfd2 是对系统调用 eventfd2 的封装
func sysEventfd2(initval, flags uintptr) (int, error) {
	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, initval, flags, 0)
	if errno != 0 {
		return -1, errno
	}
	return int(r0), nil
}

// OpenPoll 初始化并返回一个新的 Poll 结构
func OpenPoll() *Poll {
	l := new(Poll)
	p, err := syscall.EpollCreate1(0)
	if err != nil {
		panic(err)
	}
	l.fd = p
	wfd, err := sysEventfd2(0, 0)
	if err != nil {
		syscall.Close(p)
		panic(err)
	}
	l.wfd = wfd
	l.AddRead(l.wfd)
	return l
}

// Close 关闭 Poll 结构中的文件描述符
func (p *Poll) Close() error {
	if err := syscall.Close(p.wfd); err != nil {
		return err
	}
	return syscall.Close(p.fd)
}

// Trigger 添加一个通知到 notes 队列并唤醒 epoll
func (p *Poll) Trigger(note interface{}) error {
	p.notes.Add(note)
	_, err := syscall.Write(p.wfd, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	return err
}

// Wait 监听 epoll 事件，并在事件发生时调用迭代器
func (p *Poll) Wait(iter func(fd int, note interface{}) error) error {
	buf := make([]byte, 8)
	events := make([]syscall.EpollEvent, 64)
	for {
		n, err := syscall.EpollWait(p.fd, events[:cap(events)], -1)
		if err != nil && err != syscall.EINTR {
			return err
		}

		for i := 0; i < n; i++ {
			fd := int(events[i].Fd)
			if fd != p.wfd {
				if err := iter(fd, nil); err != nil {
					return err
				}
			} else {
				_, err := syscall.Read(p.wfd, buf)
				if err != nil && err != syscall.EAGAIN && err != syscall.EWOULDBLOCK {
					panic(err)
				}
			}
		}

		if err := p.notes.ForEach(func(note interface{}) error {
			return iter(0, note)
		}); err != nil {
			return err
		}
	}
}

// ctlEpoll 对 EpollCtl 进行封装，简化调用过程
func (p *Poll) ctlEpoll(op, fd int, events uint32) {
	if err := syscall.EpollCtl(p.fd, op, fd, &syscall.EpollEvent{Fd: int32(fd), Events: events}); err != nil {
		panic(err)
	}
}

// AddReadWrite 添加文件描述符为边缘触发的读写事件
func (p *Poll) AddReadWrite(fd int) {
	p.ctlEpoll(syscall.EPOLL_CTL_ADD, fd, syscall.EPOLLIN|syscall.EPOLLET|syscall.EPOLLOUT)
}

// AddRead 添加文件描述符为边缘触发的读事件
func (p *Poll) AddRead(fd int) {
	p.ctlEpoll(syscall.EPOLL_CTL_ADD, fd, syscall.EPOLLIN|syscall.EPOLLET|syscall.EPOLLPRI)
}

// ModRead 修改文件描述符为边缘触发的读事件
func (p *Poll) ModRead(fd int) {
	p.ctlEpoll(syscall.EPOLL_CTL_MOD, fd, syscall.EPOLLIN|syscall.EPOLLET|syscall.EPOLLPRI)
}

// ModReadWrite 修改文件描述符为边缘触发的读写事件
func (p *Poll) ModReadWrite(fd int) {
	p.ctlEpoll(syscall.EPOLL_CTL_MOD, fd, syscall.EPOLLIN|syscall.EPOLLOUT|syscall.EPOLLET)
}

// ModDetach 从 epoll 中移除文件描述符
func (p *Poll) ModDetach(fd int) {
	p.ctlEpoll(syscall.EPOLL_CTL_DEL, fd, syscall.EPOLLIN|syscall.EPOLLOUT)
}
