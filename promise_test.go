package promise

import (
	"errors"
	"log"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		resolve(nil)
	})

	if promise == nil {
		t.Fatal("PROMISE IS NIL")
	}
}

func TestPromise_Then(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		resolve("very complicated result")
	})

	promise.Then(func(data interface{}) {
		log.Println("1", data)
	}).Then(func(data interface{}) {
		log.Println("2", data)
	}).Then(func(data interface{}) {
		log.Println("3", data)
	}).Then(func(data interface{}) {
		log.Println("4", data)
	}).Then(func(data interface{}) {
		log.Println("5", data)
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
		log.Println("1", error.Error())
	}).Catch(func(error error) {
		log.Println("2", error.Error())
	}).Catch(func(error error) {
		log.Println("3", error.Error())
	}).Catch(func(error error) {
		log.Println("4", error.Error())
	}).Catch(func(error error) {
		log.Println("5", error.Error())
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
		log.Println("Panic Recovered:", error.Error())
	})

	promise.Await()
}

func TestPromise_Await(t *testing.T) {
	var promises = make([]*Promise, 10)

	for x := 0; x < 10; x++ {
		var promise = New(func(resolve func(interface{}), reject func(error)) {
			resolve(time.Now().Second())
		}).Then(func(data interface{}) {
			log.Println(data)
		})
		promises[x] = promise
		log.Println("Added", x+1)
	}

	log.Println("Waiting")
	AwaitAll(promises...)
}
