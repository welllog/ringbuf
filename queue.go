package ringbuf

import (
	"runtime"
	"sync/atomic"
	"time"
)

type item struct {
	value interface{}
	tail uint32
	head uint32
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
	if size & (size - 1) > 0 {
		size = roundupPowOfTwo(size)
	}
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
	return tail - head >= q.capmod
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
	head := atomic.LoadUint32(&q.head)
	tail := atomic.LoadUint32(&q.tail)

	if tail - head >= q.capmod {
		return false
	}

	nt := tail & q.capmod
	holder := &q.data[nt]

	if !atomic.CompareAndSwapUint32(&holder.tail, tail, tail + q.cap) {
		return false
	}

	for {
		if atomic.LoadUint32(&holder.head) == tail {
			holder.value = val
			atomic.AddUint32(&q.tail, 1)
			return true
		}
		runtime.Gosched()
	}
}

func (q *Queue) Get() (interface{}, bool) {
	tail := atomic.LoadUint32(&q.tail)
	head := atomic.LoadUint32(&q.head)

	if head == tail { // empty
		return nil, false
	} else if head > tail && (tail - head >= q.capmod) {
		return nil, false
	}

	nh := head & q.capmod
	holder := &q.data[nh]

	if !atomic.CompareAndSwapUint32(&holder.head, head, head + q.cap) {
		return nil, false
	}

	for {
		if atomic.LoadUint32(&holder.tail) == q.cap + head {
			val := holder.value
			holder.value = nil
			atomic.AddUint32(&q.head, 1)
			return val, true
		}
		runtime.Gosched()
	}
}

func (q *Queue) PutWait(val interface{}, delay ...time.Duration) bool {
	if q.Put(val) {
		return true
	}
	
	ticker := time.NewTicker(50 * time.Millisecond)
	
	var end time.Time
	start := time.Now()
	if len(delay) > 0 {
		end = start.Add(delay[0])
	} else {
		end = start.Add(500 * time.Millisecond)
	}
	
	for {
		now := <- ticker.C
		
		if q.Put(val) {
			ticker.Stop()
			return true
		}
		
		if now.After(end) {
			ticker.Stop()
			return false
		}
		
	}
	
}

func (q *Queue) GetWait(delay ...time.Duration) (interface{}, bool) {
	val,ok := q.Get()
	if ok {
		return val, true
	}
	
	ticker := time.NewTicker(50 * time.Millisecond)
	
	var end time.Time
	start := time.Now()
	if len(delay) > 0 {
		end = start.Add(delay[0])
	} else {
		end = start.Add(500 * time.Millisecond)
	}
	
	for {
		now := <- ticker.C
		
		val, ok = q.Get()
		if ok {
			ticker.Stop()
			return val, true
		}
		
		if now.After(end) {
			ticker.Stop()
			return nil, false
		}
		
	}
	
}

func roundupPowOfTwo(x uint32) uint32 {
	var pos int
	for i := x; i != 0; pos++ {
		i >>= 1
	}
	return 1 << pos
}
