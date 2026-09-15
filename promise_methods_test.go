package promise

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/panjf2000/ants/v2"
	"github.com/stretchr/testify/require"
)

func TestGenericMethodChain(t *testing.T) {
	p := New(func(resolve func(int), _ func(error)) { resolve(42) })
	var result *Promise[bool] = p.Then(ctx, func(n int) (string, error) {
		return strconv.Itoa(n), nil
	}).Then(ctx, func(s string) ([]byte, error) {
		return []byte(s), nil
	}).Then[bool](ctx, func(b []byte) (bool, error) {
		return string(b) == "42", nil
	})
	value, err := result.Await(ctx)
	require.NoError(t, err)
	require.True(t, value)

	// Generic method values can also be explicitly instantiated.
	transform := p.Then[string]
	valueString, err := transform(ctx, func(n int) (string, error) {
		return strconv.Itoa(n), nil
	}).Await(ctx)
	require.NoError(t, err)
	require.Equal(t, "42", valueString)
}

func TestMethodErrorsAndPanics(t *testing.T) {
	for _, mode := range []string{"source rejection", "callback error", "panic error", "panic string", "panic nil"} {
		t.Run(mode, func(t *testing.T) {
			p := New(func(resolve func(int), reject func(error)) {
				if mode == "source rejection" {
					reject(errExpected)
				} else {
					resolve(42)
				}
			})
			var called atomic.Bool
			q := p.Then(ctx, func(n int) (string, error) {
				called.Store(true)
				switch mode {
				case "panic error":
					panic(errExpected)
				case "panic string":
					panic("expected error")
				case "panic nil":
					panic(nil)
				}
				return "", errExpected
			})
			value, err := q.Await(ctx)
			require.Zero(t, value)
			require.Error(t, err)
			if mode != "panic nil" {
				require.EqualError(t, err, errExpected.Error())
			}
			require.Equal(t, mode != "source rejection", called.Load())
		})
	}
}

func TestCatchMethods(t *testing.T) {
	for _, mode := range []string{"success", "recover", "zero", "replace", "panic"} {
		t.Run(mode, func(t *testing.T) {
			p := New(func(resolve func(int), reject func(error)) {
				if mode == "success" {
					resolve(42)
				} else {
					reject(errExpected)
				}
			})
			replacement := errors.New("replacement")
			var called atomic.Bool
			q := p.Catch(ctx, func(err error) (int, error) {
				called.Store(true)
				if !errors.Is(err, errExpected) {
					return 0, errors.New("wrong source error")
				}
				switch mode {
				case "recover":
					return 42, nil
				case "zero":
					return 0, nil
				case "panic":
					panic(replacement)
				default:
					return 99, replacement
				}
			})
			value, err := q.Await(ctx)
			require.Equal(t, mode != "success", called.Load())
			switch mode {
			case "success", "recover":
				require.NoError(t, err)
				require.Equal(t, 42, value)
			case "zero":
				require.NoError(t, err)
				require.Zero(t, value)
			default:
				require.ErrorIs(t, err, replacement)
				require.Zero(t, value)
			}
		})
	}
}

func TestConcurrentSettlementAndAwait(t *testing.T) {
	p := newPromise[int]()
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			if i%2 == 0 {
				p.resolve(i)
			} else {
				p.reject(errExpected)
			}
		})
	}
	wg.Wait()
	value, err := p.Await(ctx)
	for range 100 {
		wg.Go(func() {
			got, gotErr := p.Await(ctx)
			if got != value || gotErr != err {
				t.Error("settlement changed")
			}
		})
	}
	wg.Wait()
}

func TestFirstSettlementWins(t *testing.T) {
	for _, rejectFirst := range []bool{false, true} {
		p := New(func(resolve func(int), reject func(error)) {
			if rejectFirst {
				reject(errExpected)
			}
			resolve(42)
			resolve(99)
			panic("late panic")
		})
		value, err := p.Await(ctx)
		if rejectFirst {
			require.Zero(t, value)
			require.ErrorIs(t, err, errExpected)
		} else {
			require.NoError(t, err)
			require.Equal(t, 42, value)
		}
	}
}

func TestCancellationDoesNotSettleSource(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPromise[int]()
		waitCtx, cancel := context.WithCancel(ctx)
		var called atomic.Bool
		q := p.Then(waitCtx, func(int) (string, error) { called.Store(true); return "", nil })
		cancel()
		_, err := q.Await(ctx)
		require.ErrorIs(t, err, context.Canceled)
		require.False(t, called.Load())
		select {
		case <-p.ch:
			t.Fatal("cancellation settled source")
		default:
		}
		p.resolve(42)
		value, err := p.Await(ctx)
		require.NoError(t, err)
		require.Equal(t, 42, value)
	})
}

