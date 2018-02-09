package main

import (
	"fmt"
	"github.com/chebyrash/promise"
	"sync"
)

func findFactorial(n int) int {
	if n == 1 {
		return 1
	}
	return n * findFactorial(n-1)
}

func findFactorialPromise(n int) *promise.Promise {
	var p = promise.New(func(resolve func(interface{}), reject func(error)) {
		resolve(findFactorial(n))
	})
	return p
}

func main() {

	var wg = &sync.WaitGroup{}
	wg.Add(3)

	var factorial1 = findFactorialPromise(5)
	var factorial2 = findFactorialPromise(10)
	var factorial3 = findFactorialPromise(15)

	factorial1.Then(func(data interface{}) {
		fmt.Println("Result of 5! is", data)
		wg.Done()
	})

	factorial2.Then(func(data interface{}) {
		fmt.Println("Result of 10! is", data)
		wg.Done()
	})

	factorial3.Then(func(data interface{}) {
		fmt.Println("Result of 15! is", data)
		wg.Done()
	})

	wg.Wait()

}
