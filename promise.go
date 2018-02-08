package promise

const (
	PENDING = iota
	FULFILLED
	REJECTED
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
	then func(data interface{})

	// Appends a rejection handler callback to the promise,
	// and returns a new promise.
	catch func(error error)

	// Stores the result passed to resolve()
	result chan interface{}

	// Stores the error passed to reject()
	error chan error
}

func New(executor func(resolve func(interface{}), reject func(error))) *Promise {
	var promise = &Promise{
		state:    PENDING,
		executor: executor,
		then:     nil,
		catch:    nil,
		result:   make(chan interface{}, 1),
		error:    make(chan error, 1),
	}

	go promise.executor(promise.resolve, promise.reject)
	go promise.process()

	return promise
}

func (promise *Promise) resolve(resolution interface{}) {
	if !promise.IsPending() {
		return
	}
	promise.result <- resolution
	promise.close()
	promise.state = FULFILLED
}

func (promise *Promise) reject(error error) {
	if !promise.IsPending() {
		return
	}
	promise.error <- error
	promise.close()
	promise.state = REJECTED
}

func (promise *Promise) close() {
	close(promise.result)
	close(promise.error)
}

func (promise *Promise) process() {
	select {
	case result := <-promise.result:
		for promise.then == nil {
		}
		promise.then(result)
	case error := <-promise.error:
		for promise.catch == nil {
		}
		promise.catch(error)
	}
}

func (promise *Promise) Then(fulfillment func(data interface{})) *Promise {
	promise.then = fulfillment
	return promise
}

func (promise *Promise) Catch(rejection func(error error)) *Promise {
	promise.catch = rejection
	return promise
}

func (promise *Promise) IsPending() bool {
	return promise.state == PENDING
}

func (promise *Promise) IsFulfilled() bool {
	return promise.state == FULFILLED
}

func (promise *Promise) IsRejected() bool {
	return promise.state == REJECTED
}
