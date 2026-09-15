package promise_test

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chebyrash/promise"
)

func ExamplePromise_Then() {
	ctx := context.Background()
	p := promise.New(func(resolve func(int), _ func(error)) { resolve(42) })
	text := p.Then(ctx, func(n int) (string, error) {
		return strconv.Itoa(n), nil
	})
	length := text.Then(ctx, func(s string) (int, error) { return len(s), nil })
	value, err := length.Await(ctx)
	fmt.Println(value, err)
	// Output: 2 <nil>
}

func ExamplePromise_Catch() {
	ctx := context.Background()
	p := promise.New(func(_ func(int), reject func(error)) {
		reject(fmt.Errorf("temporary failure"))
	})
	recovered := p.Catch(ctx, func(err error) (int, error) {
		return 42, nil
	})
	value, err := recovered.Then(ctx, func(n int) (string, error) {
		return strconv.Itoa(n), nil
	}).Await(ctx)
	fmt.Println(value, err)
	// Output: 42 <nil>
}
