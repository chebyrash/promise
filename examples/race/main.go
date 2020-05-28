package main

import (
	"fmt"

	"github.com/chebyrash/promise"
)

func main() {
	var p1 = promise.Resolve("Promise 1")
	var p2 = promise.Resolve("Promise 2")

	fastestResult, _ := promise.Race(p1, p2).Await()

	fmt.Printf("Both resolve, but %s is faster\n", fastestResult)
}
