package ringbuf

import (
	"runtime"
	"sync/atomic"
)

type item struct {
	value interface{}
	_ [56]byte
	tail uint32
	_ [60]byte
	head uint32
	_ [56]byte
}

type Queue struct {
	cap uint32
	capmod uint32
	_ [56]byte
	head uint32
	_ [60]byte
	tail uint32 // tail cannot catch up with head
	_ [60]byte
	data []item
}

func NewQueue(size uint32) *Queue {
	size = roundUpToPower2(size)
	q := &Queue{
		cap:    size,
		capmod: size - 1,
		data:   make([]item, size),
	}
	for i := range q.data {
		q.data[i].tail = uint32(i)
		q.data[i].head = uint32(i)
	}
	return q
}

func (q *Queue) Cap() uint32 {
	return q.cap
}

func (q *Queue) Quantity() uint32 {
	var tail, head uint32
	//var quad uint64
	//quad = atomic.LoadUint64((*uint64)(unsafe.Pointer(&rb.head)))
	//head = (uint32)(quad & MaxUint32_64)
	//tail = (uint32)(quad >> 32)
	head = atomic.LoadUint32(&q.head)
	tail = atomic.LoadUint32(&q.tail)
	return tail - head
}

func (q *Queue) IsFull() bool {
	var tail, head uint32
	//var quad uint64
	//quad = atomic.LoadUint64((*uint64)(unsafe.Pointer(&rb.head)))
	//head = (uint32)(quad & MaxUint32_64)
	//tail = (uint32)(quad >> 32)
	head = atomic.LoadUint32(&q.head)
	tail = atomic.LoadUint32(&q.tail)
	return tail + 1 == head
}

func (q *Queue) IsEmpty() (b bool) {
	var tail, head uint32
	//var quad uint64
	//quad = atomic.LoadUint64((*uint64)(unsafe.Pointer(&rb.head)))
	//head = (uint32)(quad & MaxUint32_64)
	//tail = (uint32)(quad >> 32)
	head = atomic.LoadUint32(&q.head)
	tail = atomic.LoadUint32(&q.tail)
	return head == tail
}

func (q *Queue) Put(val interface{}) bool {
	var tail, head, nt uint32
	var holder *item
	head = atomic.LoadUint32(&q.head)
	tail = atomic.LoadUint32(&q.tail)

	if tail - head >= q.capmod {
		return false
	}

	nt = tail & q.capmod
	holder = &q.data[nt]

	if !atomic.CompareAndSwapUint32(&holder.tail, tail, tail + q.cap) {
		return false
	}

	for {
		hd := atomic.LoadUint32(&holder.head)
		if hd == tail {
			holder.value = val
			atomic.AddUint32(&q.tail, 1)
			return true
		}
		runtime.Gosched()
	}
}

func (q *Queue) Get() (interface{}, bool) {
	var tail, head, nh uint32
	var holder *item
	tail = atomic.LoadUint32(&q.tail)
	head = atomic.LoadUint32(&q.head)

	if head == tail { // empty
		return nil, false
	} else if head > tail && (tail - head >= q.capmod) {
		return nil, false
	}

	nh = head & q.capmod
	holder = &q.data[nh]

	if !atomic.CompareAndSwapUint32(&holder.head, head, head + q.cap) {
		return nil, false
	}

	for {
		ht := atomic.LoadUint32(&holder.tail)
		if ht - head == q.cap {
			val := holder.value
			holder.value = nil
			atomic.AddUint32(&q.head, 1)
			return val, true
		}
		runtime.Gosched()
	}
}

func roundUpToPower2(v uint32) uint32 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}
