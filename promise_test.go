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
		t.Log("1", data)
		wg.Done()
	}).Then(func(data interface{}) {
		t.Log("2", data)
		wg.Done()
	}).Then(func(data interface{}) {
		t.Log("3", data)
		wg.Done()
	}).Catch(func(error error) {
		wg.Done()
		t.Fatal("CATCH TRIGGERED")
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
	}).Catch(func(error error) {
		t.Log("1", error.Error())
		wg.Done()
	}).Catch(func(error error) {
		t.Log("2", error.Error())
		wg.Done()
	}).Catch(func(error error) {
		t.Log("3", error.Error())
		wg.Done()
	})

	wg.Wait()
}

func TestPromise_Panic(t *testing.T) {
	var wg = &sync.WaitGroup{}
	wg.Add(1)

	var promise = New(func(resolve func(interface{}), reject func(error)) {
		panic("much panic")
	})

	promise.Then(func(data interface{}) {
		wg.Done()
		t.Fatal("THEN TRIGGERED")
	}).Catch(func(error error) {
		t.Log("Panic Recovered:", error.Error())
		wg.Done()
	})

	wg.Wait()
}
