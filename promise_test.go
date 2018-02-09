package promise

import (
	"errors"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
	})

	if promise == nil {
		t.Fatal("PROMISE IS NIL")
	} else {
		t.Log("PROMISE INITIALISED")
	}
}

func TestPromise_Then(t *testing.T) {
	var wg = &sync.WaitGroup{}
	wg.Add(3)

	var promise = New(func(resolve func(interface{}), reject func(error)) {
		resolve("very complicated result")
	})

	promise.Then(func(data interface{}) {
		t.Log("1")
		wg.Done()
	})

	promise.Then(func(data interface{}) {
		t.Log("2")
		wg.Done()
	})

	promise.Then(func(data interface{}) {
		t.Log("3")
		wg.Done()
	})

	promise.Catch(func(error error) {
		wg.Done()
		t.Fatal("CATCH TRIGGERED")
		t.Fail()
	})

	wg.Wait()
}

func TestPromise_Catch(t *testing.T) {
	var wg = &sync.WaitGroup{}
	wg.Add(3)

	var promise = New(func(resolve func(interface{}), reject func(error)) {
		reject(errors.New("very serious error"))
	})

	promise.Then(func(data interface{}) {
		wg.Done()
		t.Fatal("THEN TRIGGERED")
		t.Fail()
	})

	promise.Catch(func(error error) {
		t.Log("1")
		wg.Done()
	})

	promise.Catch(func(error error) {
		t.Log("2")
		wg.Done()
	})

	promise.Catch(func(error error) {
		t.Log("3")
		wg.Done()
	})

	wg.Wait()
}

func TestPromise_Panic(t *testing.T) {
	var wg = &sync.WaitGroup{}
	wg.Add(1)

	var promise = New(func(resolve func(interface{}), reject func(error)) {
		panic("panicking")
	})

	promise.Then(func(data interface{}) {
		wg.Done()
		t.Fatal("THEN TRIGGERED")
		t.Fail()
	})

	promise.Catch(func(error error) {
		t.Log("Panic Recovered", error.Error())
		wg.Done()
	})

	wg.Wait()
}
