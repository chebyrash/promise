package promise

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var expectedError = errors.New("very serious error")

func TestNew(t *testing.T) {
	p := New(func(resolve func(int), reject func(error)) {
		resolve(0)
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
	p1 := New(func(resolve func(string), reject func(error)) {
		reject(expectedError)
	})
	p2 := Then(p1, func(data string) *string {
		t.Fatal("should not execute Then")
		return nil
	})

	_, err := p1.Await()
	require.Error(t, err)
	require.Equal(t, expectedError, err)

	p2.Await()
}

func TestPromise_Panic(t *testing.T) {
	p1 := New(func(resolve func(string), reject func(error)) {
		panic(nil)
	})
	p2 := New(func(resolve func(string), reject func(error)) {
		panic("random stringy error")
	})
	p3 := New(func(resolve func(string), reject func(error)) {
		panic(expectedError)
	})

	_, err := p1.Await()
	require.Error(t, err)
	require.Equal(t, errors.New("panic recovery: <nil>"), err)

	_, err = p2.Await()
	require.Error(t, err)
	require.Equal(t, errors.New("panic recovery: random stringy error"), err)

	_, err = p3.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, expectedError)
}
