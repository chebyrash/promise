package promise

import (
	"errors"
	"sync"
)

const (
	pending = iota
	fulfilled
	rejected
)

// A Promise is a proxy for a value not necessarily known when
// the promise is created. It allows you to associate handlers
// with an asynchronous action's eventual success value or failure reason.
// This lets asynchronous methods return values like synchronous methods:
// instead of immediately returning the final value, the asynchronous method
// returns a promise to supply the value at some point in the future.
type Promise struct {
	// A Promise is in one of these states:
	// Pending - 0. Initial state, neither fulfilled nor rejected.
	// Fulfilled - 1. Operation completed successfully.
	// Rejected - 2. Operation failed.
	state int

	// A function that is passed with the arguments resolve and reject.
	// The executor function is executed immediately by the Promise implementation,
	// passing resolve and reject functions (the executor is called
	// before the Promise constructor even returns the created object).
	// The resolve and reject functions, when called, resolve or reject
	// the promise, respectively. The executor normally initiates some
	// asynchronous work, and then, once that completes, either calls the
	// resolve function to resolve the promise or else rejects it if
	// an error or panic occurred.
	executor func(resolve func(interface{}), reject func(error))

	// Appends fulfillment to the promise,
	// and returns a new promise.
	then []func(data interface{}) interface{}

	// Appends a rejection handler to the promise,
	// and returns a new promise.
	catch []func(err error) error

	// Stores the result passed to resolve()
	result interface{}

	// Stores the error passed to reject()
	err error

	// Mutex protects against data race conditions.
	mutex *sync.Mutex

	// WaitGroup allows to block until all callbacks are executed.
	wg *sync.WaitGroup
}

// New instantiates and returns a pointer to the Promise.
func New(executor func(resolve func(interface{}), reject func(error))) *Promise {
	var wg = &sync.WaitGroup{}
	wg.Add(1)

	var promise = &Promise{
		state:    pending,
		executor: executor,
		then:     make([]func(interface{}) interface{}, 0),
		catch:    make([]func(error) error, 0),
		result:   nil,
		err:      nil,
		mutex:    &sync.Mutex{},
		wg:       wg,
	}

	go func() {
		defer promise.handlePanic()
		promise.executor(promise.resolve, promise.reject)
	}()

	return promise
}

func (promise *Promise) resolve(resolution interface{}) {
	promise.mutex.Lock()

	if promise.state != pending {
		promise.mutex.Unlock()
		return
	}

	switch result := resolution.(type) {
	case *Promise:
		res, err := result.Await()
		if err != nil {
			promise.mutex.Unlock()
			promise.reject(err)
			return
		}
		promise.result = res
	default:
		promise.result = result
	}

	promise.wg.Done()
	for range promise.catch {
		promise.wg.Done()
	}

	for _, fn := range promise.then {
		switch result := fn(promise.result).(type) {
		case *Promise:
			res, err := result.Await()
			if err != nil {
				promise.mutex.Unlock()
				promise.reject(err)
				return
			}
			promise.result = res
		default:
			promise.result = result
		}
		promise.wg.Done()
	}

	promise.state = fulfilled

	promise.mutex.Unlock()
}

func (promise *Promise) reject(err error) {
	promise.mutex.Lock()
	defer promise.mutex.Unlock()

	if promise.state != pending {
		return
	}

	promise.err = err

	promise.wg.Done()
	for range promise.then {
		promise.wg.Done()
	}

	for _, fn := range promise.catch {
		promise.err = fn(promise.err)
		promise.wg.Done()
	}

	promise.state = rejected
}

func (promise *Promise) handlePanic() {
	var r = recover()
	if r != nil {
		promise.reject(errors.New(r.(string)))
	}
}

// Then appends fulfillment handler to the Promise, and returns a new promise.
func (promise *Promise) Then(fulfillment func(data interface{}) interface{}) *Promise {
	promise.mutex.Lock()
	defer promise.mutex.Unlock()

	switch promise.state {
	case pending:
		promise.wg.Add(1)
		promise.then = append(promise.then, fulfillment)
	case fulfilled:
		promise.result = fulfillment(promise.result)
	}

	return promise
}

