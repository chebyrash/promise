package promise

import (
	"errors"
	"fmt"
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
			if data.(int) != 3 {
				t.Error("RESULT DOES NOT PROPAGATE")
			}
			return nil
		}).
		Catch(func(err error) error {
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
			return nil
		}).
		Catch(func(err error) error {
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
		})

	promise.Await()
}

func TestPromise_Catch(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		reject(errors.New("very serious err"))
	})

	promise.
		Catch(func(err error) error {
			if err.Error() == "very serious err" {
				return errors.New("dealing with err at this stage")
			}
			return nil
		}).
		Catch(func(err error) error {
			if err.Error() != "dealing with err at this stage" {
				t.Error("ERROR DOES NOT PROPAGATE")
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
	}

	var promise1 = Resolve("WinRAR")
	var promise2 = Reject(errors.New("fail"))

	for _, p := range promises {
		_, err := p.Await()

		if err != nil {
			t.Error(err)
		}
	}

	result, err := promise1.Await()
	if err != nil && result != "WinRAR" {
		t.Error(err)
	}

	result, err = promise2.Await()
	if err == nil {
		t.Error(err)
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
	var promise = Reject(errors.New("rejected")).
		Then(func(data interface{}) interface{} {
			return data.(int) + 1
		}).
		Catch(func(err error) error {
			if err.Error() != "rejected" {
				t.Errorf("CATCH REJECTED WITH UNEXPECTED VALUE: %v", err)
			}
			return nil
		})

	promise.Await()
}

func TestPromise_All(t *testing.T) {
	var promises = make([]*Promise, 10)

	for x := 0; x < 10; x++ {
		if x == 8 {
			promises[x] = Reject(errors.New("bad promise"))
			continue
		}

		promises[x] = Resolve("All Good")
	}

	combined := All(promises...)
	_, err := combined.Await()
	if err == nil {
		t.Error("Combined promise failed to return single err")
	}
}

func TestPromise_All2(t *testing.T) {
	var promises = make([]*Promise, 10)

	for index := 0; index < 10; index++ {
		promises[index] = Resolve(fmt.Sprintf("All Good %d", index))
	}

	combined := All(promises...)
	result, err := combined.Await()
	if err != nil {
		t.Error(err)
	} else {
		for index, res := range result.([]interface{}) {
			s := fmt.Sprintf("All Good %d", index)
			if res == nil {
				t.Error("RESULT IS NIL!")
				return
			}
			if res.(string) != s {
				t.Error("Wrong index!")
				return
			}
		}
	}
}

func TestPromise_All3(t *testing.T) {
	var promises []*Promise

	combined := All(promises...)
	result, err := combined.Await()

	if err != nil {
		t.Error(err)
		return
	}

	res := result.([]interface{})
	if len(res) != 0 {
		t.Error("Wrong result on nil slice")
		return
	}
}
