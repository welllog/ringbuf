package ringbuf

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkNewQueue(b *testing.B) {
	q := NewQueue(8)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Put(1)
			q.Get()
		}
	})
}

func BenchmarkNewRbQueue(b *testing.B) {
	q := NewRbQueue(8)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Put(1)
			q.Get()
		}
	})
}

func TestQueue(t *testing.T) {
	q := NewQueue(8)

	var (
		w    sync.WaitGroup
		get  uint32
		put  uint32
		num  = 10
		jump = 100000
	)

	w.Add(num)
	go func() {
		for i := 0; i < num; i++ {
			go func() {
				defer w.Done()

				var val int
				for {
					ok := q.Put(val)
					for !ok {
						time.Sleep(time.Millisecond)
						ok = q.Put(val)
					}
					atomic.AddUint32(&put, 1)
					val++
					if val == jump {
						break
					}
				}
			}()
		}
	}()

	w.Add(num)
	go func() {
		for i := 0; i < num; i++ {
			go func() {
				defer w.Done()

				var val int
				for {
					_, ok := q.Get()
					for !ok {
						time.Sleep(time.Millisecond)
						_, ok = q.Get()
					}
					atomic.AddUint32(&get, 1)
					val++
					if val == jump {
						break
					}
				}
			}()
		}
	}()

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		var oput, oget uint32
		for {
			<-ticker.C

			head := atomic.LoadUint32(&q.head)
			tail := atomic.LoadUint32(&q.tail)

			var full, empty bool
			var quantity uint32

			if head == tail {
				empty = true
			} else if (tail + 1) & q.capmod == head {
				full = true
			}
			if tail >= head {
				quantity = tail - head
			} else {
				quantity = q.capmod + (tail - head)
			}

			nput := atomic.LoadUint32(&put)
			nget := atomic.LoadUint32(&get)

			fmt.Printf("put: %d; get: %d; full: %t; empty: %t; size: %d; head: %d; tail: %d \n",
				nput, nget, full, empty, quantity, head, tail)

			if oput == nput && oget == nget {
				for i := range q.data {
					fmt.Printf("value: %v \n", q.data[i].value)
				}
			}
			oput = nput
			oget = nget
		}
	}()

	w.Wait()
}

func TestRepeatQueue(t *testing.T) {
	q := NewQueue(8)

	var (
		w    sync.WaitGroup
		get  uint32
		put  uint32
		num  = 4
		jump = 1000000
	)
	count := newSafeMap(jump)
	for i := 0; i < jump; i++ {
		count.data[i] = 0
	}

	w.Add(num)
	go func() {
		for i := 0; i < num; i++ {
			circle := i + 1
			go func() {
				defer w.Done()

				val := jump / num * (circle - 1)
				for {
					ok := q.Put(val)
					for !ok {
						time.Sleep(time.Millisecond)
						ok = q.Put(val)
					}
					atomic.AddUint32(&put, 1)
					val++
					if val >= jump / num * circle {
						break
					}
				}
			}()
		}
	}()

	w.Add(num)
	go func() {
		for i := 0; i < num; i++ {
			go func() {
				defer w.Done()

				var val int
				for {
					gval, ok := q.Get()
					for !ok {
						time.Sleep(time.Millisecond)
						gval, ok = q.Get()
					}
					count.Incr(gval)
					atomic.AddUint32(&get, 1)
					val++
					if val == jump / 4 {
						break
					}
				}
			}()
		}
	}()

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		//var oput, oget uint32
		for {
			<-ticker.C

			head := atomic.LoadUint32(&q.head)
			tail := atomic.LoadUint32(&q.tail)

			var full, empty bool
			var quantity uint32

			if head == tail {
				empty = true
			} else if (tail + 1) & q.capmod == head {
				full = true
			}
			if tail >= head {
				quantity = tail - head
			} else {
				quantity = q.capmod + (tail - head)
			}

			nput := atomic.LoadUint32(&put)
			nget := atomic.LoadUint32(&get)

			fmt.Printf("put: %d; get: %d; full: %t; empty: %t; size: %d; head: %d; tail: %d \n",
				nput, nget, full, empty, quantity, head, tail)

			//if oput == nput && oget == nget {
			//	for i := range q.data {
			//		fmt.Printf("status: %d; value: %v; tail: %d; head: %d \n", q.data[i].stat, q.data[i].value,
			//			q.data[i].tail, q.data[i].head)
			//	}
			//}
			//oput = nput
			//oget = nget
		}
	}()

	w.Wait()

	for i, v := range count.data {
		if v != 1 {
			fmt.Printf("key: %v; count: %d \n", i, v)
		}
	}
}

func TestSortQueue(t *testing.T) {
	q := NewQueue(8)

	var (
		w    sync.WaitGroup
		get  uint32
		put  uint32
		num  = 4
		jump = 40
	)

	w.Add(num)
	go func() {
		for i := 0; i < num; i++ {
			circle := i + 1
			go func() {
				defer w.Done()

				val := jump / num * (circle - 1)
				for {
					ok := q.Put(val)
					for !ok {
						time.Sleep(time.Millisecond)
						ok = q.Put(val)
					}
					atomic.AddUint32(&put, 1)
					val++
					if val >= jump / num * circle {
						break
					}
				}
			}()
		}
	}()

	w.Add(1)
	go func() {
		defer w.Done()

		var val int
		for {
			gval, ok := q.Get()
			for !ok {
				time.Sleep(time.Millisecond)
				gval, ok = q.Get()
			}
			fmt.Println(gval)
			atomic.AddUint32(&get, 1)
			val++
			if val == jump {
				break
			}
		}
	}()

	w.Wait()
}

