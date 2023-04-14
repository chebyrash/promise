package promise

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var errExpected = errors.New("expected error")
var ctx = context.Background()

func TestNew(t *testing.T) {
	p := New(func(resolve func(any), reject func(error)) {
		resolve(nil)
	})
	require.NotNil(t, p)
}

func TestPromise_Then(t *testing.T) {
	p1 := New(func(resolve func(string), reject func(error)) {
		resolve("Hello, ")
	})
	p2 := Then(p1, ctx, func(data string) string {
		return data + "world!"
	})

	val, err := p1.Await(ctx)
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, "Hello, ", *val)

	val, err = p2.Await(ctx)
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, "Hello, world!", *val)
}

func TestPromise_Catch(t *testing.T) {
	p1 := New(func(resolve func(any), reject func(error)) {
		reject(errExpected)
	})
	p2 := Then(p1, ctx, func(data any) any {
		t.Fatal("should not execute Then")
		return nil
	})

	val, err := p1.Await(ctx)
	require.Error(t, err)
	require.Equal(t, errExpected, err)
	require.Nil(t, val)

	p2.Await(ctx)
}

func TestPromise_Panic(t *testing.T) {
	p1 := New(func(resolve func(any), reject func(error)) {
		panic("random error")
	})
	p2 := New(func(resolve func(any), reject func(error)) {
		panic(errExpected)
	})

	val, err := p1.Await(ctx)
	require.Error(t, err)
	require.Equal(t, errors.New("random error"), err)
	require.Nil(t, val)

	val, err = p2.Await(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, errExpected)
	require.Nil(t, val)
}

func TestAll_Happy(t *testing.T) {
	p1 := New(func(resolve func(string), reject func(error)) {
		resolve("one")
	})
	p2 := New(func(resolve func(string), reject func(error)) {
		resolve("two")
	})
	p3 := New(func(resolve func(string), reject func(error)) {
		resolve("three")
	})

	p := All(ctx, p1, p2, p3)

	val, err := p.Await(ctx)
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, []string{"one", "two", "three"}, *val)
}

func TestAll_ContainsRejected(t *testing.T) {
	p1 := New(func(resolve func(string), reject func(error)) {
		resolve("one")
	})
	p2 := New(func(resolve func(string), reject func(error)) {
		reject(errExpected)
	})
	p3 := New(func(resolve func(string), reject func(error)) {
		resolve("three")
	})

	p := All(ctx, p1, p2, p3)

	val, err := p.Await(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, errExpected)
	require.Nil(t, val)
}

func TestAll_OnlyRejected(t *testing.T) {
	p1 := New(func(resolve func(any), reject func(error)) {
		reject(errExpected)
	})
	p2 := New(func(resolve func(any), reject func(error)) {
		reject(errExpected)
	})
	p3 := New(func(resolve func(any), reject func(error)) {
		reject(errExpected)
	})

	p := All(ctx, p1, p2, p3)

	val, err := p.Await(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, errExpected)
	require.Nil(t, val)
}

func TestRace_Happy(t *testing.T) {
	p1 := New(func(resolve func(string), reject func(error)) {
		time.Sleep(time.Millisecond * 100)
		resolve("faster")
	})
	p2 := New(func(resolve func(string), reject func(error)) {
		time.Sleep(time.Millisecond * 500)
		resolve("slower")
	})

	p := Race(ctx, p1, p2)

	val, err := p.Await(ctx)
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, "faster", *val)
}

func TestRace_ContainsRejected(t *testing.T) {
	p1 := New(func(resolve func(any), reject func(error)) {
		time.Sleep(time.Millisecond * 100)
		resolve(nil)
	})
	p2 := New(func(resolve func(any), reject func(error)) {
		reject(errExpected)
	})

	p := Race(ctx, p1, p2)

	val, err := p.Await(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, errExpected)
	require.Nil(t, val)
}

func TestRace_OnlyRejected(t *testing.T) {
	p1 := New(func(resolve func(any), reject func(error)) {
		reject(errExpected)
	})
	p2 := New(func(resolve func(any), reject func(error)) {
		reject(errExpected)
	})

	p := Race(ctx, p1, p2)

	val, err := p.Await(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, errExpected)
	require.Nil(t, val)
}
