package promise_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/chebyrash/promise"
	"github.com/panjf2000/ants/v2"
	"github.com/stretchr/testify/require"
)

type testPool func(func())

func (p testPool) Go(f func()) { p(f) }

func resolved[T any](value T) *promise.Promise[T] {
	return promise.NewWithPool(func(resolve func(T), _ func(error)) { resolve(value) }, testPool(func(f func()) { f() }))
}
func rejected[T any](err error) *promise.Promise[T] {
	return promise.NewWithPool(func(_ func(T), reject func(error)) { reject(err) }, testPool(func(f func()) { f() }))
}

func TestPublicSignatures(t *testing.T) {
	ctx := context.Background()
	p := resolved(42)
	var await func(context.Context) (int, error) = p.Await
	var then func(context.Context, func(int) (string, error)) *promise.Promise[string] = p.Then[string]
	var thenPool func(context.Context, func(int) (string, error), promise.Pool) *promise.Promise[string] = p.ThenWithPool[string]
	var catch func(context.Context, func(error) (int, error)) *promise.Promise[int] = p.Catch
	var catchPool func(context.Context, func(error) (int, error), promise.Pool) *promise.Promise[int] = p.CatchWithPool
	value, err := await(ctx)
	require.NoError(t, err)
	require.Equal(t, 42, value)
	transform := func(n int) (string, error) { return strconv.Itoa(n), nil }
	pool := testPool(func(f func()) { f() })
	for _, q := range []*promise.Promise[string]{then(ctx, transform), thenPool(ctx, transform, pool)} {
		text, err := q.Await(ctx)
		require.NoError(t, err)
		require.Equal(t, "42", text)
	}
	recoverValue := func(error) (int, error) { t.Error("recovery ran on success"); return 0, nil }
	for _, q := range []*promise.Promise[int]{catch(ctx, recoverValue), catchPool(ctx, recoverValue, pool)} {
		value, err := q.Await(ctx)
		require.NoError(t, err)
		require.Equal(t, 42, value)
	}
	// Function forms that construct promises and aggregates also preserve T.
	var all func(context.Context, ...*promise.Promise[int]) *promise.Promise[[]int] = promise.All[int]
	var race func(context.Context, ...*promise.Promise[int]) *promise.Promise[int] = promise.Race[int]
	values, err := all(ctx, p).Await(ctx)
	require.NoError(t, err)
	require.Equal(t, []int{42}, values)
	value, err = race(ctx, p).Await(ctx)
	require.NoError(t, err)
	require.Equal(t, 42, value)
}

func TestPublicErrorPropagation(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("cause")
	wrapped := fmt.Errorf("operation: %w", sentinel)
	p := rejected[int](wrapped)
	for _, pool := range []promise.Pool{testPool(func(f func()) { f() }), testPool(func(f func()) { go f() })} {
		var called atomic.Bool
		q := p.ThenWithPool(ctx, func(int) (string, error) { called.Store(true); return "unexpected", nil }, pool)
		value, err := q.Await(ctx)
		require.Empty(t, value)
		require.Same(t, wrapped, err)
		require.ErrorIs(t, err, sentinel)
		require.False(t, called.Load())
		mapped := resolved(1).ThenWithPool(ctx, func(int) (int, error) { return 99, wrapped }, pool)
		recovered := p.CatchWithPool(ctx, func(error) (int, error) { return 99, wrapped }, pool)
		for _, q := range []*promise.Promise[int]{p, mapped, recovered, promise.RaceWithPool(ctx, pool, p)} {
			value, err := q.Await(ctx)
			require.Zero(t, value)
			require.Same(t, wrapped, err)
			require.ErrorIs(t, err, sentinel)
		}
		values, err := promise.AllWithPool(ctx, pool, p, resolved(2)).Await(ctx)
		require.Nil(t, values)
		require.Same(t, wrapped, err)
	}
}

func TestRecoveredValuesCompose(t *testing.T) {
	ctx := context.Background()
	for _, recovery := range []int{0, 42} {
		q := rejected[int](errors.New("source")).Catch(ctx, func(error) (int, error) { return recovery, nil })
		value, err := q.Then(ctx, func(n int) (string, error) { return strconv.Itoa(n), nil }).Await(ctx)
		require.NoError(t, err)
		require.Equal(t, strconv.Itoa(recovery), value)
		values, err := promise.All(ctx, q, resolved(7)).Await(ctx)
		require.NoError(t, err)
		require.Equal(t, []int{recovery, 7}, values)
		winner, err := promise.Race(ctx, q).Await(ctx)
		require.NoError(t, err)
		require.Equal(t, recovery, winner)
	}
}

