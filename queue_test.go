package ringbuf

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRbQueueQuantity(t *testing.T) {
	q := NewQueue(8)

	// Test empty queue
	if q.Quantity() != 0 {
		t.Errorf("Expected quantity to be 0, but got %d", q.Quantity())
	}

	// Test single element
	q.Put(1)
	if q.Quantity() != 1 {
		t.Errorf("Expected quantity to be 1, but got %d", q.Quantity())
	}

	// Test multiple elements
	q.Put(2)
	q.Put(3)
	if q.Quantity() != 3 {
		t.Errorf("Expected quantity to be 3, but got %d", q.Quantity())
	}

	// Test concurrent access
	var wg sync.WaitGroup
	var expected uint32 = 1000
	for i := uint32(0); i < expected; i++ {
		wg.Add(1)
		go func() {
			q.Put(i)
			wg.Done()
		}()
	}
	wg.Wait()
	if q.Quantity() != 8 {
		t.Errorf("Expected quantity to be %d, but got %d", 8, q.Quantity())
	}

	// Test concurrent access with removal
	var removed uint32
	for i := uint32(0); i < expected; i++ {
		wg.Add(1)
		go func() {
			_, ok := q.Get()
			if ok {
				atomic.AddUint32(&removed, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if q.Quantity() != 0 {
		t.Errorf("Expected quantity to be 0, but got %d", q.Quantity())
	}
}

func TestRbQueueIsFull(t *testing.T) {
	// Test empty queue
	q := NewQueue(8)
	if q.IsFull() {
		t.Errorf("Expected IsFull to return false, but it returned true")
	}

	// Test partially full queue
	q.Put(1)
	q.Put(2)
	if q.IsFull() {
		t.Errorf("Expected IsFull to return false, but it returned true")
	}

	// Test full queue
	for i := 3; i <= 8; i++ {
		q.Put(i)
	}
	if !q.IsFull() {
		t.Errorf("Expected IsFull to return true, but it returned false")
	}

	// Test concurrent access
	var wg sync.WaitGroup
	var expected uint32 = 1000
	for i := uint32(0); i < expected; i++ {
		wg.Add(1)
		go func() {
			q.Put(0)
			wg.Done()
		}()
	}
	wg.Wait()
	if !q.IsFull() {
		t.Errorf("Expected IsFull to return true, but it returned false")
	}

	// Test concurrent access with removal
	for i := uint32(0); i < expected; i++ {
		q.Get()
	}
	if q.IsFull() {
		t.Errorf("Expected IsFull to return false, but it returned true")
	}

	// Test concurrent access with removal and addition
	for i := uint32(0); i < expected; i++ {
		wg.Add(1)
		go func() {
			q.Put(0)
			q.Get()
			wg.Done()
		}()
	}
	wg.Wait()
	if q.IsFull() {
		t.Errorf("Expected IsFull to return false, but it returned true")
	}
}

func TestRbQueueIsEmpty(t *testing.T) {
	q := NewQueue(8)

	// Test empty queue
	if !q.IsEmpty() {
		t.Errorf("Expected queue to be empty, but it is not")
	}

	// Test single element
	q.Put(1)
	if q.IsEmpty() {
		t.Errorf("Expected queue to not be empty, but it is")
	}

	// Test multiple elements
	q.Put(2)
	q.Put(3)
	if q.IsEmpty() {
		t.Errorf("Expected queue to not be empty, but it is")
	}

	// Test concurrent access
	var wg sync.WaitGroup
	var expected uint32 = 1000
	for i := uint32(0); i < expected; i++ {
		wg.Add(1)
		go func(i uint32) {
			q.Put(i)
			wg.Done()
		}(i)
	}
	wg.Wait()
	if q.IsEmpty() {
		t.Errorf("Expected queue to not be empty, but it is")
	}

	// Test concurrent access with removal
	var removed uint32
	for i := uint32(0); i < expected; i++ {
		wg.Add(1)
		go func() {
			_, ok := q.Get()
			if ok {
				atomic.AddUint32(&removed, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if !q.IsEmpty() {
		t.Errorf("Expected queue to be empty, but it is not")
	}
}

func TestRbQueuePut(t *testing.T) {
	q := NewQueue(8)

	// Test single element
	if !q.Put(1) {
		t.Errorf("Expected Put to return true, but it returned false")
	}
	if q.Quantity() != 1 {
		t.Errorf("Expected quantity to be 1, but got %d", q.Quantity())
	}

	// Test multiple elements
	if !q.Put(2) {
		t.Errorf("Expected Put to return true, but it returned false")
	}
	if !q.Put(3) {
		t.Errorf("Expected Put to return true, but it returned false")
	}
	if q.Quantity() != 3 {
		t.Errorf("Expected quantity to be 3, but got %d", q.Quantity())
	}

	// Test concurrent access
	var wg sync.WaitGroup
	var expected uint32 = 1000
	for i := uint32(0); i < expected; i++ {
		wg.Add(1)
		go func(i uint32) {
			q.Put(i)
			wg.Done()
		}(i)
	}
	wg.Wait()
	if q.Quantity() != 8 {
		t.Errorf("Expected quantity to be %d, but got %d", 8, q.Quantity())
	}

	// Test concurrent access with contention
	q = NewQueue(1)
	q.Put(1)
	wg.Add(2)
	go func() {
		if !q.Put(2) {
			t.Errorf("Expected Put to return true, but it returned false")
		}
		wg.Done()
	}()
	go func() {
		if !q.Put(3) {
			t.Errorf("Expected Put to return true, but it returned false")
		}
		wg.Done()
	}()
	wg.Wait()
	if q.Quantity() != 3 {
		t.Errorf("Expected quantity to be %d, but got %d", 3, q.Quantity())
	}
}

func TestRbQueueGet(t *testing.T) {
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
						runtime.Gosched()
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
						runtime.Gosched()
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
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			<-ticker.C

			head := atomic.LoadUint32(&q.head)
			tail := atomic.LoadUint32(&q.tail)

			var full, empty bool
			var quantity uint32

			if head == tail {
				empty = true
			} else if tail-head == q.cap {
				full = true
			} else {
				quantity = tail - head
			}

			nput := atomic.LoadUint32(&put)
			nget := atomic.LoadUint32(&get)

			fmt.Printf("put: %d; get: %d; full: %t; empty: %t; size: %d; head: %d; tail: %d \n",
				nput, nget, full, empty, quantity, head, tail)
		}
	}()

	w.Wait()

	fmt.Printf("put: %d; get: %d \n", put, get)
}

func BenchmarkQueue(b *testing.B) {
	b.Run("Queue", func(b *testing.B) {
		q := NewQueue(8)
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for {
					if q.Put(1) {
						break
					} else {
						runtime.Gosched()
					}
				}

				for {
					if _, ok := q.Get(); ok {
						break
					} else {
						runtime.Gosched()
					}
				}

			}
		})
	})

	b.Run("Channel", func(b *testing.B) {
		q := make(chan int, 8)
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				q <- 1

				<-q
			}
		})
	})
}
