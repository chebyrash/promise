package main

import (
	"errors"
	"fmt"

	"github.com/chebyrash/promise"
)

func main() {
	var p = promise.New(func(resolve func(interface{}), reject func(error)) {
		// Do something asynchronously.
		const sum = 2 + 2

		// If your work was successful call resolve() passing the result.
		if sum == 4 {
			resolve(sum)
			return
		}

		// If you encountered an error call reject() passing the error.
		if sum != 4 {
			reject(errors.New("2 + 2 doesnt't equal 4"))
			return
		}

		// If you forgot to check for errors and your function panics the promise will
		// automatically reject.
		// panic() == reject()
	}).
		// You may continue working with the result of
		// a previous async operation.
		Then(func(data interface{}) interface{} {
			fmt.Println("The result is:", data)
			return data.(int) + 1
		}).

		// Handlers can be added even after the success or failure of the asynchronous operation.
		// Multiple handlers may be added by calling .Then or .Catch several times,
		// to be executed independently in insertion order.
		Then(func(data interface{}) interface{} {
			fmt.Println("The new result is:", data)
			return nil
		}).
		Catch(func(error error) error {
			fmt.Println("Error during execution:", error.Error())
			return nil
		})

	// Since handlers are executed asynchronously you can wait for them.
	p.Await()
}
