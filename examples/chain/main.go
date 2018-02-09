package main

import (
	"fmt"
	"github.com/chebyrash/promise"
	"sync"
)

func main() {
	var wg = &sync.WaitGroup{}
	wg.Add(3)

	var p = promise.New(func(resolve func(interface{}), reject func(error)) {
		resolve(0)
	})

	p.Then(func(data interface{}) {
		fmt.Println("I will execute first!")
		wg.Done()
	}).Then(func(data interface{}) {
		fmt.Println("And I will execute second!")
		wg.Done()
	}).Then(func(data interface{}) {
		fmt.Println("Oh I'm last :(")
		wg.Done()
	})

	wg.Wait()
}
