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
	then []func(data interface{})

	// Appends a rejection handler to the promise,
	// and returns a new promise.
	catch []func(error error)

	// Stores the result passed to resolve()
	result interface{}

	// Stores the error passed to reject()
	error error

	// Mutex protects against data race conditions.
	mutex *sync.Mutex

	// WaitGroup allows to block until all callbacks are executed.
	wg *sync.WaitGroup
}

// New instantiates and returns a *Promise object.
func New(executor func(resolve func(interface{}), reject func(error))) *Promise {
	var promise = &Promise{
		state:    pending,
		executor: executor,
		then:     make([]func(interface{}), 0),
		catch:    make([]func(error), 0),
		result:   nil,
		error:    nil,
		mutex:    &sync.Mutex{},
		wg:       &sync.WaitGroup{},
	}

	go func() {
		defer promise.handlePanic()
		promise.executor(promise.resolve, promise.reject)
	}()

	return promise
}

func (promise *Promise) resolve(resolution interface{}) {
	promise.mutex.Lock()
	defer promise.mutex.Unlock()

	if promise.state != pending {
		return
	}

	for x := 0; x < len(promise.catch); x++ {
		promise.wg.Done()
	}

	promise.result = resolution

	for _, value := range promise.then {
		value(promise.result)
		promise.wg.Done()
	}

	promise.state = fulfilled
}

func (promise *Promise) reject(error error) {
	promise.mutex.Lock()
	defer promise.mutex.Unlock()

	if promise.state != pending {
		return
	}

	for x := 0; x < len(promise.then); x++ {
		promise.wg.Done()
	}

	promise.error = error

	for _, value := range promise.catch {
		value(promise.error)
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

// Then appends fulfillment handler to the promise, and returns a new promise.
func (promise *Promise) Then(fulfillment func(data interface{})) *Promise {
	promise.mutex.Lock()
	defer promise.mutex.Unlock()

	if promise.state == pending {
		promise.wg.Add(1)
		promise.then = append(promise.then, fulfillment)
	} else if promise.state == fulfilled {
		promise.wg.Add(1)
		fulfillment(promise.result)
		promise.wg.Done()
	}

	return promise
}

// Catch appends a rejection handler callback to the promise, and returns a new promise.
func (promise *Promise) Catch(rejection func(error error)) *Promise {
	promise.mutex.Lock()
	defer promise.mutex.Unlock()

	if promise.state == pending {
		promise.wg.Add(1)
		promise.catch = append(promise.catch, rejection)
	} else if promise.state == rejected {
		promise.wg.Add(1)
		rejection(promise.error)
		promise.wg.Done()
	}

	return promise
}

// Await is a blocking function that waits for all callbacks to be executed.
func (promise *Promise) Await() {
	promise.wg.Wait()
}

func AwaitAll(promises ...*Promise) {
	for _, promise := range promises {
		promise.Await()
	}
}
