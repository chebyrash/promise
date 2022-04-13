package promise

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var expectedError = errors.New("expected error")

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
	p2 := Then(p1, func(data string) string {
		return data + "world!"
	})

	val, err := p1.Await()
	require.NoError(t, err)
	require.Equal(t, "Hello, ", val)

	val, err = p2.Await()
	require.NoError(t, err)
	require.Equal(t, val, "Hello, world!")
}

func TestPromise_Catch(t *testing.T) {
	p1 := New(func(resolve func(any), reject func(error)) {
		reject(expectedError)
	})
	p2 := Then(p1, func(data any) any {
		t.Fatal("should not execute Then")
		return nil
	})

	val, err := p1.Await()
	require.Error(t, err)
	require.Equal(t, expectedError, err)
	require.Nil(t, val)

	p2.Await()
}

func TestPromise_Panic(t *testing.T) {
	p1 := New(func(resolve func(any), reject func(error)) {
		panic(nil)
	})
	p2 := New(func(resolve func(any), reject func(error)) {
		panic("random error")
	})
	p3 := New(func(resolve func(any), reject func(error)) {
		panic(expectedError)
	})

	val, err := p1.Await()
	require.Error(t, err)
	require.Equal(t, errors.New("panic recovery: <nil>"), err)
	require.Nil(t, val)

	val, err = p2.Await()
	require.Error(t, err)
	require.Equal(t, errors.New("panic recovery: random error"), err)
	require.Nil(t, val)

	val, err = p3.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, expectedError)
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

	p := All(p1, p2, p3)

	val, err := p.Await()
	require.NoError(t, err)
	require.Equal(t, []string{"one", "two", "three"}, val)
}

func TestAll_Empty(t *testing.T) {
	var empty []*Promise[any]
	p := All(empty...)
	require.Nil(t, p)
}

func TestAll_ContainsRejected(t *testing.T) {
	p1 := New(func(resolve func(string), reject func(error)) {
		resolve("one")
	})
	p2 := New(func(resolve func(string), reject func(error)) {
		reject(expectedError)
	})
	p3 := New(func(resolve func(string), reject func(error)) {
		resolve("three")
	})

	p := All(p1, p2, p3)

	val, err := p.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, expectedError)
	require.Nil(t, val)
}

func TestAll_OnlyRejected(t *testing.T) {
	p1 := New(func(resolve func(any), reject func(error)) {
		reject(expectedError)
	})
	p2 := New(func(resolve func(any), reject func(error)) {
		reject(expectedError)
	})
	p3 := New(func(resolve func(any), reject func(error)) {
		reject(expectedError)
	})

	p := All(p1, p2, p3)

	val, err := p.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, expectedError)
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

	p := Race(p1, p2)

	val, err := p.Await()
	require.NoError(t, err)
	require.Equal(t, "faster", val)
}

func TestRace_Empty(t *testing.T) {
	var empty []*Promise[any]
	p := Race(empty...)
	require.Nil(t, p)
}

func TestRace_ContainsRejected(t *testing.T) {
	p1 := New(func(resolve func(any), reject func(error)) {
		time.Sleep(time.Millisecond * 100)
		resolve(nil)
	})
	p2 := New(func(resolve func(any), reject func(error)) {
		reject(expectedError)
	})

	p := Race(p1, p2)

	val, err := p.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, expectedError)
	require.Nil(t, val)
}

func TestRace_OnlyRejected(t *testing.T) {
	p1 := New(func(resolve func(any), reject func(error)) {
		reject(expectedError)
	})
	p2 := New(func(resolve func(any), reject func(error)) {
		reject(expectedError)
	})

	p := Race(p1, p2)

	val, err := p.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, expectedError)
	require.Nil(t, val)
}

func TestAny_Happy(t *testing.T) {
	p1 := New(func(resolve func(string), reject func(error)) {
		time.Sleep(time.Millisecond * 250)
		resolve("faster")
	})
	p2 := New(func(resolve func(string), reject func(error)) {
		time.Sleep(time.Millisecond * 500)
		resolve("slower")
	})
	p3 := New(func(resolve func(string), reject func(error)) {
		reject(expectedError)
	})

	p := Any(p3, p2, p1)

	val, err := p.Await()
	require.NoError(t, err)
	require.Equal(t, "faster", val)
}

func TestAny_Empty(t *testing.T) {
	var empty []*Promise[any]
	p := Any(empty...)
	require.Nil(t, p)
}

func TestAny_OnlyRejected(t *testing.T) {
	p1 := New(func(resolve func(any), reject func(error)) {
		reject(expectedError)
	})
	p2 := New(func(resolve func(any), reject func(error)) {
		reject(expectedError)
	})

	p := Any(p1, p2)

	val, err := p.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, expectedError)
	require.Nil(t, val)
}
