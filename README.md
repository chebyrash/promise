# PROMISE
[![Go Report Card](https://goreportcard.com/badge/github.com/chebyrash/promise)](https://goreportcard.com/report/github.com/chebyrash/promise)
[![Build Status](https://github.com/chebyrash/promise/actions/workflows/test.yml/badge.svg)](https://github.com/chebyrash/promise/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/chebyrash/promise.svg)](https://pkg.go.dev/github.com/chebyrash/promise)

## Install

    $ go get -u github.com/chebyrash/promise

## Introduction

`promise` allows you to write async code in sync fashion

Supports **1.18 generics** and **automatic panic recovery**

## Usage Example
```go
package main

import (
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
			return
		}
		resolve(ip)
	})

	factorial, _ := p1.Await()
	fmt.Println(factorial)

	IP, _ := p2.Await()
	fmt.Println(IP)
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

	type Response struct {
	    Origin string `json:"origin"`
	}
	var response Response

	err = json.NewDecoder(resp.Body).Decode(&response)
	return response.Origin, err
}
```
