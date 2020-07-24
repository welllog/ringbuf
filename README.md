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
=== RUN   TestQueueGeneral
go1.14.2, Grp:   1, Times:    1000000, miss:  4225, use: 474.866511ms,    474ns/op
go1.14.2, Grp:   2, Times:    2000000, miss: 21738, use: 972.083324ms,    486ns/op
go1.14.2, Grp:   3, Times:    3000000, miss: 46660, use:  1.19623443s,    398ns/op
go1.14.2, Grp:   4, Times:    4000000, miss: 80429, use: 1.522187672s,    380ns/op
go1.14.2, Grp:   5, Times:    5000000, miss:103987, use: 1.711450626s,    342ns/op
go1.14.2, Grp:   6, Times:    6000000, miss:148825, use: 1.995207422s,    332ns/op
go1.14.2, Grp:   7, Times:    7000000, miss:184464, use: 2.108491305s,    301ns/op
go1.14.2, Grp:   8, Times:    8000000, miss:236993, use: 2.477101217s,    309ns/op
go1.14.2, Grp:   9, Times:    9000000, miss:280990, use: 2.816015924s,    312ns/op
go1.14.2, Grp:  10, Times:    1000000, miss: 31815, use: 345.640898ms,    345ns/op
go1.14.2, Grp:  11, Times:    1100000, miss: 43880, use: 331.291381ms,    301ns/op
go1.14.2, Grp:  12, Times:    1200000, miss: 50667, use: 362.230543ms,    301ns/op
go1.14.2, Grp:  13, Times:    1300000, miss: 55593, use: 385.910124ms,    296ns/op
go1.14.2, Grp:  14, Times:    1400000, miss: 59258, use: 415.664597ms,    296ns/op
go1.14.2, Grp:  15, Times:    1500000, miss: 70241, use: 446.199656ms,    297ns/op
go1.14.2, Grp:  16, Times:    1600000, miss: 63632, use: 504.010162ms,    315ns/op
go1.14.2, Grp: Sum, Times:   54100000, miss:     0, use: 18.064585792s,    333ns/op
--- PASS: TestQueueGeneral (18.07s)
```