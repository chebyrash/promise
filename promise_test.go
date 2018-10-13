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
		resolve(1 + 1)
	})

	promise.
		Then(func(data interface{}) interface{} {
			return data.(int) + 1
		}).
		Then(func(data interface{}) interface{} {
			log.Println(data)
			if data.(int) != 3 {
				t.Fatal("RESULT DOES NOT PROPAGATE")
			}
			return nil
		})

	promise.Catch(func(error error) error {
		t.Fatal("CATCH TRIGGERED IN .THEN TEST")
		return nil
	})

	promise.Await()
}

func TestPromise_Catch(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		reject(errors.New("very serious error"))
	})

	promise.
		Catch(func(error error) error {
			if error.Error() == "very serious error" {
				return errors.New("dealing with error at this stage")
			}
			return nil
		}).
		Catch(func(error error) error {
			if error.Error() != "dealing with error at this stage" {
				t.Fatal("ERROR DOES NOT PROPAGATE")
			} else {
				log.Println(error.Error())
			}
			return nil
		})

	promise.Then(func(data interface{}) interface{} {
		t.Fatal("THEN TRIGGERED IN .CATCH TEST")
		return nil
	})

	promise.Await()
}

func TestPromise_Panic(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		panic("much panic")
	})

	promise.
		Then(func(data interface{}) interface{} {
			t.Fatal("THEN TRIGGERED")
			return nil
		}).
		Catch(func(error error) error {
			log.Println("Panic Recovered:", error.Error())
			return nil
		})

	promise.Await()
}

func TestPromise_Await(t *testing.T) {
	var promises = make([]*Promise, 10)

	for x := 0; x < 10; x++ {
		var promise = New(func(resolve func(interface{}), reject func(error)) {
			resolve(time.Now())
		})

		promise.Then(func(data interface{}) interface{} {
			return data.(time.Time).Add(time.Second).Nanosecond()
		})

		promises[x] = promise
		log.Println("Added", x+1)
	}

	log.Println("Waiting")

	AwaitAll(promises...)

	for _, promise := range promises {
		promise.Then(func(data interface{}) interface{} {
			log.Println(data)
			return nil
		})
	}
}
