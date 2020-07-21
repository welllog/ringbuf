package ringbuf

import (
	"runtime"
	"sync/atomic"
)

type item struct {
	stat uint32
	value interface{}
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
	if tail >= head {
		return tail - head
	}
	return q.capmod + (tail - head)
}

func (q *Queue) IsFull() bool {
	var tail, head uint32
	//var quad uint64
	//quad = atomic.LoadUint64((*uint64)(unsafe.Pointer(&rb.head)))
	//head = (uint32)(quad & MaxUint32_64)
	//tail = (uint32)(quad >> 32)
	head = atomic.LoadUint32(&q.head)
	tail = atomic.LoadUint32(&q.tail)
	return ((tail + 1) & q.capmod) == head
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

	nt = (tail + 1) & q.capmod
	if nt == head { // tail + 1 = head  full
		next := &q.data[head]
		if atomic.LoadUint32(&next.stat) != 0 {
			return false
		}
		//return false
	}

	if !atomic.CompareAndSwapUint32(&q.tail, tail, nt) {
		return false
	}

	holder = &q.data[tail]
	for {
		if atomic.CompareAndSwapUint32(&holder.stat, 0, 2) {
			holder.value = val
			atomic.CompareAndSwapUint32(&holder.stat, 2, 1)
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

	holder = &q.data[head]
	if head == tail { // empty
		if atomic.LoadUint32(&holder.stat) != 1 {
			return nil, false
		}
		//return nil, false
	}

	nh = (head + 1) & q.capmod
	if !atomic.CompareAndSwapUint32(&q.head, head, nh) {
		return nil, false
	}

	//holder = &q.data[head]
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
