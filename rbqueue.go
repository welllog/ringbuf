package ringbuf

import (
	"runtime"
	"sync/atomic"
)

type rbItem struct {
	value interface{}
	_ [56]byte
	tail uint32
	_ [60]byte
	head uint32
	_ [56]byte
}

type RbQueue struct {
	cap uint32
	capmod uint32
	_ [56]byte
	head uint32
	_ [60]byte
	tail uint32
	_ [60]byte
	data []rbItem
}

func NewRbQueue(size uint32) *RbQueue {
	size = roundupPowOfTwo(size)
	q := &RbQueue{
		cap:    size,
		capmod: size - 1,
		data:   make([]rbItem, size),
	}
	for i := range q.data {
		q.data[i].tail = uint32(i)
		q.data[i].head = uint32(i)
	}
	return q
}

func (q *RbQueue) Put(val interface{}) bool {
	var tail, head, index uint32
	var holder *rbItem
	head = atomic.LoadUint32(&q.head)
	tail = atomic.LoadUint32(&q.tail)

	if tail - head >= q.capmod {
		return false
	}

	if !atomic.CompareAndSwapUint32(&q.tail, tail, tail + 1) {
		return false
	}

	index = tail & q.capmod
	holder = &q.data[index]

	for {
		hd := atomic.LoadUint32(&holder.head)
		ht := atomic.LoadUint32(&holder.tail)

		if ht == tail && hd == ht {
			holder.value = val
			atomic.AddUint32(&holder.tail, q.cap)
			return true
		}
		runtime.Gosched()
	}
}

func (q *RbQueue) Get() (interface{}, bool) {
	var tail, head, index uint32
	var holder *rbItem
	tail = atomic.LoadUint32(&q.tail)
	head = atomic.LoadUint32(&q.head)

	if head == tail { // empty
		return nil, false
	} else if head > tail && (tail - head >= q.capmod) {
		return nil, false
	}

	if !atomic.CompareAndSwapUint32(&q.head, head, head + 1) {
		return nil, false
	}

	index = head & q.capmod
	holder = &q.data[index]

	for {
		hd := atomic.LoadUint32(&holder.head)
		ht := atomic.LoadUint32(&holder.tail)

		if hd == head && (ht - hd) == q.cap {
			val := holder.value
			holder.value = nil
			atomic.AddUint32(&holder.head, q.cap)
			return val, true
		}
		runtime.Gosched()
	}
}
