package promise

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

// Promise represents the eventual completion (or failure) of an asynchronous operation and its resulting value.
type Promise[T any] struct {
	result T
	err    error

	pending bool
	mutex   *sync.Mutex
	wg      *sync.WaitGroup
}

// New creates a new Promise
func New[T any](executor func(resolve func(T), reject func(error))) *Promise[T] {
	if executor == nil {
		panic("executor cannot be nil")
	}

	p := &Promise[T]{
		pending: true,
		mutex:   &sync.Mutex{},
		wg:      &sync.WaitGroup{},
	}

	p.wg.Add(1)

	go func() {
		defer p.handlePanic()
		executor(p.resolve, p.reject)
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
	if err == nil {
		return
	}
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
// Use it to add a handler to the resolved promise.
func Then[A, B any](promise *Promise[A], resolveA func(data A) B) *Promise[B] {
	return New(func(resolveB func(B), reject func(error)) {
		result, err := promise.Await()
		if err != nil {
			reject(err)
			return
		}
		resolveB(resolveA(result))
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
	Helpers
*/

type tuple[T1, T2 any] struct {
	_1 T1
	_2 T2
}

// Resolve returns a Promise that has been resolved with a given value.
func Resolve[T any](resolution T) *Promise[T] {
	return &Promise[T]{
		result:  resolution,
		pending: false,
		mutex:   &sync.Mutex{},
		wg:      &sync.WaitGroup{},
	}
}

// Reject returns a Promise that has been rejected with a given error.
func Reject[T any](err error) *Promise[T] {
	return &Promise[T]{
		err:     err,
		pending: false,
		mutex:   &sync.Mutex{},
		wg:      &sync.WaitGroup{},
	}
}

// All resolves when all of the input's promises have resolved.
// All rejects immediately upon any of the input promises rejecting.
// All returns nil if the input is empty.
func All[T any](promises ...*Promise[T]) *Promise[[]T] {
	if len(promises) == 0 {
		return nil
	}

	return New(func(resolve func([]T), reject func(error)) {
		valsChan := make(chan tuple[T, int], len(promises))
		errsChan := make(chan error, 1)

		for idx, p := range promises {
			idx := idx // https://golang.org/doc/faq#closures_and_goroutines
			_ = Then(p, func(data T) T {
				valsChan <- tuple[T, int]{_1: data, _2: idx}
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
				resolutions[val._2] = val._1
			case err := <-errsChan:
				reject(err)
				return
			}
		}
		resolve(resolutions)
	})
}

// Race resolves or rejects as soon as one of the input's Promises resolve or reject, with the value or error of that Promise.
// Race returns nil if the input is empty.
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

// Any resolves as soon as any of the input's Promises resolve, with the value of the resolved Promise.
// Any rejects if all of the given Promises are rejected with a combination of all errors.
// Any returns nil if the input is empty.
func Any[T any](promises ...*Promise[T]) *Promise[T] {
	if len(promises) == 0 {
		return nil
	}

	return New(func(resolve func(T), reject func(error)) {
		valsChan := make(chan T, 1)
		errsChan := make(chan tuple[error, int], len(promises))

		for idx, p := range promises {
			idx := idx // https://golang.org/doc/faq#closures_and_goroutines
			_ = Then(p, func(data T) T {
				valsChan <- data
				return data
			})
			_ = Catch(p, func(err error) error {
				errsChan <- tuple[error, int]{_1: err, _2: idx}
				return err
			})
		}

		errs := make([]error, len(promises))
		for idx := 0; idx < len(promises); idx++ {
			select {
			case val := <-valsChan:
				resolve(val)
				return
			case err := <-errsChan:
				errs[err._2] = err._1
			}
		}

		errCombo := errs[0]
		for _, err := range errs[1:] {
			errCombo = errors.Wrap(err, errCombo.Error())
		}
		reject(errCombo)
	})
}
