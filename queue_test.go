package ringbuf

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	q := NewQueue(8)

	var (
		w    sync.WaitGroup
		get  uint32
		put  uint32
		num  = 4
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
					fmt.Printf("status: %d; value: %v \n", q.data[i].stat, q.data[i].value)
				}
			}
			oput = nput
			oget = nget
		}
	}()

	w.Wait()
}

func TestRbQueue(t *testing.T) {
	q := NewRbQueue(8)

	var (
		w    sync.WaitGroup
		get  uint32
		put  uint32
		num  = 4
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

			head := atomic.LoadUint64(&q.head)
			tail := atomic.LoadUint64(&q.tail)

			var full, empty bool
			var quantity uint64

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
					fmt.Printf("status: %d; value: %v; tail: %d; head: %d \n", q.data[i].stat, q.data[i].value,
						q.data[i].tail, q.data[i].head)
				}
			}
			oput = nput
			oget = nget
		}
	}()

	w.Wait()
}