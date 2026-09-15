# PROMISE
[![Go Report Card](https://goreportcard.com/badge/github.com/chebyrash/promise)](https://goreportcard.com/report/github.com/chebyrash/promise)
[![Build Status](https://github.com/chebyrash/promise/actions/workflows/test.yml/badge.svg)](https://github.com/chebyrash/promise/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/chebyrash/promise.svg)](https://pkg.go.dev/github.com/chebyrash/promise)

## Introduction

`promise` allows you to write async code in sync fashion

- First class [context.Context](https://blog.golang.org/context) support
- Automatic panic recovery
- Type-safe promise chains with Go 1.27 generic methods
- Goroutine pool support
	- [sourcegraph/conc](https://github.com/sourcegraph/conc)
	- [panjf2000/ants](https://github.com/panjf2000/ants)
	- Your own!

See [performance measurements](BENCHMARKS.md) for before/after benchmarks and
reproduction commands.

## Install

Requires **Go 1.27 or later**.

    $ go get github.com/chebyrash/promise

## Quickstart
```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chebyrash/promise"
)

func main() {
	p1 := promise.New(func(resolve func(int), reject func(error)) {
		factorial := findFactorial(20)
		resolve(factorial)
	})
	p2 := promise.New(func(resolve func(string), reject func(error)) {
		ip, err := fetchIP()
		if err != nil {
			reject(err)
		} else {
			resolve(ip)
		}
	})

	factorial, err := p1.Await(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(factorial)

	IP, err := p2.Await(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(IP)
}

func findFactorial(n int) int {
	if n == 1 {
		return 1
	}
	return n * findFactorial(n-1)
}

func fetchIP() (string, error) {
	resp, err := http.Get("https://httpbin.org/ip")
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	type Response struct {
		Origin string `json:"origin"`
	}
	var response Response

	err = json.NewDecoder(resp.Body).Decode(&response)
	return response.Origin, err
}
```

## Transform values with methods

`Then` can change the result type at each step. Go infers the output type from
its callback; explicit type arguments such as `p.Then[string](...)` also work.

```go
ctx := context.Background()
p := promise.New(func(resolve func(int), reject func(error)) {
    resolve(42)
})

text := p.Then(ctx, func(n int) (string, error) {
    return strconv.Itoa(n), nil
})
length := text.Then(ctx, func(s string) (int, error) {
    return len(s), nil
})
value, err := length.Await(ctx) // int containing 2, nil
```

Import `strconv` for this example. `ThenWithPool`, `Catch`, and `CatchWithPool`
are also methods. The package-level `Then`, `ThenWithPool`, `Catch`, and
`CatchWithPool` functions have been removed; call these methods on a promise.

`Catch` recovers errors with a value of the same type. Successful source values
pass through without running its callback:

```go
recovered := p.Catch(ctx, func(err error) (int, error) {
    return 0, nil // recover with a valid zero value
})
value, err := recovered.Await(ctx)
```

To propagate or wrap an error, return a zero value and the error:
`return 0, fmt.Errorf("load failed: %w", err)`.

## Public API and errors

| Operation | Callback | Result |
| --- | --- | --- |
| `New[T]` / `NewWithPool[T]` | `func(resolve func(T), reject func(error))` | `*Promise[T]` |
| `p.Then[U]` / `p.ThenWithPool[U]` | `func(T) (U, error)` | `*Promise[U]` |
| `p.Catch` / `p.CatchWithPool` | `func(error) (T, error)` | `*Promise[T]` |
| `p.Await(ctx)` | — | `(T, error)` |
| `All[T]` / `AllWithPool[T]` | — | `*Promise[[]T]` |
| `Race[T]` / `RaceWithPool[T]` | — | `*Promise[T]` |

- `Await` returns `(value, nil)` on success and `(zero value, error)` on failure.
  A nil value is a valid success when `T` allows nil, such as a pointer or slice.
- Callback results with a non-nil error discard the accompanying value.
  Errors retain their identity and wrapping, so `errors.Is` and `errors.As` work.
- `reject(nil)` rejects with `promise.ErrNilRejection`; it never creates a
  successful promise without a value.
- Executor, callback, and pool submission panics become promise rejections.
  Error-valued panics retain the original error; other panic values become errors
  containing their formatted value.
- Invalid arguments panic synchronously: nil callbacks, nil contexts or pools,
  nil or uninitialized promises, and empty `All`/`Race` inputs. All aggregate
  inputs are validated before work starts. Create promises with `New` or
  `NewWithPool`; `Promise` has no usable zero value.

These signatures replace the old pointer-returning `Await`, error-only `Catch`,
and package-level chaining functions. There are no compatibility wrappers.

## Concurrency and cancellation

- The first `resolve` or `reject` wins; subsequent calls have no effect.
- Executor and callback panics reject the promise.
- Canceling an `Await` context stops that wait without canceling or settling the
  source promise. An already canceled context takes precedence; later concurrent
  settlement and cancellation may be observed in either order. Cancellation
  returns `ctx.Err()` (`context.Canceled` or `context.DeadlineExceeded`).
- Cancellation observed before `Then` or `Catch` invokes its callback skips the
  callback and rejects the child. `Catch` can recover a source context error when
  its own context is still active. Cancellation does not interrupt an executor
  or callback that is already running.
- `All` preserves input order and fails as soon as a rejection is observed.
  `Race` settles on the first observed result; ties between settled inputs are
  unspecified. Both require at least one input.
- `All` and `Race` settle directly with the default pool once an outcome is
  known. Custom pools receive a settlement task; cancellation after the outcome
  was selected does not replace it. `Then` and `Catch` callbacks still run
  asynchronously with the built-in default pool.
- Pending dependencies wait in internal goroutines, leaving pool workers free
  to execute tasks. Aggregate waiters exit after settlement, even if other
  inputs never settle. Executors themselves are not canceled.
- `Await` returns a value copy. Slices, maps, and pointers retain Go's normal
  sharing of their underlying data; callers must synchronize concurrent mutation.

## Pool

Keep custom pools open until their promise chains settle. Await the final promise
before calling a pool's `Wait` or `Release`: continuations are submitted only when
their dependencies are ready.

- Promise execution can be dispatched to distinct pools, providing granular control over task distribution and concurrency.

- Better performance can be achieved by allowing different stages of a Promise chain to be executed on different goroutine pools, optimizing for the specific requirements of each task.

```go
package main

import (
	"context"

	"github.com/chebyrash/promise"
)

func main() {
	ctx := context.Background()

	// fetches data from API, runs on ioOptimizedPool
	dataPromise := promise.NewWithPool(func(resolve func(string), reject func(error)) {
		data, err := fetchDataFromAPI()
		if err != nil {
			reject(err)
		} else {
			resolve(data)
		}
	}, ioOptimizedPool)

	// computes result based on the fetched data, runs on cpuOptimizedPool
	resultPromise := dataPromise.ThenWithPool(ctx, func(data string) (string, error) {
		result, err := computeResult(data)
		return result, err
	}, cpuOptimizedPool)
}
```
