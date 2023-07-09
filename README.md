# PROMISE
[![Go Report Card](https://goreportcard.com/badge/github.com/chebyrash/promise)](https://goreportcard.com/report/github.com/chebyrash/promise)
[![Build Status](https://github.com/chebyrash/promise/actions/workflows/test.yml/badge.svg)](https://github.com/chebyrash/promise/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/chebyrash/promise.svg)](https://pkg.go.dev/github.com/chebyrash/promise)

## Introduction

`promise` allows you to write async code in sync fashion

- First class [context.Context](https://blog.golang.org/context) support
- Automatic panic recovery
- Generics support
- Goroutine pool support
	- [sourcegraph/conc](https://github.com/sourcegraph/conc)
	- [panjf2000/ants](https://github.com/panjf2000/ants)
	- Your own!

## Install

    $ go get github.com/chebyrash/promise

## Quickstart
```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chebyrash/promise"
)

func main() {
	p1 := promise.New(func(resolve func(int), reject func(error)) {
		factorial := findFactorial(20)
		resolve(factorial)
	})
	p2 := promise.New(func(resolve func(string), reject func(error)) {
		ip, err := fetchIP()
		if err != nil {
			reject(err)
		} else {
			resolve(ip)
		}
	})

	factorial, _ := p1.Await(context.Background())
	fmt.Println(*factorial)

	IP, _ := p2.Await(context.Background())
	fmt.Println(*IP)
}

func findFactorial(n int) int {
	if n == 1 {
		return 1
	}
	return n * findFactorial(n-1)
}

func fetchIP() (string, error) {
	resp, err := http.Get("https://httpbin.org/ip")
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	type Response struct {
		Origin string `json:"origin"`
	}
	var response Response

	err = json.NewDecoder(resp.Body).Decode(&response)
	return response.Origin, err
}
```

## Pool

- Promise execution can be dispatched to distinct pools, providing granular control over task distribution and concurrency.

- Better performance can be achieved by allowing different stages of a Promise chain to be executed on different goroutine pools, optimizing for the specific requirements of each task.

```go
package main

import (
	"context"

	"github.com/chebyrash/promise"
)

func main() {
	ctx := context.Background()

	// fetches data from API, runs on ioOptimizedPool
	dataPromise := promise.NewWithPool(func(resolve func(string), reject func(error)) {
		data, err := fetchDataFromAPI()
		if err != nil {
			reject(err)
		} else {
			resolve(data)
		}
	}, ioOptimizedPool)

	// computes result based on the fetched data, runs on cpuOptimizedPool
	resultPromise := promise.ThenWithPool(dataPromise, ctx, func(data string) (string, error) {
		result, err := computeResult(data)
		return result, err
	}, cpuOptimizedPool)
}
```