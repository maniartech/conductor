# Async, Await, Go Promise [![Build Status](https://travis-ci.com/maniartech/async.svg?branch=master)](https://travis-ci.com/maniartech/async)

The `github.com/maniartech/async` is a tiny go library that aims to simplify the goroutine orchestration using easy to handle Async/Await pattern. 

> This library provides a super-easy way to orchestrate the Goroutines using a promise-oriented approach but the Golang way.

## Getting Started

Run following command in your project to get the `async`.
```sh
go get github.com/maniartech/async
```


## Simple Async/Await
A simple async await!
```go
package main

import "github.com/maniartech/async"


func main() {
  // Executes the async function in a new goroutine and
  // awaits for the result.
  result, err := async.Go(Process).Await()
  if err != nil {
    panic("An error occured while executing Process")
  }

  // Pass the result of previous goroutine to the next one!
  result, err = async.Go(Process2, result).Await()
  if err != nil {
    panic("An error occured while executing Process2")
  }

  println(p1.Result.Value, p2.Result.value)
}
```

## Complex Goroutine Orchestration

The following example shows how a complex goroutines' pipeline can be orachastrated using simple structure!

```go
import "github.com/maniartech/async"


// HandleResource processes the various activities
// on the specified resouce. All these activities
// are executed using their own goroutines and
// in an orchastrated manner.
//
// This orchestration provides the concurrent, faster yet
// controlled execution of various activities.
//      |------Go--------------|
//      |                      |
// ----GoC----GoQ->>-Go->>-Go--|--- Await ----
//      |                      |
//      |      |---Go--|       |
//      |-----GoC      |-------|
//             |---Go--|
//
func HandleResource(resourceId int) {
  async.GoC( // GoC: Concurrent execution
    async.Go(fetchResource,  resourceId),
    async.GoQ( // GoQ: Sequential execution
      async.Go(processResource),
      async.Go(submitResource),
    ),
    async.GoC(
      async.Go(emailResource, resourceId),
      async.Go(publishResource, resourceId),
    ),
  ).Await()
}
```

## Commonly used functions



* `async.Go(PromiseHandler, ...interfaces{}) *Promsie`

  Eecutes the funciton in new goroutine and returns a promsie. The promise can be awaited until the execution is finised and results are retunred.

  ```go
  // process is a prmise handler function
  func process(p *Promise, args ...interface{}}) {
    processId := args[0].(int)
    value, err := fetchProcessResource(processId)
    p.Done(value, err)
  }

  // Execute the promise in a new goroutine and wait for the results.
  result, err := aysnc.Go(prosess, 1).Await()

  if err != nil {
    panic(err)
  }
  println(result)
  ```
