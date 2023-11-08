# Try new ideas here!

```go

import "github.com/maniartech/choreo"


choreography := choreo.New(
  c.Sync(
    choreo.Call(func(context choreo.Context) {

    })
  ),
  c.Async()
)
choreography.Start()

async.GoP(
  async.Go(func(p) {
    complexTask(123, 234, 234)
  })
).WithContext(ctx).Await()



o := choreo.Q(
  choreo.Q(
    choreo.Run()
    choreo.Run(func(ctx, args int) {

    }, "abc")
  ),
  choreo.Call()
).options().Start()


o.onError(func(err error, key string) {

})


o.beforePlay(func() {

})

orchestra = choreographer.Queue{
  choreographer.Concurrent{

  },
  choreographer.Func()
}

orchestra = choreographer.Concurrent{
  choreographer.Queue{

  },
  choreographer.Func()
}

o.Play()

fmt.Println("Play successful")


```


- orchestra.P()
- orchestra.Q()
- orchestgra.Func()
-
- orchestra.Start()
- orchestra.
-
- choreographer.Q()
- choreographer.P()
- choreographer.Func()
-
- choreographer.Start()
-

https://lebum.medium.com/use-of-synchronization-techniques-in-golang-53d75bc0a646