func TestCombinatorWaitersExit(t *testing.T) {
	for _, operation := range []string{"All", "Race"} {
		for _, outcome := range []string{"resolve", "reject", "cancel"} {
			t.Run(operation+"/"+outcome, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					first, second := newPromise[int](), newPromise[int]()
					waitCtx := ctx
					cancel := func() {}
					if outcome == "cancel" {
						waitCtx, cancel = context.WithCancel(ctx)
						defer cancel()
					}
					var await func(context.Context) error
					if operation == "All" {
						q := All(waitCtx, first, second)
						await = func(ctx context.Context) error { _, err := q.Await(ctx); return err }
					} else {
						q := Race(waitCtx, first, second)
						await = func(ctx context.Context) error { _, err := q.Await(ctx); return err }
					}
					synctest.Wait()
					switch outcome {
					case "resolve":
						second.resolve(2)
						if operation == "All" {
							first.resolve(1)
						}
					case "reject":
						second.reject(errExpected)
					case "cancel":
						cancel()
					}
					err := await(ctx)
					switch outcome {
					case "resolve":
						require.NoError(t, err)
					case "reject":
						require.ErrorIs(t, err, errExpected)
					case "cancel":
						require.ErrorIs(t, err, context.Canceled)
					}
					// The bubble fails if observers remain blocked on unresolved inputs.
				})
			})
		}
	}
}

func TestAllConcurrentOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		promises := make([]*Promise[int], 100)
		for i := range promises {
			promises[i] = newPromise[int]()
		}
		q := All(ctx, promises...)
		for i := len(promises) - 1; i >= 0; i-- {
			go promises[i].resolve(i)
		}
		values, err := q.Await(ctx)
		require.NoError(t, err)
		for i, value := range values {
			require.Equal(t, i, value)
		}
	})
}

func TestBoundedPoolDependencyWaits(t *testing.T) {
	for _, operation := range []string{"Then", "Catch", "All", "Race"} {
		t.Run(operation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				slot := make(chan struct{}, 1)
				var calls atomic.Int64
				pool := wrapFunc(func(f func()) {
					calls.Add(1)
					slot <- struct{}{}
					go func() { defer func() { <-slot }(); f() }()
				})
				p := newPromise[int]()
				var await func(context.Context) error
				switch operation {
				case "Then":
					q := p.ThenWithPool(ctx, func(n int) (string, error) { return strconv.Itoa(n), nil }, pool)
					await = func(ctx context.Context) error { _, err := q.Await(ctx); return err }
				case "Catch":
					q := p.CatchWithPool(ctx, func(error) (int, error) { return 42, nil }, pool)
					await = func(ctx context.Context) error { _, err := q.Await(ctx); return err }
				case "All":
					q := AllWithPool(ctx, pool, p, p)
					await = func(ctx context.Context) error { _, err := q.Await(ctx); return err }
				case "Race":
					q := RaceWithPool(ctx, pool, p, p)
					await = func(ctx context.Context) error { _, err := q.Await(ctx); return err }
				}
				synctest.Wait()
				require.Zero(t, calls.Load(), "waiting must not occupy a pool worker")
				pool.Go(func() {
					if operation == "Catch" {
						p.reject(errExpected)
					} else {
						p.resolve(42)
					}
				})
				require.NoError(t, await(ctx))
				require.EqualValues(t, 2, calls.Load())
			})
		})
	}
}

func TestInlinePool(t *testing.T) {
	pool := wrapFunc(func(f func()) { f() })
	p := NewWithPool(func(resolve func(int), _ func(error)) { resolve(42) }, pool)
	value, err := p.ThenWithPool(ctx, func(n int) (string, error) { return strconv.Itoa(n), nil }, pool).Await(ctx)
	require.NoError(t, err)
	require.Equal(t, "42", value)
	values, err := AllWithPool(ctx, pool, p, p).Await(ctx)
	require.NoError(t, err)
	require.Equal(t, []int{42, 42}, values)
	winner, err := RaceWithPool(ctx, pool, p, p).Await(ctx)
	require.NoError(t, err)
	require.Equal(t, 42, winner)
}

func TestClosedAntsPoolRejects(t *testing.T) {
	pool, err := ants.NewPool(1)
	require.NoError(t, err)
	pool.Release()
	p := NewWithPool(func(resolve func(int), _ func(error)) { resolve(42) }, FromAntsPool(pool))
	value, err := p.Await(ctx)
	require.Zero(t, value)
	require.ErrorIs(t, err, ants.ErrPoolClosed)
}

func TestDefaultPoolConcurrentReplacement(t *testing.T) {
	original := getDefaultPool()
	t.Cleanup(func() { SetDefaultPool(original) })
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			SetDefaultPool(newDefaultPool())
			p := New(func(resolve func(int), _ func(error)) { resolve(42) })
			_, err := p.Await(ctx)
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
}

