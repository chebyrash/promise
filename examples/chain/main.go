package main

import (
	"fmt"

	"github.com/chebyrash/promise"
)

func main() {
	var p = promise.Resolve(nil).
		Then(func(data promise.Any) promise.Any {
			fmt.Println("I will execute first")
			return nil
		}).
		Then(func(data promise.Any) promise.Any {
			fmt.Println("And I will execute second!")
			return nil
		}).
		Then(func(data promise.Any) promise.Any {
			fmt.Println("Oh I'm last :(")
			return nil
		})

	p.Await()
}
