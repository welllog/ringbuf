package ringbuf

import (
	"fmt"
	"runtime"
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
			q.PutWait(1, time.Second)
			q.GetWait(time.Second)
		}
	})
}

func BenchmarkNewRbQueue(b *testing.B) {
	q := NewRbQueue(8)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.PutWait(1, time.Second)
			q.GetWait(time.Second)
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
			} else if (tail+1)&q.capmod == head {
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
					if val >= jump/num*circle {
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
					if val == jump/4 {
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
			} else if (tail+1)&q.capmod == head {
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
					if val >= jump/num*circle {
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

func TestQueuePutGet(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cnt := 10000
	sum := 0
	start := time.Now()
	var putD, getD time.Duration
	for i := 0; i <= runtime.NumCPU()*4; i++ {
		sum += i * cnt
		put, get := testQueuePutGet(t, i, cnt)
		putD += put
		getD += get
	}
	end := time.Now()
	use := end.Sub(start)
	op := use / time.Duration(sum)
	t.Logf("Grp: %d, Times: %d, use: %v, %v/op", runtime.NumCPU()*4, sum, use, op)
	t.Logf("Put: %d, use: %v, %v/op", sum, putD, putD/time.Duration(sum))
	t.Logf("Get: %d, use: %v, %v/op", sum, getD, getD/time.Duration(sum))
}

func TestQueueGeneral(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var miss, Sum int
	var Use time.Duration
	for i := 1; i <= runtime.NumCPU()*4; i++ {
		cnt := 10000 * 100
		if i > 9 {
			cnt = 10000 * 10
		}
		sum := i * cnt
		start := time.Now()
		miss = testQueueGeneral(t, i, cnt)
		end := time.Now()
		use := end.Sub(start)
		op := use / time.Duration(sum)
		fmt.Printf("%v, Grp: %3d, Times: %10d, miss:%6v, use: %12v, %8v/op\n",
			runtime.Version(), i, sum, miss, use, op)
		Use += use
		Sum += sum
	}
	op := Use / time.Duration(Sum)
	fmt.Printf("%v, Grp: %3v, Times: %10d, miss:%6v, use: %12v, %8v/op\n",
		runtime.Version(), "Sum", Sum, 0, Use, op)
}

func TestQueuePutGoGet(t *testing.T) {
	var Sum, miss int
	var Use time.Duration
	for i := 1; i <= runtime.NumCPU()*4; i++ {
		//	for i := 2; i <= 2; i++ {
		cnt := 10000 * 100
		if i > 9 {
			cnt = 10000 * 10
		}
		sum := i * cnt
		start := time.Now()
		miss = testQueuePutGoGet(t, i, cnt)

		end := time.Now()
		use := end.Sub(start)
		op := use / time.Duration(sum)
		fmt.Printf("%v, Grp: %3d, Times: %10d, miss:%6v, use: %12v, %8v/op\n",
			runtime.Version(), i, sum, miss, use, op)
		Use += use
		Sum += sum
	}
	op := Use / time.Duration(Sum)
	fmt.Printf("%v, Grp: %3v, Times: %10d, miss:%6v, use: %12v, %8v/op\n",
		runtime.Version(), "Sum", Sum, 0, Use, op)
}

func TestQueuePutGetOrder(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	grp := 1
	cnt := 100

	testQueuePutGetOrder(t, grp, cnt)
	t.Logf("Grp: %d, Times: %d", grp, cnt)
}

type safeMap struct {
	data map[interface{}]int
	mu   sync.Mutex
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
			} else if (tail+1)&q.mask == head {
				full = true
			}
			if tail >= head {
				quantity = tail - head
			} else {
				quantity = q.mask + (tail - head)
			}

			nput := atomic.LoadUint32(&put)
			nget := atomic.LoadUint32(&get)

			fmt.Printf("put: %d; get: %d; full: %t; empty: %t; size: %d; head: %d; tail: %d \n",
				nput, nget, full, empty, quantity, head, tail)

			if oput == nput && oget == nget {
				for i := range q.data {
					fmt.Printf("value: %v; position: %d; \n", q.data[i].value,
						q.data[i].position)
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
					if val >= jump/num*circle {
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
					if val == jump/4 {
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
					if val >= jump/num*circle {
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

func TestRbQueuePutGet(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cnt := 10000
	sum := 0
	start := time.Now()
	var putD, getD time.Duration
	for i := 0; i <= runtime.NumCPU()*4; i++ {
		sum += i * cnt
		put, get := testRbQueuePutGet(t, i, cnt)
		putD += put
		getD += get
	}
	end := time.Now()
	use := end.Sub(start)
	op := use / time.Duration(sum)
	t.Logf("Grp: %d, Times: %d, use: %v, %v/op", runtime.NumCPU()*4, sum, use, op)
	t.Logf("Put: %d, use: %v, %v/op", sum, putD, putD/time.Duration(sum))
	t.Logf("Get: %d, use: %v, %v/op", sum, getD, getD/time.Duration(sum))
}

func TestRbQueueGeneral(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var miss, Sum int
	var Use time.Duration
	for i := 1; i <= runtime.NumCPU()*4; i++ {
		cnt := 10000 * 100
		if i > 9 {
			cnt = 10000 * 10
		}
		sum := i * cnt
		start := time.Now()
		miss = testRbQueueGeneral(t, i, cnt)
		end := time.Now()
		use := end.Sub(start)
		op := use / time.Duration(sum)
		fmt.Printf("%v, Grp: %3d, Times: %10d, miss:%6v, use: %12v, %8v/op\n",
			runtime.Version(), i, sum, miss, use, op)
		Use += use
		Sum += sum
	}
	op := Use / time.Duration(Sum)
	fmt.Printf("%v, Grp: %3v, Times: %10d, miss:%6v, use: %12v, %8v/op\n",
		runtime.Version(), "Sum", Sum, 0, Use, op)
}

func TestRbQueuePutGoGet(t *testing.T) {
	var Sum, miss int
	var Use time.Duration
	for i := 1; i <= runtime.NumCPU()*4; i++ {
		//	for i := 2; i <= 2; i++ {
		cnt := 10000 * 100
		if i > 9 {
			cnt = 10000 * 10
		}
		sum := i * cnt
		start := time.Now()
		miss = testRbQueuePutGoGet(t, i, cnt)

		end := time.Now()
		use := end.Sub(start)
		op := use / time.Duration(sum)
		fmt.Printf("%v, Grp: %3d, Times: %10d, miss:%6v, use: %12v, %8v/op\n",
			runtime.Version(), i, sum, miss, use, op)
		Use += use
		Sum += sum
	}
	op := Use / time.Duration(Sum)
	fmt.Printf("%v, Grp: %3v, Times: %10d, miss:%6v, use: %12v, %8v/op\n",
		runtime.Version(), "Sum", Sum, 0, Use, op)
}

func TestRbQueuePutGetOrder(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	grp := 1
	cnt := 100

	testRbQueuePutGetOrder(t, grp, cnt)
	t.Logf("Grp: %d, Times: %d", grp, cnt)
}

func testQueuePutGet(t *testing.T, grp, cnt int) (
	put time.Duration, get time.Duration) {
	var wg sync.WaitGroup
	var id int32
	wg.Add(grp)
	q := NewQueue(1024 * 1024)
	start := time.Now()
	for i := 0; i < grp; i++ {
		go func(g int) {
			defer wg.Done()
			for j := 0; j < cnt; j++ {
				val := fmt.Sprintf("Node.%d.%d.%d", g, j, atomic.AddInt32(&id, 1))
				ok := q.Put(&val)
				for !ok {
					time.Sleep(time.Microsecond)
					ok = q.Put(&val)
				}
			}
		}(i)
	}
	wg.Wait()
	end := time.Now()
	put = end.Sub(start)

	wg.Add(grp)
	start = time.Now()
	for i := 0; i < grp; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < cnt; {
				_, ok := q.Get()
				if !ok {
					runtime.Gosched()
				} else {
					j++
				}
			}
		}()
	}
	wg.Wait()
	end = time.Now()
	get = end.Sub(start)
	if q := q.Quantity(); q != 0 {
		t.Errorf("Grp:%v, Quantity Error: [%v] <>[%v]", grp, q, 0)
	}
	return put, get
}

func testRbQueuePutGet(t *testing.T, grp, cnt int) (
	put time.Duration, get time.Duration) {
	var wg sync.WaitGroup
	var id int32
	wg.Add(grp)
	q := NewRbQueue(1024 * 1024)
	start := time.Now()
	for i := 0; i < grp; i++ {
		go func(g int) {
			defer wg.Done()
			for j := 0; j < cnt; j++ {
				val := fmt.Sprintf("Node.%d.%d.%d", g, j, atomic.AddInt32(&id, 1))
				ok := q.Put(&val)
				for !ok {
					time.Sleep(time.Microsecond)
					ok = q.Put(&val)
				}
			}
		}(i)
	}
	wg.Wait()
	end := time.Now()
	put = end.Sub(start)

	wg.Add(grp)
	start = time.Now()
	for i := 0; i < grp; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < cnt; {
				_, ok := q.Get()
				if !ok {
					runtime.Gosched()
				} else {
					j++
				}
			}
		}()
	}
	wg.Wait()
	end = time.Now()
	get = end.Sub(start)
	if q := q.Quantity(); q != 0 {
		t.Errorf("Grp:%v, Quantity Error: [%v] <>[%v]", grp, q, 0)
	}
	return put, get
}

func testQueueGeneral(t *testing.T, grp, cnt int) int {

	var wg sync.WaitGroup
	var idPut, idGet int32
	var miss int32

	wg.Add(grp)
	q := NewQueue(1024 * 1024)
	for i := 0; i < grp; i++ {
		go func(g int) {
			defer wg.Done()
			for j := 0; j < cnt; j++ {
				val := fmt.Sprintf("Node.%d.%d.%d", g, j, atomic.AddInt32(&idPut, 1))
				ok := q.Put(&val)
				for !ok {
					time.Sleep(time.Microsecond)
					ok = q.Put(&val)
				}
			}
		}(i)
	}

	wg.Add(grp)
	for i := 0; i < grp; i++ {
		go func(g int) {
			defer wg.Done()
			ok := false
			for j := 0; j < cnt; j++ {
				_, ok = q.Get()
				for !ok {
					atomic.AddInt32(&miss, 1)
					time.Sleep(time.Microsecond * 50)
					_, ok = q.Get()
				}
				atomic.AddInt32(&idGet, 1)
			}
		}(i)
	}
	wg.Wait()
	if q := q.Quantity(); q != 0 {
		t.Errorf("Grp:%v, Quantity Error: [%v] <>[%v]", grp, q, 0)
	}
	return int(miss)
}

func testRbQueueGeneral(t *testing.T, grp, cnt int) int {

	var wg sync.WaitGroup
	var idPut, idGet int32
	var miss int32

	wg.Add(grp)
	q := NewRbQueue(1024 * 1024)
	for i := 0; i < grp; i++ {
		go func(g int) {
			defer wg.Done()
			for j := 0; j < cnt; j++ {
				val := fmt.Sprintf("Node.%d.%d.%d", g, j, atomic.AddInt32(&idPut, 1))
				ok := q.Put(&val)
				for !ok {
					time.Sleep(time.Microsecond)
					ok = q.Put(&val)
				}
			}
		}(i)
	}

	wg.Add(grp)
	for i := 0; i < grp; i++ {
		go func(g int) {
			defer wg.Done()
			ok := false
			for j := 0; j < cnt; j++ {
				_, ok = q.Get()
				for !ok {
					atomic.AddInt32(&miss, 1)
					time.Sleep(time.Microsecond * 50)
					_, ok = q.Get()
				}
				atomic.AddInt32(&idGet, 1)
			}
		}(i)
	}
	wg.Wait()
	if q := q.Quantity(); q != 0 {
		t.Errorf("Grp:%v, Quantity Error: [%v] <>[%v]", grp, q, 0)
	}
	return int(miss)
}

var (
	value int = 1
)

func testQueuePutGoGet(t *testing.T, grp, cnt int) int {
	var wg sync.WaitGroup
	wg.Add(grp)
	q := NewQueue(1024 * 1024)
	for i := 0; i < grp; i++ {
		go func(g int) {
			ok := false
			for j := 0; j < cnt; j++ {
				ok = q.Put(&value)
				//var miss int32
				for !ok {
					ok = q.Put(&value)
				}
			}
			wg.Done()
		}(i)
	}
	wg.Add(grp)
	for i := 0; i < grp; i++ {
		go func(g int) {
			ok := false
			for j := 0; j < cnt; j++ {
				//var miss int32
				_, ok = q.Get()
				for !ok {
					_, ok = q.Get()
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	return 0
}

func testRbQueuePutGoGet(t *testing.T, grp, cnt int) int {
	var wg sync.WaitGroup
	wg.Add(grp)
	q := NewRbQueue(1024 * 1024)
	for i := 0; i < grp; i++ {
		go func(g int) {
			ok := false
			for j := 0; j < cnt; j++ {
				ok = q.Put(&value)
				//var miss int32
				for !ok {
					ok = q.Put(&value)
				}
			}
			wg.Done()
		}(i)
	}
	wg.Add(grp)
	for i := 0; i < grp; i++ {
		go func(g int) {
			ok := false
			for j := 0; j < cnt; j++ {
				//var miss int32
				_, ok = q.Get()
				for !ok {
					_, ok = q.Get()
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	return 0
}

func testQueuePutGetOrder(t *testing.T, grp, cnt int) (
	residue int) {
	var wg sync.WaitGroup
	var idPut, idGet int32
	wg.Add(grp)
	q := NewQueue(1024 * 1024)
	for i := 0; i < grp; i++ {
		go func(g int) {
			defer wg.Done()
			for j := 0; j < cnt; j++ {
				v := atomic.AddInt32(&idPut, 1)
				ok := q.Put(v)
				for !ok {
					time.Sleep(time.Microsecond)
					ok = q.Put(v)
				}
			}
		}(i)
	}
	wg.Wait()
	wg.Add(grp)
	for i := 0; i < grp; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < cnt; {
				val, ok := q.Get()
				if !ok {
					fmt.Printf("Get.Fail\n")
					runtime.Gosched()
				} else {
					j++
					idGet++
					if idGet != val.(int32) {
						t.Logf("Get.Err %d <> %d\n", idGet, val)
					}
				}
			}
		}()
	}
	wg.Wait()
	return
}

func testRbQueuePutGetOrder(t *testing.T, grp, cnt int) (
	residue int) {
	var wg sync.WaitGroup
	var idPut, idGet int32
	wg.Add(grp)
	q := NewRbQueue(1024 * 1024)
	for i := 0; i < grp; i++ {
		go func(g int) {
			defer wg.Done()
			for j := 0; j < cnt; j++ {
				v := atomic.AddInt32(&idPut, 1)
				ok := q.Put(v)
				for !ok {
					time.Sleep(time.Microsecond)
					ok = q.Put(v)
				}
			}
		}(i)
	}
	wg.Wait()
	wg.Add(grp)
	for i := 0; i < grp; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < cnt; {
				val, ok := q.Get()
				if !ok {
					fmt.Printf("Get.Fail\n")
					runtime.Gosched()
				} else {
					j++
					idGet++
					if idGet != val.(int32) {
						t.Logf("Get.Err %d <> %d\n", idGet, val)
					}
				}
			}
		}()
	}
	wg.Wait()
	return
}
