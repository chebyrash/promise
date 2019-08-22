package main

import (
	"errors"
	"fmt"
	"github.com/chebyrash/promise"
)

func main() {
	var p1 = promise.Resolve(123)
	var p2 = promise.Reject(errors.New("something wrong"))

	results, _ := promise.AllSettled(p1, p2).Await()
	for _, result := range results.([]interface{}) {
		switch value := result.(type) {
		case error:
			fmt.Printf("Bad error occurred: %s\n", value.Error())
		default:
			fmt.Printf("Other result type: %d\n", value.(int))
		}
	}
}
