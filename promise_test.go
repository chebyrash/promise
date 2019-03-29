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
		t.Error("PROMISE IS NIL")
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
				t.Error("RESULT DOES NOT PROPAGATE")
			}
			return nil
		}).
		Catch(func(error error) error {
			t.Error("CATCH TRIGGERED IN .THEN TEST")
			return nil
		})

	promise.Await()
}

func TestPromise_Then2(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		resolve(New(func(res func(interface{}), rej func(error)) {
			res("Hello, World")
		}))
	})

	promise.
		Then(func(data interface{}) interface{} {
			if data.(string) != "Hello, World" {
				t.Error("PROMISE DOESN'T FLATTEN")
			}
			t.Log("PROMISE FLATTENS ON NESTED RESOLVE")
			return nil
		}).
		Catch(func(error error) error {
			t.Error("CATCH TRIGGERED IN .THEN TEST")
			return nil
		})

	promise.Await()
}

func TestPromise_Then3(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		resolve(New(func(res func(interface{}), rej func(error)) {
			rej(errors.New("nested fail"))
		}))
	})

	promise.
		Then(func(data interface{}) interface{} {
			t.Error("THEN TRIGGERED IN .CATCH TEST")
			return nil
		}).
		Catch(func(error error) error {
			log.Println("PROMISE FLATTENS ON NESTED REJECTION", error.Error())
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
				t.Error("ERROR DOES NOT PROPAGATE")
			} else {
				log.Println(error.Error())
			}
			return nil
		})

	promise.Then(func(data interface{}) interface{} {
		t.Error("THEN TRIGGERED IN .CATCH TEST")
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
			t.Error("THEN TRIGGERED IN .CATCH TEST")
			return nil
		}).
		Catch(func(error error) error {
			log.Println("Panic Recovered: ", error.Error())
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

	var promise1 = New(func(resolve func(interface{}), reject func(error)) {
		resolve("WinRAR")
	})

	var promise2 = New(func(resolve func(interface{}), reject func(error)) {
		reject(errors.New("fail"))
	})

	AwaitAll(promises...)
	AwaitAll(promise1, promise2)

	for _, promise := range promises {
		promise.Then(func(data interface{}) interface{} {
			log.Println(data)
			return nil
		})
	}
}

func TestPromise_Resolve(t *testing.T) {
	var promise = Resolve(123).
		Then(func(data interface{}) interface{} {
			return data.(int) + 1
		}).
		Then(func(data interface{}) interface{} {
			t.Helper()
			if data.(int) != 124 {
				t.Errorf("THEN RESOLVED WITH UNEXPECTED VALUE: %v", data.(int))
			}
			return nil
		})

	promise.Await()
}

func TestPromise_Reject(t *testing.T) {
	var promise = Reject(errors.New("Rejected")).
		Then(func(data interface{}) interface{} {
			return data.(int) + 1
		}).
		Catch(func(error error) error {
			if error.Error() != "Rejected" {
				t.Errorf("CATCH REJECTED WITH UNEXPECTED VALUE: %v", error)
			}
			return nil
		})

	promise.Await()
}
