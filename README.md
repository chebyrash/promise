# PROMISE
[![Go Report Card](https://goreportcard.com/badge/github.com/chebyrash/promise)](https://goreportcard.com/report/github.com/chebyrash/promise)
[![Build Status](https://travis-ci.org/chebyrash/promise.svg?branch=master)](https://travis-ci.org/chebyrash/promise)
[![](https://godoc.org/github.com/chebyrash/promise?status.svg)](https://pkg.go.dev/github.com/chebyrash/promise?tab=doc)

## About
Promises library for Golang. Inspired by [JS Promises](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise)

Supports:

• Automatic panic recovery

• Nested promise flattening

• Promise cancellation

Note that the library **will be updated** as soon as generics arrive in Go 2.0 to allow for greater type safety, less boilerplate and better performance

## Install

    $ go get -u github.com/chebyrash/promise

## Quick Start
```go
var p = promise.New(func(resolve func(interface{}), reject func(error)) {
  // Do something asynchronously.
  const sum = 2 + 2

  // If your work was successful call resolve() passing the result.
  if sum == 4 {
    resolve(sum)
    return
  }

  // If you encountered an error call reject() passing the error.
  if sum != 4 {
    reject(errors.New("2 + 2 doesnt't equal 4"))
    return
  }

  // If you forgot to check for errors and your function panics the promise will
  // automatically reject.
  // panic() == reject()
}).
  // You may continue working with the result of
  // a previous async operation.
  Then(func(data interface{}) interface{} {
    fmt.Println("The result is:", data)
    return data.(int) + 1
  }).

  // Handlers can be added even after the success or failure of the asynchronous operation.
  // Multiple handlers may be added by calling .Then or .Catch several times,
  // to be executed independently in insertion order.
  Then(func(data interface{}) interface{} {
    fmt.Println("The new result is:", data)
    return nil
  }).
  Catch(func(error error) error {
    fmt.Println("Error during execution:", error.Error())
    return nil
  })

// Since handlers are executed asynchronously you can wait for them.
p.Await()
```

## Methods

### All

Wait for all promises to be resolved, or for any to be rejected.
If the returned promise resolves, it is resolved with an aggregating array of the values from the resolved promises in the same order as defined in the iterable of multiple promises. If it rejects, it is rejected with the reason from the first promise in the iterable that was rejected.

Example:
```go
var p1 = promise.Resolve(123)
var p2 = promise.Resolve("Hello, World")
var p3 = promise.Resolve([]string{"one", "two", "three"})

results, _ := promise.All(p1, p2, p3).Await()
fmt.Println(results)
// [123 Hello, World [one two three]]
```

### AllSettled

Wait until all promises have settled (each may resolve, or reject).
Returns a promise that resolves after all of the given promises have either resolved or rejected, with an array of objects that each describe the outcome of each promise.

Example:
```go
var p1 = promise.Resolve(123)
var p2 = promise.Reject(errors.New("something wrong"))

results, _ := promise.AllSettled(p1, p2).Await()
for _, result := range results.([]interface{}) {
    switch value := result.(type) {
    case error:
        fmt.Printf("Bad error occurred: %s", value.Error())
    default:
        fmt.Printf("Other result type: %d", value.(int))
    }
}
// Other result type: 123
// Bad error occurred: something wrong
```

### Race

Wait until any of the promises is resolved or rejected.
If the returned promise resolves, it is resolved with the value of the first promise in the iterable that resolved. If it rejects, it is rejected with the reason from the first promise that was rejected.

Example:
```go
var p1 = promise.Resolve("Promise 1")
var p2 = promise.Resolve("Promise 2")

fastestResult, _ := promise.Race(p1, p2).Await()

fmt.Printf("Both resolve, but %s is faster", fastestResult)
// Both resolve, but Promise 1 is faster
// OR
// Both resolve, but Promise 2 is faster
```

### Resolve

Returns a new Promise that is resolved with the given value. If the value is a thenable (i.e. has a then method), the returned promise will "follow" that thenable, adopting its eventual state; otherwise the returned promise will be fulfilled with the value.

Example:
```go
var p1 = promise.Resolve("Hello, World")
result, _ := p1.Await()
fmt.Println(result)
// Hello, World
```

### Reject

Returns a new Promise that is rejected with the given reason.

Example:
```go
var p1 = promise.Reject(errors.New("bad error"))
_, err := p1.Await()
fmt.Println(err)
// bad error
```

## Examples

### [HTTP Request](https://github.com/chebyrash/promise/blob/master/examples/http_request/main.go)
```go
var requestPromise = promise.New(func(resolve func(interface{}), reject func(error)) {
  resp, err := http.Get("https://httpbin.org/ip")
  defer resp.Body.Close()
  if err != nil {
    reject(err)
    return
  }

  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    reject(err)
    return
  }
  resolve(body)
})

// Parse JSON body in async manner
parsed, err := requestPromise.
  Then(func(data interface{}) interface{} {
    // This can be a promise, it will automatically flatten
    return parseJSON(data.([]byte))
  }).Await()

if err != nil {
  fmt.Printf("Error: %s\n", err.Error())
  return
}

origin := parsed.(map[string]string)["origin"]
fmt.Printf("Origin: %s\n", origin)
```

### [Finding Factorial](https://github.com/chebyrash/promise/blob/master/examples/factorial/main.go)

```go
func findFactorial(n int) int {
	if n == 1 {
		return 1
	}
	return n * findFactorial(n-1)
}

func findFactorialPromise(n int) *promise.Promise {
	return promise.Resolve(findFactorial(n))
}

func main() {
	var factorial1 = findFactorialPromise(5)
	var factorial2 = findFactorialPromise(10)
	var factorial3 = findFactorialPromise(15)

	// Results calculated asynchronously
	results, _ := promise.All(factorial1, factorial2, factorial3).Await()
	values := results.([]interface{})

	fmt.Println("Result of 5! is", values[0])
	fmt.Println("Result of 10! is", values[1])
	fmt.Println("Result of 15! is", values[2])
}
```

### [Chaining](https://github.com/Chebyrash/promise/blob/master/examples/http_request/main.go)
```go
var p = promise.Resolve(nil).
  Then(func(data interface{}) interface{} {
    fmt.Println("I will execute first")
    return nil
  }).
  Then(func(data interface{}) interface{} {
    fmt.Println("And I will execute second!")
    return nil
  }).
  Then(func(data interface{}) interface{} {
    fmt.Println("Oh I'm last :(")
    return nil
  })

p.Await()
```