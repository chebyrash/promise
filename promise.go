package promise

import (
	"fmt"
	"sync"
)

// A Promise is a proxy for a value not necessarily known when
// the promise is created. It allows you to associate handlers
// with an asynchronous action's eventual success value or failure reason.
// This lets asynchronous methods return values like synchronous methods:
// instead of immediately returning the final value, the asynchronous method
// returns a promise to supply the value at some point in the future.
type Promise[T any] struct {
	executor func(resolve func(T), reject func(error))
	result   T
	err      error

	pending bool
	mutex   sync.Mutex
	wg      sync.WaitGroup
}

// New creates a new Promise
func New[T any](executor func(resolve func(T), reject func(error))) *Promise[T] {
	if executor == nil {
		panic("executor cannot be nil")
	}
	
	p := &Promise[T]{
		executor: executor,
		pending:  true,
	}

	p.wg.Add(1)

	go func() {
		defer p.handlePanic()
		p.executor(p.resolve, p.reject)
	}()

	return p
}

func (p *Promise[T]) resolve(resolution T) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.pending {
		return
	}

	p.result = resolution
	p.pending = false

	p.wg.Done()
}

func (p *Promise[T]) reject(err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.pending {
		return
	}

	p.err = err
	p.pending = false

	p.wg.Done()
}

func (p *Promise[T]) handlePanic() {
	err := recover()
	if validErr, ok := err.(error); ok {
		p.reject(fmt.Errorf("panic recovery: %w", validErr))
	} else {
		p.reject(fmt.Errorf("panic recovery: %+v", err))
	}
}

// Then appends fulfillment handler to the promise,
// and returns a new promise resolving to the return value of the called handler.
func Then[A, B any](promise *Promise[A], fulfillment func(data A) B) *Promise[B] {
	return New[B](func(resolve func(B), reject func(error)) {
		result, err := promise.Await()
		if err != nil {
			reject(err)
			return
		}
		resolve(fulfillment(result))
	})
}

// Catch appends a rejection handler to the promise,
// and returns a new promise resolving to the return value of the handler.
func Catch[T any](promise *Promise[T], rejection func(err error) error) *Promise[T] {
	return New[T](func(resolve func(T), reject func(error)) {
		result, err := promise.Await()
		if err != nil {
			reject(rejection(err))
			return
		}
		resolve(result)
	})
}

// Await is a blocking function that waits for the promise to resolve
func (promise *Promise[T]) Await() (T, error) {
	promise.wg.Wait()
	return promise.result, promise.err
}
