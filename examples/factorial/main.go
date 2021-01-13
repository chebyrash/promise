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
	return promise.Resolve(findFactorial(n))
}

func main() {
	var factorial1 = findFactorialPromise(5)
	var factorial2 = findFactorialPromise(10)
	var factorial3 = findFactorialPromise(15)

	// Results calculated asynchronously
	results, _ := promise.All(factorial1, factorial2, factorial3).Await()
	values := results.([]promise.Any)

	fmt.Println("Result of 5! is", values[0])
	fmt.Println("Result of 10! is", values[1])
	fmt.Println("Result of 15! is", values[2])
}