func TestSuccessfulNilAndErrorValues(t *testing.T) {
	ctx := context.Background()
	p := resolved[any](nil)
	value, err := p.Await(ctx)
	require.Nil(t, value)
	require.NoError(t, err)
	mapped, err := p.Then(ctx, func(value any) (bool, error) { return value == nil, nil }).Await(ctx)
	require.True(t, mapped)
	require.NoError(t, err)
	values, err := promise.All(ctx, p).Await(ctx)
	require.NoError(t, err)
	require.Equal(t, []any{nil}, values)
	value, err = promise.Race(ctx, p).Await(ctx)
	require.Nil(t, value)
	require.NoError(t, err)
	recovered, err := rejected[*int](errors.New("source")).Catch(ctx, func(error) (*int, error) { return nil, nil }).Await(ctx)
	require.Nil(t, recovered)
	require.NoError(t, err)
	errorValue := errors.New("this is a value")
	got, err := resolved[error](errorValue).Await(ctx)
	require.Same(t, errorValue, got)
	require.NoError(t, err)
}

func TestNilRejectionPropagation(t *testing.T) {
	ctx := context.Background()
	p := rejected[int](nil)
	q := p.Then(ctx, func(n int) (int, error) { t.Error("transform ran"); return n, nil })
	for _, q := range []*promise.Promise[int]{p, q, promise.Race(ctx, p)} {
		value, err := q.Await(ctx)
		require.Zero(t, value)
		require.ErrorIs(t, err, promise.ErrNilRejection)
	}
	values, err := promise.All(ctx, p).Await(ctx)
	require.Nil(t, values)
	require.ErrorIs(t, err, promise.ErrNilRejection)
	value, err := p.Catch(ctx, func(err error) (int, error) {
		if !errors.Is(err, promise.ErrNilRejection) {
			return 0, err
		}
		return 42, nil
	}).Await(ctx)
	require.NoError(t, err)
	require.Equal(t, 42, value)
}

func TestPublicPanicRecovery(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("panic error")
	for _, panicValue := range []any{sentinel, "panic string", 123, nil} {
		for _, operation := range []string{"New", "Then", "Catch"} {
			t.Run(fmt.Sprintf("%s/%v", operation, panicValue), func(t *testing.T) {
				var p *promise.Promise[int]
				switch operation {
				case "New":
					p = promise.New(func(func(int), func(error)) { panic(panicValue) })
				case "Then":
					p = resolved(1).Then(ctx, func(int) (int, error) { panic(panicValue) })
				case "Catch":
					p = rejected[int](sentinel).Catch(ctx, func(error) (int, error) { panic(panicValue) })
				}
				value, err := p.Await(ctx)
				require.Zero(t, value)
				require.Error(t, err)
				switch panicValue {
				case sentinel:
					require.Same(t, sentinel, err)
				case nil:
					var nilPanic *runtime.PanicNilError
					require.ErrorAs(t, err, &nilPanic)
				default:
					require.EqualError(t, err, fmt.Sprint(panicValue))
				}
			})
		}
	}
}

func TestOperationCancellationSkipsCallbacks(t *testing.T) {
	for _, operation := range []string{"Then", "Catch", "All", "Race"} {
		t.Run(operation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				p := promise.New(func(func(int), func(error)) {})
				var called atomic.Bool
				var await func() error
				switch operation {
				case "Then":
					q := p.Then(ctx, func(n int) (int, error) { called.Store(true); return n, nil })
					await = func() error { _, err := q.Await(context.Background()); return err }
				case "Catch":
					q := p.Catch(ctx, func(error) (int, error) { called.Store(true); return 42, nil })
					await = func() error { _, err := q.Await(context.Background()); return err }
				case "All":
					q := promise.All(ctx, p)
					await = func() error { _, err := q.Await(context.Background()); return err }
				case "Race":
					q := promise.Race(ctx, p)
					await = func() error { _, err := q.Await(context.Background()); return err }
				}
				require.ErrorIs(t, await(), context.DeadlineExceeded)
				require.False(t, called.Load())
			})
		})
	}
}

func TestSourceContextErrorsAreRecoverable(t *testing.T) {
	for _, sourceErr := range []error{context.Canceled, context.DeadlineExceeded} {
		value, err := rejected[int](sourceErr).Catch(context.Background(), func(err error) (int, error) {
			if err != sourceErr {
				return 0, err
			}
			return 42, nil
		}).Await(context.Background())
		require.NoError(t, err)
		require.Equal(t, 42, value)
	}
}

func TestQueuedCallbacksHonorCancellation(t *testing.T) {
	for _, operation := range []string{"Then", "Catch"} {
		tasks := make(chan func(), 1)
		pool := testPool(func(f func()) { tasks <- f })
		ctx, cancel := context.WithCancel(context.Background())
		var called bool
		var q *promise.Promise[int]
		if operation == "Then" {
			q = resolved(42).ThenWithPool(ctx, func(n int) (int, error) { called = true; return n, nil }, pool)
		} else {
			q = rejected[int](errors.New("source")).CatchWithPool(ctx, func(error) (int, error) { called = true; return 42, nil }, pool)
		}
		cancel()
		(<-tasks)()
		value, err := q.Await(context.Background())
		require.Zero(t, value)
		require.ErrorIs(t, err, context.Canceled)
		require.False(t, called)
	}
}

