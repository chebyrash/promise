package main

import (
	"errors"
	"fmt"
	"github.com/chebyrash/promise"
)

func main() {
	var p1 = promise.Reject(errors.New("bad error"))
	_, err := p1.Await()
	fmt.Println(err)
	// bad error
}
