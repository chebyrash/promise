package promise

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
)

// ErrNilRejection is returned when an executor calls reject(nil).
var ErrNilRejection = errors.New("promise: rejected with nil error")

// Promise represents the eventual completion (or failure) of an asynchronous operation.
// Create promises with New or NewWithPool; the zero value is not usable.
// A Promise must not be copied after first use.
type Promise[T any] struct {
	value T
	err   error
	ch    chan struct{}
	once  sync.Once
}

func newPromise[T any]() *Promise[T] {
	return &Promise[T]{ch: make(chan struct{})}
}

// New runs executor using the default pool, initially one goroutine per executor.
// The first call to resolve or reject wins. Panics are converted to rejections.
// Calling reject(nil) rejects with ErrNilRejection. A nil executor panics.
// Returning from executor alone does not settle the promise; it may arrange to
// call resolve or reject later, including from another goroutine.
func New[T any](executor func(resolve func(T), reject func(error))) *Promise[T] {
	return NewWithPool(executor, getDefaultPool())
}

// NewWithPool behaves like New, using pool to run executor. The pool determines
// whether execution is asynchronous. A nil executor or pool panics.
// Submission failures are returned by Await as errors.
func NewWithPool[T any](executor func(resolve func(T), reject func(error)), pool Pool) *Promise[T] {
	if executor == nil {
		panic("executor is nil")
	}
	if pool == nil {
		panic("pool is nil")
	}
	p := newPromise[T]()
	p.submit(pool, func() {
		defer p.handlePanic()
		executor(p.resolve, p.reject)
	})
	return p
}

func (p *Promise[T]) submit(pool Pool, f func()) {
	defer p.handlePanic()
	pool.Go(f)
}

// schedule recovers both submission failures and panics in the submitted task.
func (p *Promise[T]) schedule(pool Pool, f func()) {
	p.submit(pool, func() {
		defer p.handlePanic()
		select {
		case <-p.ch:
			return
		default:
			f()
		}
	})
}

// Then transforms a resolved value of type T into a value of type U. A rejected
// source propagates its error without calling resolve. A callback error discards
// the callback's value. Cancellation observed before invocation skips resolve.
// Panics in resolve become rejections. A nil context or callback, or a nil or
// uninitialized receiver, panics synchronously.
func (p *Promise[T]) Then[U any](ctx context.Context, resolve func(T) (U, error)) *Promise[U] {
	return p.ThenWithPool(ctx, resolve, getDefaultPool())
}

// ThenWithPool behaves like Then, using pool to run the transformation once p
// is ready. A nil pool panics, as do the invalid arguments described by Then.
func (p *Promise[T]) ThenWithPool[U any](ctx context.Context, resolve func(T) (U, error), pool Pool) *Promise[U] {
	p.validate()
	validateContext(ctx)
	if resolve == nil {
		panic("resolve is nil")
	}
	if pool == nil {
		panic("pool is nil")
	}
	q := newPromise[U]()
	p.dispatch(ctx, pool, q, func() {
		defer q.handlePanic()
		value, err := p.Await(ctx)
		if ctxErr := ctx.Err(); ctxErr != nil {
			q.reject(ctxErr)
			return
		}
		if err != nil {
			q.reject(err)
			return
		}
		result, err := resolve(value)
		if err != nil {
			q.reject(err)
			return
		}
		q.resolve(result)
	})
	return q
}

// Catch recovers a source rejection with a value of type T, or returns a new
// error. Successful source values pass through without calling recoverValue.
// A callback error discards its value. Cancellation of ctx skips recoverValue;
// a context error from the source is recoverable while ctx remains active.
// Callback panics become rejections. A nil context or callback, or a nil or
// uninitialized receiver, panics synchronously.
func (p *Promise[T]) Catch(ctx context.Context, recoverValue func(error) (T, error)) *Promise[T] {
	return p.CatchWithPool(ctx, recoverValue, getDefaultPool())
}