// Catch appends a rejection handler callback to the Promise, and returns a new promise.
func (promise *Promise) Catch(rejection func(err error) error) *Promise {
	promise.mutex.Lock()
	defer promise.mutex.Unlock()

	switch promise.state {
	case pending:
		promise.wg.Add(1)
		promise.catch = append(promise.catch, rejection)
	case rejected:
		promise.err = rejection(promise.err)
	}

	return promise
}

// Await is a blocking function that waits for all callbacks to be executed.
// Returns value and error.
// Call on an already resolved Promise to get its result and error
func (promise *Promise) Await() (interface{}, error) {
	promise.wg.Wait()
	return promise.result, promise.err
}

// All waits for all promises to be resolved, or for any to be rejected.
// If the returned promise resolves, it is resolved with an aggregating array of the values
// from the resolved promises in the same order as defined in the iterable of multiple promises.
// If it rejects, it is rejected with the reason from the first promise in the iterable that was rejected.
func All(promises ...*Promise) *Promise {
	psLen := len(promises)
	if psLen == 0 {
		return Resolve(make([]interface{}, 0))
	}

	return New(func(resolve func(interface{}), reject func(error)) {
		resolutionsChan := make(chan []interface{}, psLen)
		errorChan := make(chan error, psLen)

		for index, promise := range promises {
			func(i int) {
				promise.Then(func(data interface{}) interface{} {
					resolutionsChan <- []interface{}{i, data}
					return data
				}).Catch(func(err error) error {
					errorChan <- err
					return err
				})
			}(index)
		}

		resolutions := make([]interface{}, psLen)
		for x := 0; x < psLen; x++ {
			select {
			case resolution := <-resolutionsChan:
				resolutions[resolution[0].(int)] = resolution[1]

			case err := <-errorChan:
				reject(err)
				return
			}
		}
		resolve(resolutions)
	})
}

// Race waits until any of the promises is resolved or rejected.
// If the returned promise resolves, it is resolved with the value of the first promise in the iterable
// that resolved. If it rejects, it is rejected with the reason from the first promise that was rejected.
func Race(promises ...*Promise) *Promise {
	psLen := len(promises)
	if psLen == 0 {
		return Resolve(nil)
	}

	return New(func(resolve func(interface{}), reject func(error)) {
		resolutionsChan := make(chan interface{}, psLen)
		errorChan := make(chan error, psLen)

		for _, promise := range promises {
			promise.Then(func(data interface{}) interface{} {
				resolutionsChan <- data
				return data
			}).Catch(func(err error) error {
				errorChan <- err
				return err
			})
		}

		select {
		case resolution := <-resolutionsChan:
			resolve(resolution)

		case err := <-errorChan:
			reject(err)
		}
	})
}

// AllSettled waits until all promises have settled (each may resolve, or reject).
// Returns a promise that resolves after all of the given promises have either resolved or rejected,
// with an array of objects that each describe the outcome of each promise.
func AllSettled(promises ...*Promise) *Promise {
	psLen := len(promises)
	if psLen == 0 {
		return Resolve(make([]interface{}, 0))
	}

	return New(func(resolve func(interface{}), reject func(error)) {
		resolutionsChan := make(chan []interface{}, psLen)

		for index, promise := range promises {
			func(i int) {
				promise.Then(func(data interface{}) interface{} {
					resolutionsChan <- []interface{}{i, data}
					return data
				}).Catch(func(err error) error {
					resolutionsChan <- []interface{}{i, err}
					return err
				})
			}(index)
		}

		resolutions := make([]interface{}, psLen)
		for x := 0; x < psLen; x++ {
			resolution := <-resolutionsChan
			resolutions[resolution[0].(int)] = resolution[1]
		}
		resolve(resolutions)
	})
}

// Resolve returns a Promise that has been resolved with a given value.
func Resolve(resolution interface{}) *Promise {
	return New(func(resolve func(interface{}), reject func(error)) {
		resolve(resolution)
	})
}

// Reject returns a Promise that has been rejected with a given error.
func Reject(err error) *Promise {
	return New(func(resolve func(interface{}), reject func(error)) {
		reject(err)
	})
}
