package promise

import (
	"errors"
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
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		resolve("very complicated result")
	})

	promise.Then(func(data interface{}) {
		t.Log("1", data)
	}).Then(func(data interface{}) {
		t.Log("2", data)
	}).Then(func(data interface{}) {
		t.Log("3", data)
	}).Then(func(data interface{}) {
		t.Log("4", data)
	}).Then(func(data interface{}) {
		t.Log("5", data)
	})

	promise.Catch(func(error error) {
		t.Fatal("CATCH TRIGGERED IN .THEN TEST")
	})

	promise.Await()
}

func TestPromise_Catch(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		reject(errors.New("very serious error"))
	})

	promise.Catch(func(error error) {
		t.Log("1", error.Error())
	}).Catch(func(error error) {
		t.Log("2", error.Error())
	}).Catch(func(error error) {
		t.Log("3", error.Error())
	}).Catch(func(error error) {
		t.Log("4", error.Error())
	}).Catch(func(error error) {
		t.Log("5", error.Error())
	})

	promise.Then(func(data interface{}) {
		t.Fatal("THEN TRIGGERED IN .CATCH TEST")
	})

	promise.Await()
}

func TestPromise_Panic(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		panic("much panic")
	})

	promise.Then(func(data interface{}) {
		t.Fatal("THEN TRIGGERED")
	}).Catch(func(error error) {
		t.Log("Panic Recovered:", error.Error())
	})

	promise.Await()
}
