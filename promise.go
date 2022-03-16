package promise

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

// Promise represents the eventual completion (or failure) of an asynchronous operation and its resulting value.
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

// Await blocks until the promise is resolved or rejected.
func (p *Promise[T]) Await() (T, error) {
	p.wg.Wait()
	return p.result, p.err
}

// Then allows to chain promises.
// Use it to add a fulfillment handler to the resolved promise.
func Then[A, B any](promise *Promise[A], fulfillment func(data A) B) *Promise[B] {
	return New(func(resolve func(B), reject func(error)) {
		result, err := promise.Await()
		if err != nil {
			reject(err)
			return
		}
		resolve(fulfillment(result))
	})
}

// Catch allows to chain promises.
// Use it to add an error handler to the rejected promise.
func Catch[T any](promise *Promise[T], rejection func(err error) error) *Promise[T] {
	return New(func(resolve func(T), reject func(error)) {
		result, err := promise.Await()
		if err != nil {
			reject(rejection(err))
			return
		}
		resolve(result)
	})
}

/*
	Utilities
*/

type pair[L, R any] struct {
	left  L
	right R
}

// Resolve returns a Promise that has been resolved with a given value.
func Resolve[T any](resolution T) *Promise[T] {
	return New(func(resolve func(T), reject func(error)) {
		resolve(resolution)
	})
}

// Reject returns a Promise that has been rejected with a given error.
func Reject[T any](err error) *Promise[T] {
	return New(func(resolve func(T), reject func(error)) {
		reject(err)
	})
}

// All returns a Promise that will resolve when all of the input's promises have resolved.
// It rejects immediately upon any of the input promises rejecting.
// If the input is empty, All will return nil.
func All[T any](promises ...*Promise[T]) *Promise[[]T] {
	if len(promises) == 0 {
		return nil
	}

	return New(func(resolve func([]T), reject func(error)) {
		valsChan := make(chan pair[T, int], len(promises))
		errsChan := make(chan error, 1)

		for idx, p := range promises {
			idx := idx // https://golang.org/doc/faq#closures_and_goroutines
			_ = Then(p, func(data T) T {
				valsChan <- pair[T, int]{left: data, right: idx}
				return data
			})
			_ = Catch(p, func(err error) error {
				errsChan <- err
				return err
			})
		}

		resolutions := make([]T, len(promises))
		for idx := 0; idx < len(promises); idx++ {
			select {
			case val := <-valsChan:
				resolutions[val.right] = val.left
			case err := <-errsChan:
				reject(err)
				return
			}
		}
		resolve(resolutions)
	})
}

// Race returns a Promise that fulfills or rejects as soon as one of the input's promises fulfills or rejects,
// with the value or error from that promise.
// If the input is empty, Race will return nil.
func Race[T any](promises ...*Promise[T]) *Promise[T] {
	if len(promises) == 0 {
		return nil
	}

	return New(func(resolve func(T), reject func(error)) {
		valsChan := make(chan T, 1)
		errsChan := make(chan error, 1)

		for _, p := range promises {
			_ = Then(p, func(data T) T {
				valsChan <- data
				return data
			})
			_ = Catch(p, func(err error) error {
				errsChan <- err
				return err
			})
		}

		select {
		case val := <-valsChan:
			resolve(val)
		case err := <-errsChan:
			reject(err)
		}
	})
}

// Any returns a Promise that resolves as soon as any of the input's promises fulfills, with the value of the fulfilled promise.
// If all of the given promises are rejected, then the returned promise is rejected with a combination of all errors.
// If the input is empty, Race will return nil.
func Any[T any](promises ...*Promise[T]) *Promise[T] {
	if len(promises) == 0 {
		return nil
	}

	return New(func(resolve func(T), reject func(error)) {
		var errCombo error
		for _, p := range promises {
			val, err := p.Await()
			if err == nil {
				resolve(val)
				return
			}

			if errCombo == nil {
				errCombo = err
				continue
			}

			errCombo = errors.Wrap(err, errCombo.Error())
		}
		reject(errCombo)
	})
}
