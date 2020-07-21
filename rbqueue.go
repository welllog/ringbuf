package ringbuf

import (
	"runtime"
	"sync/atomic"
)

type rbItem struct {
	head uint64
	tail uint64
	stat uint32
	value interface{}
}

type RbQueue struct {
	cap uint64
	capmod uint64
	_ [48]byte
	head uint64
	_ [56]byte
	tail uint64
	_ [56]byte
	data []rbItem
}

func NewRbQueue(size uint32) *RbQueue {
	size = roundUpToPower2(size)
	q := &RbQueue{
		cap:    uint64(size),
		capmod: uint64(size - 1),
		data:   make([]rbItem, size),
	}
	return q
}

func (q *RbQueue) Put(val interface{}) bool {
	var tail, head, nt uint64
	var holder *rbItem
	head = atomic.LoadUint64(&q.head)
	tail = atomic.LoadUint64(&q.tail)

	if tail - head > q.capmod { // full
		return false
	}

	nt = tail & q.capmod
	holder = &q.data[nt]

	circle := tail / q.cap
	if !atomic.CompareAndSwapUint64(&holder.tail, circle, circle + 1) {
		return false
	}

	atomic.AddUint64(&q.tail, 1)

	for {
		if atomic.CompareAndSwapUint32(&holder.stat, 0, 2) {
			holder.value = val
			atomic.CompareAndSwapUint32(&holder.stat, 2, 1)
			return true
		}
		runtime.Gosched()
	}
}

func (q *RbQueue) Get() (interface{}, bool) {
	var tail, head, nh uint64
	var holder *rbItem
	tail = atomic.LoadUint64(&q.tail)
	head = atomic.LoadUint64(&q.head)

	if head >= tail { // empty
		return nil, false
	}

	nh = head & q.capmod
	holder = &q.data[nh]

	circle := head / q.cap
	if !atomic.CompareAndSwapUint64(&holder.head, circle, circle + 1) {
		return nil, false
	}

	atomic.AddUint64(&q.head, 1)

	for {
		if atomic.CompareAndSwapUint32(&holder.stat, 1, 3) {
			val := holder.value
			holder.value = nil
			atomic.CompareAndSwapUint32(&holder.stat, 3, 0)
			return val, true
		}
		runtime.Gosched()
	}
}