// CatchWithPool behaves like Catch, using pool to run the recovery handler once
// p is ready. A nil pool panics, as do the invalid arguments described by Catch.
func (p *Promise[T]) CatchWithPool(ctx context.Context, recoverValue func(error) (T, error), pool Pool) *Promise[T] {
	p.validate()
	validateContext(ctx)
	if recoverValue == nil {
		panic("recoverValue is nil")
	}
	if pool == nil {
		panic("pool is nil")
	}
	q := newPromise[T]()
	p.dispatch(ctx, pool, q, func() {
		defer q.handlePanic()
		value, err := p.Await(ctx)
		if ctxErr := ctx.Err(); ctxErr != nil {
			q.reject(ctxErr)
			return
		}
		if err != nil {
			value, err = recoverValue(err)
			if err != nil {
				q.reject(err)
				return
			}
		}
		q.resolve(value)
	})
	return q
}

// Await returns the resolved value and nil, or the zero value of T and an error.
// Canceling this wait does not cancel or settle the source promise. A context
// already canceled on entry takes precedence over a settled result; otherwise
// concurrent settlement and cancellation may be observed in either order.
// The value is copied, with Go's normal sharing for slices, maps, and pointers.
// A nil context, nil receiver, or uninitialized promise panics synchronously.
func (p *Promise[T]) Await(ctx context.Context) (T, error) {
	p.validate()
	validateContext(ctx)
	done := ctx.Done()
	if done == nil {
		<-p.ch
		return p.result()
	}
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	select {
	case <-done:
		var zero T
		return zero, ctx.Err()
	case <-p.ch:
		return p.result()
	}
}

// result may only be read after receiving from the closed settlement channel.
func (p *Promise[T]) result() (T, error) {
	return p.value, p.err
}

// poll reads a ready result once, avoiding a second select in aggregation.
func (p *Promise[T]) poll(ctx context.Context) (T, error, bool) {
	var zero T
	done := ctx.Done()
	if done == nil {
		select {
		case <-p.ch:
			value, err := p.result()
			return value, err, true
		default:
			return zero, nil, false
		}
	}
	if err := ctx.Err(); err != nil {
		return zero, err, true
	}
	select {
	case <-done:
		return zero, ctx.Err(), true
	case <-p.ch:
		value, err := p.result()
		return value, err, true
	default:
		return zero, nil, false
	}
}

// Aggregate settlement runs no user code, so the default pool needs no extra
// goroutine. Explicit custom pools still control where settlement runs.
func (p *Promise[T]) resolveWithPool(pool Pool, value T) {
	if _, unbounded := pool.(goroutinePool); unbounded {
		p.resolve(value)
		return
	}
	p.schedule(pool, func() { p.resolve(value) })
}

func (p *Promise[T]) rejectWithPool(pool Pool, err error) {
	if _, unbounded := pool.(goroutinePool); unbounded {
		p.reject(err)
		return
	}
	p.schedule(pool, func() { p.reject(err) })
}

func (p *Promise[T]) resolve(value T) {
	p.once.Do(func() {
		p.value = value
		close(p.ch)
	})
}

func (p *Promise[T]) reject(err error) {
	p.once.Do(func() {
		if err == nil {
			err = ErrNilRejection
		}
		p.err = err
		close(p.ch)
	})
}

func (p *Promise[T]) validate() {
	if p == nil || p.ch == nil {
		panic("promise is nil or uninitialized")
	}
}

func validateContext(ctx context.Context) {
	if ctx == nil {
		panic("context is nil")
	}
}

func validateInputs[T any](ctx context.Context, pool Pool, promises []*Promise[T]) {
	validateContext(ctx)
	if pool == nil {
		panic("pool is nil")
	}
	if len(promises) == 0 {
		panic("missing promises")
	}
	for _, p := range promises {
		p.validate()
	}
}

func (p *Promise[T]) handlePanic() {
	if err := recover(); err != nil {
		switch v := err.(type) {
		case error:
			p.reject(v)
		default:
			p.reject(fmt.Errorf("%+v", v))
		}
	}
}