func TestAlreadyCanceledContextWins(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := resolved(42)
	value, err := p.Await(ctx)
	require.Zero(t, value)
	require.ErrorIs(t, err, context.Canceled)
	for _, q := range []*promise.Promise[int]{
		p.Then(ctx, func(int) (int, error) { t.Error("transform ran"); return 0, nil }),
		p.Catch(ctx, func(error) (int, error) { t.Error("recovery ran"); return 0, nil }),
		promise.Race(ctx, p),
	} {
		value, err := q.Await(context.Background())
		require.Zero(t, value)
		require.ErrorIs(t, err, context.Canceled)
	}
	values, err := promise.All(ctx, p).Await(context.Background())
	require.Nil(t, values)
	require.ErrorIs(t, err, context.Canceled)
}

func TestSubmissionErrorsReachEveryAPI(t *testing.T) {
	sentinel := errors.New("submission failure")
	antsPool, err := ants.NewPool(1)
	require.NoError(t, err)
	antsPool.Release()
	for _, pool := range []promise.Pool{testPool(func(func()) { panic(sentinel) }), promise.FromAntsPool(antsPool)} {
		ctx := context.Background()
		expected := sentinel
		if _, custom := pool.(testPool); !custom {
			expected = ants.ErrPoolClosed
		}
		var called atomic.Bool
		for _, p := range []*promise.Promise[int]{
			promise.NewWithPool(func(func(int), func(error)) { called.Store(true) }, pool),
			resolved(42).ThenWithPool(ctx, func(n int) (int, error) { called.Store(true); return n, nil }, pool),
			rejected[int](errors.New("source")).CatchWithPool(ctx, func(error) (int, error) { called.Store(true); return 42, nil }, pool),
			promise.RaceWithPool(ctx, pool, resolved(42)),
		} {
			value, err := p.Await(ctx)
			require.Zero(t, value)
			require.ErrorIs(t, err, expected)
		}
		values, err := promise.AllWithPool(ctx, pool, resolved(42)).Await(ctx)
		require.Nil(t, values)
		require.ErrorIs(t, err, expected)
		require.False(t, called.Load())
	}
}

func TestInvalidPublicArgumentsPanicBeforeWork(t *testing.T) {
	ctx := context.Background()
	var calls int
	pool := testPool(func(f func()) { calls++; f() })
	p := resolved(42)
	var nilPromise *promise.Promise[int]
	var zeroPromise promise.Promise[int]
	transform := func(n int) (string, error) { return strconv.Itoa(n), nil }
	recoverValue := func(error) (int, error) { return 42, nil }
	for _, invalid := range []*promise.Promise[int]{nilPromise, &zeroPromise} {
		require.Panics(t, func() { invalid.Await(ctx) })
		require.Panics(t, func() { invalid.ThenWithPool(ctx, transform, pool) })
		require.Panics(t, func() { invalid.CatchWithPool(ctx, recoverValue, pool) })
		require.Panics(t, func() { promise.AllWithPool(ctx, pool, p, invalid) })
		require.Panics(t, func() { promise.RaceWithPool(ctx, pool, p, invalid) })
		pending := promise.New(func(func(int), func(error)) {})
		require.Panics(t, func() { promise.AllWithPool(ctx, pool, pending, invalid) })
		require.Panics(t, func() { promise.RaceWithPool(ctx, pool, pending, invalid) })
	}
	require.Panics(t, func() { p.Await(nil) })
	require.Panics(t, func() { p.ThenWithPool(nil, transform, pool) })
	require.Panics(t, func() { p.CatchWithPool(nil, recoverValue, pool) })
	require.Panics(t, func() { promise.AllWithPool(nil, pool, p) })
	require.Panics(t, func() { promise.RaceWithPool(nil, pool, p) })
	require.Panics(t, func() { p.ThenWithPool[string](ctx, nil, pool) })
	require.Panics(t, func() { p.CatchWithPool(ctx, nil, pool) })
	require.Panics(t, func() { promise.FromConcPool(nil) })
	require.Panics(t, func() { promise.FromAntsPool(nil) })
	require.Zero(t, calls)
}

func TestAwaitDoesNotExposeStoredScalar(t *testing.T) {
	type result struct{ Number int }
	p := resolved(result{Number: 42})
	value, err := p.Await(context.Background())
	require.NoError(t, err)
	value.Number = 99
	again, err := p.Await(context.Background())
	require.NoError(t, err)
	require.Equal(t, 42, again.Number)
}

type typedError struct{}

func (*typedError) Error() string { return "typed error" }

func TestTypedNilErrorRemainsAnError(t *testing.T) {
	var typedNil *typedError
	var sourceErr error = typedNil
	ctx := context.Background()
	p := rejected[int](sourceErr)
	value, err := p.Await(ctx)
	require.Zero(t, value)
	require.True(t, err != nil)
	require.True(t, err == sourceErr)
	var target *typedError
	require.ErrorAs(t, err, &target)
	q := p.Catch(ctx, func(err error) (int, error) {
		if err != sourceErr {
			return 0, errors.New("error identity changed")
		}
		return 42, nil
	})
	value, err = q.Await(ctx)
	require.NoError(t, err)
	require.Equal(t, 42, value)
}
