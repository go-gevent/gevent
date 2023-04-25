package timingwheel

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type Timer struct {
	expiration int64
	task       func()

	b unsafe.Pointer // type: *bucket

	element interface{}
}

func (t *Timer) getBucket() *bucket {
	return (*bucket)(atomic.LoadPointer(&t.b))
}

func (t *Timer) setBucket(b *bucket) {
	atomic.StorePointer(&t.b, unsafe.Pointer(b))
}

func (t *Timer) Stop() bool {
	stopped := false
	for b := t.getBucket(); b != nil; b = t.getBucket() {
		stopped = b.Remove(t)
	}
	return stopped
}

type bucket struct {
	timers sync.Map

	expiration int64
}

func newBucket() *bucket {
	return &bucket{
		expiration: -1,
	}
}

func (b *bucket) Expiration() int64 {
	return atomic.LoadInt64(&b.expiration)
}

func (b *bucket) SetExpiration(expiration int64) bool {
	return atomic.SwapInt64(&b.expiration, expiration) != expiration
}

func (b *bucket) Add(t *Timer) {
	e := t
	t.setBucket(b)
	t.element = e

	b.timers.Store(e, t)
}

func (b *bucket) remove(t *Timer) bool {
	if t.getBucket() != b {
		return false
	}
	b.timers.Delete(t.element)
	t.setBucket(nil)
	t.element = nil
	return true
}

func (b *bucket) Remove(t *Timer) (ok bool) {
	ok = b.remove(t)
	return
}

func (b *bucket) Flush(reinsert func(*Timer)) {
	b.timers.Range(func(k, v interface{}) bool {
		t := v.(*Timer)
		b.remove(t)
		reinsert(t)
		return true
	})

	b.SetExpiration(-1)
}
