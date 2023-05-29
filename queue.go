package ringbuf

import (
	"sync/atomic"
	"time"
)

type item struct {
	value    interface{}
	position uint32
}

type Queue struct {
	cap  uint32
	mask uint32
	_    [56]byte
	head uint32
	_    [60]byte
	tail uint32
	_    [60]byte
	data []item
}

func NewQueue(size uint32) *Queue {
	if size&(size-1) > 0 {
		size = roundupPowOfTwo(size)
	}
	q := &Queue{
		cap:  size,
		mask: size - 1,
		data: make([]item, size),
	}
	for i := range q.data {
		q.data[i].position = uint32(i)
	}
	return q
}

func (q *Queue) Cap() uint32 {
	return q.cap
}

func (q *Queue) Quantity() uint32 {
	return atomic.LoadUint32(&q.tail) - atomic.LoadUint32(&q.head)
}

func (q *Queue) IsFull() bool {
	return atomic.LoadUint32(&q.tail)-atomic.LoadUint32(&q.head) == q.cap
}

func (q *Queue) IsEmpty() (b bool) {
	return atomic.LoadUint32(&q.head) == atomic.LoadUint32(&q.tail)
}

func (q *Queue) Put(val interface{}) bool {
	pos := atomic.LoadUint32(&q.tail)

	holder := &q.data[pos&q.mask]
	seq := atomic.LoadUint32(&holder.position)

	if seq != pos {
		return false
	}

	if !atomic.CompareAndSwapUint32(&q.tail, pos, pos+1) {
		return false
	}

	holder.value = val
	atomic.AddUint32(&holder.position, 1)
	return true
}

func (q *Queue) Get() (interface{}, bool) {
	pos := atomic.LoadUint32(&q.head)

	holder := &q.data[pos&q.mask]
	seq := atomic.LoadUint32(&holder.position)
	if seq != pos+1 {
		return nil, false
	}

	if !atomic.CompareAndSwapUint32(&q.head, pos, pos+1) {
		return nil, false
	}

	val := holder.value
	holder.value = nil
	atomic.AddUint32(&holder.position, q.mask)
	return val, true
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
		now := <-ticker.C

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
	val, ok := q.Get()
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
		now := <-ticker.C

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