func TestNilAndZeroValues(t *testing.T) {
	p := New(func(resolve func(*int), _ func(error)) { resolve(nil) })
	value, err := p.Await(ctx)
	require.NoError(t, err)
	require.Nil(t, value)
	isNil, err := p.Then(ctx, func(n *int) (bool, error) { return n == nil, nil }).Await(ctx)
	require.NoError(t, err)
	require.True(t, isNil)

	rejected := New(func(_ func(int), reject func(error)) { reject(nil) })
	zero, err := rejected.Await(ctx)
	require.ErrorIs(t, err, ErrNilRejection)
	require.Zero(t, zero)
	recovered, err := rejected.Catch(ctx, func(err error) (int, error) { return 42, nil }).Await(ctx)
	require.NoError(t, err)
	require.Equal(t, 42, recovered)
	zero, err = Race(ctx, rejected).Await(ctx)
	require.ErrorIs(t, err, ErrNilRejection)
	require.Zero(t, zero)
}

func TestInvalidArguments(t *testing.T) {
	p := newPromise[int]()
	require.Panics(t, func() { New[int](nil) })
	require.Panics(t, func() { NewWithPool(func(func(int), func(error)) {}, nil) })
	require.Panics(t, func() { p.ThenWithPool(ctx, func(n int) (int, error) { return n, nil }, nil) })
	require.Panics(t, func() { p.CatchWithPool(ctx, func(err error) (int, error) { return 0, err }, nil) })
	require.Panics(t, func() { All[int](ctx) })
	require.Panics(t, func() { Race[int](ctx) })
	require.Panics(t, func() { AllWithPool(ctx, nil, p) })
	require.Panics(t, func() { RaceWithPool(ctx, nil, p) })
	require.Panics(t, func() { SetDefaultPool(nil) })
}

func TestCombinatorsCapturePendingInputs(t *testing.T) {
	for _, operation := range []string{"All", "Race"} {
		t.Run(operation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				first, second, third := newPromise[int](), newPromise[int](), newPromise[int]()
				inputs := []*Promise[int]{first, second, third}
				if operation == "All" {
					first.resolve(1) // Exercise an already-consumed prefix before the pending suffix.
					q := All(ctx, inputs...)
					clear(inputs) // The caller may reuse its slice as soon as All returns.
					third.resolve(3)
					second.resolve(2)
					values, err := q.Await(ctx)
					require.NoError(t, err)
					require.Equal(t, []int{1, 2, 3}, values)
				} else {
					q := Race(ctx, inputs...)
					clear(inputs)
					second.resolve(2)
					value, err := q.Await(ctx)
					require.NoError(t, err)
					require.Equal(t, 2, value)
				}
			})
		})
	}
}

func TestDefaultCallbacksRemainAsynchronous(t *testing.T) {
	for _, operation := range []string{"Then", "Catch"} {
		t.Run(operation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				gate := make(chan struct{})
				p := newPromise[int]()
				var q *Promise[int]
				if operation == "Then" {
					p.resolve(42)
					q = p.Then(ctx, func(n int) (int, error) { <-gate; return n, nil })
				} else {
					p.reject(errExpected)
					q = p.Catch(ctx, func(err error) (int, error) { <-gate; return 0, err })
				}
				close(gate) // This must be reachable while the callback is waiting.
				value, err := q.Await(ctx)
				if operation == "Then" {
					require.NoError(t, err)
					require.Equal(t, 42, value)
				} else {
					require.Zero(t, value)
					require.ErrorIs(t, err, errExpected)
				}
			})
		})
	}
}

func TestCustomPoolControlsAggregateSettlement(t *testing.T) {
	for _, operation := range []string{"All", "Race"} {
		for _, rejected := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/rejected=%t", operation, rejected), func(t *testing.T) {
				p := newPromise[int]()
				if rejected {
					p.reject(errExpected)
				} else {
					p.resolve(42)
				}
				tasks := make(chan func(), 1)
				pool := wrapFunc(func(f func()) { tasks <- f })
				waitCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				var done <-chan struct{}
				var await func() error
				if operation == "All" {
					q := AllWithPool(waitCtx, pool, p)
					done = q.ch
					await = func() error {
						value, err := q.Await(ctx)
						if !rejected {
							require.Equal(t, []int{42}, value)
						}
						return err
					}
				} else {
					q := RaceWithPool(waitCtx, pool, p)
					done = q.ch
					await = func() error {
						value, err := q.Await(ctx)
						if !rejected {
							require.Equal(t, 42, value)
						}
						return err
					}
				}
				select {
				case <-done:
					t.Fatal("settled before pool ran its task")
				default:
				}
				cancel() // Once an outcome is selected, a queued task must publish that outcome.
				(<-tasks)()
				if rejected {
					require.ErrorIs(t, await(), errExpected)
				} else {
					require.NoError(t, await())
				}
			})
		}
	}
}

func TestAwaitWithCancelableContext(t *testing.T) {
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	p := newPromise[int]()
	p.resolve(42)
	value, err := p.Await(waitCtx)
	require.NoError(t, err)
	require.Equal(t, 42, value)
	pending := newPromise[int]()
	cancel()
	value, err = pending.Await(waitCtx)
	require.Zero(t, value)
	require.ErrorIs(t, err, context.Canceled)
	select {
	case <-pending.ch:
		t.Fatal("Await settled its source")
	default:
	}
}
