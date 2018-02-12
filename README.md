# PROMISE
[![Go Report Card](https://goreportcard.com/badge/github.com/chebyrash/promise)](https://goreportcard.com/report/github.com/chebyrash/promise)
[![Contributions Welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/chebyrash/promise)
[![Build Status](https://travis-ci.org/chebyrash/promise.svg?branch=master)](https://travis-ci.org/chebyrash/promise)

## About
Promises library for Golang. Inspired by [JS Promises.](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise)

## Installation

    $ go get -u github.com/chebyrash/promise
    
## Examples

### [HTTP Request](https://github.com/Chebyrash/promise/blob/master/examples/http_request/main.go)
```go
var requestPromise = promise.New(func(resolve func(interface{}), reject func(error)) {
	var url = "https://httpbin.org/ip"

	resp, err := http.Get(url)
	defer resp.Body.Close()

	if err != nil {
		reject(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		reject(err)
	}

	resolve(string(body))
})

requestPromise.Then(func(data interface{}) {
	fmt.Println(data)
})

requestPromise.Catch(func(error error) {
	fmt.Println(error.Error())
})

requestPromise.Await()
```

### [Finding Factorial](https://github.com/Chebyrash/promise/blob/master/examples/calculation/main.go)

```go
func findFactorial(n int) int {
	if n == 1 {
		return 1
	}
	return n * findFactorial(n-1)
}

func findFactorialPromise(n int) *promise.Promise {
	return promise.New(func(resolve func(interface{}), reject func(error)) {
		resolve(findFactorial(n))
	})
}

func main() {
	var factorial1 = findFactorialPromise(5)
	var factorial2 = findFactorialPromise(10)
	var factorial3 = findFactorialPromise(15)

	factorial1.Then(func(data interface{}) {
		fmt.Println("Result of 5! is", data)
	})

	factorial2.Then(func(data interface{}) {
		fmt.Println("Result of 10! is", data)
	})

	factorial3.Then(func(data interface{}) {
		fmt.Println("Result of 15! is", data)
	})

	factorial1.Await()
	factorial2.Await()
	factorial3.Await()
}
```

### [Chaining](https://github.com/Chebyrash/promise/blob/master/examples/http_request/main.go)
```go
var p = promise.New(func(resolve func(interface{}), reject func(error)) {
	resolve(0)
})

p.Then(func(data interface{}) {
	fmt.Println("I will execute first!")
}).Then(func(data interface{}) {
	fmt.Println("And I will execute second!")
}).Then(func(data interface{}) {
	fmt.Println("Oh I'm last :(")
})

p.Await()
```