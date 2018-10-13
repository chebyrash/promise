package main

import (
	"fmt"
	"github.com/chebyrash/promise"
)

func findFactorial(n int) int {
	if n == 1 {
		return 1
	}
	return n * findFactorial(n-1)
}

func findFactorialPromise(n int) *promise.Promise {
	return promise.New(func(resolve func(interface{}), reject func(error)) {
		resolve(findFactorial(n))
	})
}

func main() {
	var factorial1 = findFactorialPromise(5)
	var factorial2 = findFactorialPromise(10)
	var factorial3 = findFactorialPromise(15)

	factorial1.Then(func(data interface{}) interface{} {
		fmt.Println("Result of 5! is", data)
		return nil
	})

	factorial2.Then(func(data interface{}) interface{} {
		fmt.Println("Result of 10! is", data)
		return nil
	})

	factorial3.Then(func(data interface{}) interface{} {
		fmt.Println("Result of 15! is", data)
		return nil
	})

	promise.AwaitAll(factorial1, factorial2, factorial3)
}
