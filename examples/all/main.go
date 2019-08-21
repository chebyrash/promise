package main

import (
	"fmt"
	"github.com/chebyrash/promise"
)

func main() {
	var p1 = promise.Resolve(123)
	var p2 = promise.Resolve("Hello, World")
	var p3 = promise.Resolve([]string{"one", "two", "three"})

	results, _ := promise.All(p1, p2, p3).Await()
	fmt.Println(results)
}
