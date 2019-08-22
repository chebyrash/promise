package main

import (
	"fmt"
	"github.com/chebyrash/promise"
)

func main() {
	var p1 = promise.Resolve("Hello, World")
	result, _ := p1.Await()
	fmt.Println(result)
	// Hello, World
}