type safeMap struct {
	data map[interface{}]int
	mu sync.Mutex
}

func newSafeMap(cap int) *safeMap {
	return &safeMap{
		data: make(map[interface{}]int, cap),
	}
}

func (sm *safeMap) Incr(key interface{}) {
	sm.mu.Lock()
	sm.data[key]++
	sm.mu.Unlock()
}

func TestRbQueue(t *testing.T) {
	q := NewRbQueue(8)

	var (
		w    sync.WaitGroup
		get  uint32
		put  uint32
		num  = 10
		jump = 100000
	)

	w.Add(num)
	go func() {
		for i := 0; i < num; i++ {
			go func() {
				defer w.Done()

				var val int
				for {
					ok := q.Put(val)
					for !ok {
						time.Sleep(time.Millisecond)
						ok = q.Put(val)
					}
					atomic.AddUint32(&put, 1)
					val++
					if val == jump {
						break
					}
				}
			}()
		}
	}()

	w.Add(num)
	go func() {
		for i := 0; i < num; i++ {
			go func() {
				defer w.Done()

				var val int
				for {
					_, ok := q.Get()
					for !ok {
						time.Sleep(time.Millisecond)
						_, ok = q.Get()
					}
					atomic.AddUint32(&get, 1)
					val++
					if val == jump {
						break
					}
				}
			}()
		}
	}()

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		var oput, oget uint32
		for {
			<-ticker.C

			head := atomic.LoadUint32(&q.head)
			tail := atomic.LoadUint32(&q.tail)

			var full, empty bool
			var quantity uint32

			if head == tail {
				empty = true
			} else if (tail + 1) & q.capmod == head {
				full = true
			}
			if tail >= head {
				quantity = tail - head
			} else {
				quantity = q.capmod + (tail - head)
			}

			nput := atomic.LoadUint32(&put)
			nget := atomic.LoadUint32(&get)

			fmt.Printf("put: %d; get: %d; full: %t; empty: %t; size: %d; head: %d; tail: %d \n",
				nput, nget, full, empty, quantity, head, tail)

			if oput == nput && oget == nget {
				for i := range q.data {
					fmt.Printf("value: %v; tail: %d; head: %d \n", q.data[i].value,
						q.data[i].tail, q.data[i].head)
				}
			}
			oput = nput
			oget = nget
		}
	}()

	w.Wait()
}

func TestRepeatRbQueue(t *testing.T) {
	q := NewRbQueue(8)

	var (
		w    sync.WaitGroup
		get  uint32
		put  uint32
		num  = 4
		jump = 100000
	)
	count := newSafeMap(jump)
	for i := 0; i < jump; i++ {
		count.data[i] = 0
	}

	w.Add(num)
	go func() {
		for i := 0; i < num; i++ {
			circle := i + 1
			go func() {
				defer w.Done()

				val := jump / num * (circle - 1)
				for {
					ok := q.Put(val)
					for !ok {
						time.Sleep(time.Millisecond)
						ok = q.Put(val)
					}
					atomic.AddUint32(&put, 1)
					val++
					if val >= jump / num * circle {
						break
					}
				}
			}()
		}
	}()

	w.Add(num)
	go func() {
		for i := 0; i < num; i++ {
			go func() {
				defer w.Done()

				var val int
				for {
					gval, ok := q.Get()
					for !ok {
						time.Sleep(time.Millisecond)
						gval, ok = q.Get()
					}
					count.Incr(gval)
					atomic.AddUint32(&get, 1)
					val++
					if val == jump / 4 {
						break
					}
				}
			}()
		}
	}()

	w.Wait()

	for i, v := range count.data {
		if v != 1 {
			fmt.Printf("key: %v; count: %d \n", i, v)
		}
	}
}

func TestSortRbQueue(t *testing.T) {
	q := NewQueue(8)

	var (
		w    sync.WaitGroup
		get  uint32
		put  uint32
		num  = 4
		jump = 40
	)

	w.Add(num)
	go func() {
		for i := 0; i < num; i++ {
			circle := i + 1
			go func() {
				defer w.Done()

				val := jump / num * (circle - 1)
				for {
					ok := q.Put(val)
					for !ok {
						time.Sleep(time.Millisecond)
						ok = q.Put(val)
					}
					atomic.AddUint32(&put, 1)
					val++
					if val >= jump / num * circle {
						break
					}
				}
			}()
		}
	}()

	w.Add(1)
	go func() {
		defer w.Done()

		var val int
		for {
			gval, ok := q.Get()
			for !ok {
				time.Sleep(time.Millisecond)
				gval, ok = q.Get()
			}
			fmt.Println(gval)
			atomic.AddUint32(&get, 1)
			val++
			if val == jump {
				break
			}
		}
	}()

	w.Wait()
}

