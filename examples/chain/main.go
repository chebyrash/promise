package main

import (
	"fmt"
	"github.com/chebyrash/promise"
)

func main() {
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
}