func (p *Promise[T]) ready(ctx context.Context) bool {
	select {
	case <-p.ch:
		return true
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// dispatch lets the unbounded default pool wait and transform in one goroutine.
// Custom pools receive work only once the source is ready.
func (p *Promise[T]) dispatch[U any](ctx context.Context, pool Pool, q *Promise[U], f func()) {
	if _, unbounded := pool.(goroutinePool); unbounded {
		q.submit(pool, f)
		return
	}
	if p.ready(ctx) {
		q.submit(pool, f)
		return
	}
	p.observe(ctx, q.ch, func() { q.submit(pool, f) })
}

// observe avoids a goroutine for ready inputs. Pending inputs wait outside the
// execution pool so a bounded pool can still run the work they depend on.
// done releases losing Race waiters and unfinished All waiters after rejection.
func (p *Promise[T]) observe(ctx context.Context, done <-chan struct{}, f func()) {
	select {
	case <-done:
		return
	default:
	}
	select {
	case <-p.ch:
		f()
	case <-ctx.Done():
		f()
	default:
		go func() {
			select {
			case <-done:
				return
			case <-p.ch:
			case <-ctx.Done():
			}
			f()
		}()
	}
}

// observeInputs consumes ready inputs together. Once an input is pending, a
// single coordinator takes over registration so callers can continue producing
// results without paying to start every waiter first. Copy the remaining slice
// because the caller may reuse its variadic arguments after this call returns.
func observeInputs[T any](ctx context.Context, done <-chan struct{}, finished *atomic.Bool, promises []*Promise[T], consume func(int, T, error)) {
	for i, p := range promises {
		if finished.Load() {
			return
		}
		if value, err, ready := p.poll(ctx); ready {
			consume(i, value, err)
			continue
		}
		pending := slices.Clone(promises[i:])
		go func() {
			for j, p := range pending {
				if finished.Load() {
					return
				}
				if value, err, ready := p.poll(ctx); ready {
					consume(i+j, value, err)
				} else {
					p.observe(ctx, done, func() {
						value, err := p.Await(ctx)
						consume(i+j, value, err)
					})
				}
			}
		}()
		return
	}
}

// All resolves with results in input order, or rejects upon any rejection.
// It uses the default pool. Nil contexts, empty inputs, and nil or uninitialized
// input promises panic synchronously, before observing any inputs.
func All[T any](ctx context.Context, promises ...*Promise[T]) *Promise[[]T] {
	return AllWithPool(ctx, getDefaultPool(), promises...)
}

// AllWithPool dispatches settlement through custom pools. Dependency waits do
// not occupy pool workers. It otherwise behaves like All; a nil pool also panics.
func AllWithPool[T any](ctx context.Context, pool Pool, promises ...*Promise[T]) *Promise[[]T] {
	validateInputs(ctx, pool, promises)
	q := newPromise[[]T]()
	results := make([]T, len(promises))
	state := &struct {
		remaining atomic.Int64
		finished  atomic.Bool
	}{}
	state.remaining.Store(int64(len(promises)))
	consume := func(i int, value T, err error) {
		defer q.handlePanic()
		if err != nil {
			if state.finished.CompareAndSwap(false, true) {
				q.rejectWithPool(pool, err)
			}
			return
		}
		results[i] = value
		// Each waiter owns one index; the atomic counter publishes every
		// write before the final waiter resolves the complete slice.
		if state.remaining.Add(-1) == 0 && state.finished.CompareAndSwap(false, true) {
			q.resolveWithPool(pool, results)
		}
	}
	observeInputs(ctx, q.ch, &state.finished, promises, consume)
	return q
}

// Race resolves or rejects when an input first becomes observable as settled.
// When multiple inputs are already settled, any one may win.
// It uses the default pool. Nil contexts, empty inputs, and nil or uninitialized
// input promises panic synchronously, before observing any inputs.
func Race[T any](ctx context.Context, promises ...*Promise[T]) *Promise[T] {
	return RaceWithPool(ctx, getDefaultPool(), promises...)
}

// RaceWithPool dispatches settlement through custom pools. Dependency waits do
// not occupy pool workers. It otherwise behaves like Race; a nil pool also panics.
func RaceWithPool[T any](ctx context.Context, pool Pool, promises ...*Promise[T]) *Promise[T] {
	validateInputs(ctx, pool, promises)
	q := newPromise[T]()
	var finished atomic.Bool
	consume := func(_ int, value T, err error) {
		if finished.CompareAndSwap(false, true) {
			if err != nil {
				q.rejectWithPool(pool, err)
			} else {
				q.resolveWithPool(pool, value)
			}
		}
	}
	observeInputs(ctx, q.ch, &finished, promises, consume)
	return q
}
