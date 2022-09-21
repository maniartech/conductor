# Try new ideas here!

```go
async.Go()

async.GoP(
  async.Go(func(p) {
    complexTask(123, 234, 234)
  })
).WithContext(ctx).Await()

```

https://lebum.medium.com/use-of-synchronization-techniques-in-golang-53d75bc0a646