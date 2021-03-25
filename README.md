# Async, Await, Go Promise (WIP)

The `github.com/maniartech/async` is a tiny go library that aims to simplify the go routine orchestration using easy to handle Async/Await pattern. This library is currently under development and very soon it will be available during April 2021.

## Getting Started

Run following command in your project to get the `async`.
```sh
# Not yet ready!
go get github.com/maniartech/async
```


## Simple Async/Await
A simple async await!
```go
package main

import "github.com/maniartech/async"


func main() {
  // Create a new go routine and wait for it to finish
  p1 := async.Go(Process).Await()

  // Pass the result of previous go routine to the next one!
  p2 := async.Go(Process2, p1.Result.Value).Await()

  print(p1.Result.Value, p2.Result.value)
}
```

## Complex Go Routine Orchestration
The following example shows how a complex go routines' pipeline can be orachastrated using simple structure!
```go
import "github.com/maniartech/async"


// HandleResource processes the various activities
// on the specified resouce. All these activities
// are invoked using their own go routines.
//
// This orchestration provides the parallel, faster yet
// controlled execution of various activities.
func HandleResource(resourceId int) {
  async.Go( // Go: Parallel execution
    async.Go(fetchResource,  resourceId),
    async.GoQ( // GoQ: Sequential execution
      async.Go(processResource),
      async.Go(submitResource),
    ),
    async.Go(
      async.Go(emailResource, resourceId),
      async.Go(publishResource, resourceId),
    ),
  ).Await()
}
```
