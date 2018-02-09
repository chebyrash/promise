package promise

import "errors"

const (
	pending = iota
	fulfilled
	rejected
)

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
	// resolve function to resolve the promise or else rejects it if an error occurred.
	executor func(resolve func(interface{}), reject func(error))

	// Appends fulfillment to the promise,
	// and returns a new promise
	then []func(data interface{})

	// Appends a rejection handler to the promise,
	// and returns a new promise.
	catch []func(error error)

	// Stores the result passed to resolve()
	result interface{}

	// Stores the error passed to reject()
	error error
}

func New(executor func(resolve func(interface{}), reject func(error))) *Promise {
	var promise = &Promise{
		state:    pending,
		executor: executor,
		then:     make([]func(interface{}), 0),
		catch:    make([]func(error), 0),
		result:   nil,
		error:    nil,
	}

	go func() {
		defer promise.handlePanic()
		promise.executor(promise.resolve, promise.reject)
	}()

	return promise
}

func (promise *Promise) resolve(resolution interface{}) {
	if !promise.IsPending() {
		return
	}
	promise.result = resolution
	for len(promise.then) == 0 {
	}
	for _, value := range promise.then {
		value(promise.result)
	}
	promise.state = fulfilled
}

func (promise *Promise) reject(error error) {
	if !promise.IsPending() {
		return
	}
	promise.error = error
	for len(promise.catch) == 0 {
	}
	for _, value := range promise.catch {
		value(promise.error)
	}
	promise.state = rejected
}

func (promise *Promise) handlePanic() {
	r := recover()
	if r != nil {
		promise.reject(errors.New(r.(string)))
	}
}

func (promise *Promise) Then(fulfillment func(data interface{})) *Promise {
	if promise.IsPending() {
		promise.then = append(promise.then, fulfillment)
	}
	if promise.IsFulfilled() {
		fulfillment(promise.result)
	}
	return promise
}

func (promise *Promise) Catch(rejection func(error error)) *Promise {
	if promise.IsPending() {
		promise.catch = append(promise.catch, rejection)

	}
	if promise.IsRejected() {
		rejection(promise.error)
	}
	return promise
}

func (promise *Promise) IsPending() bool {
	return promise.state == pending
}

func (promise *Promise) IsFulfilled() bool {
	return promise.state == fulfilled
}

func (promise *Promise) IsRejected() bool {
	return promise.state == rejected
}
