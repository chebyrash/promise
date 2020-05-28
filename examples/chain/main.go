package main

import (
	"fmt"

	"github.com/chebyrash/promise"
)

func main() {
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
}
