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
