# lock-free ring queue

### Usage

```go
queue := NewQueue(8)
ok := queue.Put(1)
val, ok := queue.Get()
```

#### block operate

```go
// block at least 50ms
ok = queue.PutWait(2, time.Second)
val, ok = queue.GetWait(time.Second)
```

#### stress test

```go
=== RUN   TestRbQueueGeneral
go1.14.2, Grp:   1, Times:    1000000, miss:  4144, use: 393.227304ms,    393ns/op
go1.14.2, Grp:   2, Times:    2000000, miss: 20969, use: 807.913525ms,    403ns/op
go1.14.2, Grp:   3, Times:    3000000, miss: 45288, use: 1.138342491s,    379ns/op
go1.14.2, Grp:   4, Times:    4000000, miss: 63370, use: 1.275393518s,    318ns/op
go1.14.2, Grp:   5, Times:    5000000, miss: 85476, use: 1.467711974s,    293ns/op
go1.14.2, Grp:   6, Times:    6000000, miss:125316, use: 1.773097125s,    295ns/op
go1.14.2, Grp:   7, Times:    7000000, miss:168618, use: 1.929599644s,    275ns/op
go1.14.2, Grp:   8, Times:    8000000, miss:212333, use:  2.18940863s,    273ns/op
go1.14.2, Grp:   9, Times:    9000000, miss:274333, use: 2.448981989s,    272ns/op
go1.14.2, Grp:  10, Times:    1000000, miss: 33367, use: 289.027354ms,    289ns/op
go1.14.2, Grp:  11, Times:    1100000, miss: 37532, use: 308.374394ms,    280ns/op
go1.14.2, Grp:  12, Times:    1200000, miss: 44305, use: 340.100035ms,    283ns/op
go1.14.2, Grp:  13, Times:    1300000, miss: 50891, use: 360.981396ms,    277ns/op
go1.14.2, Grp:  14, Times:    1400000, miss: 54634, use: 385.316676ms,    275ns/op
go1.14.2, Grp:  15, Times:    1500000, miss: 57693, use: 415.020261ms,    276ns/op
go1.14.2, Grp:  16, Times:    1600000, miss: 61032, use: 484.634991ms,    302ns/op
go1.14.2, Grp: Sum, Times:   54100000, miss:     0, use: 16.007131307s,    295ns/op
--- PASS: TestRbQueueGeneral (16.01s)
```
