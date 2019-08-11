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

func TestAll(t *testing.T) {
	type TestAllTestCase struct {
		Name           string
		Promises       []*Promise
		ExpectedResult []interface{}
		ExpectedError  interface{}
	}

	FakeError1 := errors.New("FakeError1")
	FakeError2 := errors.New("FakeError2")
	FakeError3 := errors.New("FakeError3")

	// Loop through test cases
	for _, tc := range []TestAllTestCase{
		{
			Name:           "With no promises",
			Promises:       nil,
			ExpectedResult: nil,
			ExpectedError:  nil,
		},
		{
			Name: "With one promise",
			Promises: []*Promise{
				Resolve(99),
			},
			ExpectedResult: []interface{}{99},
			ExpectedError:  nil,
		},
		{
			Name: "With only resolved promises",
			Promises: []*Promise{
				Resolve(99),
				Resolve(100),
				Resolve(101),
			},
			ExpectedResult: []interface{}{99, 100, 101},
			ExpectedError:  nil,
		},
		{
			Name:           "With one promise (rejected)",
			Promises:       []*Promise{Reject(FakeError1)},
			ExpectedResult: nil,
			ExpectedError:  FakeError1,
		},
		{
			Name: "With multiple promises (and one rejected)",
			Promises: []*Promise{
				Resolve("Hello World"),
				Reject(FakeError1),
				Resolve("Hola mundo"),
			},
			ExpectedResult: nil,
			ExpectedError:  FakeError1,
		},
		{
			Name: "With multiple promises (and one rejected)",
			Promises: []*Promise{
				Resolve("Hello World"),
				Resolve("Hello World"),
				Resolve("Hello World"),
				Reject(FakeError2),
				Resolve("Hola mundo"),
				Resolve("Hola mundo"),
				Resolve("Hola mundo"),
			},
			ExpectedResult: nil,
			ExpectedError:  FakeError2,
		},
		{
			Name: "With multiple promises and multiple rejecting",
			Promises: []*Promise{
				Resolve("Hello World"),
				New(func(resolve func(interface{}), reject func(error)) {
					time.Sleep(time.Millisecond * 20)
					reject(FakeError1)
				}),
				Resolve("Ola mundo?"),
				Reject(FakeError3),
				Resolve("Hola mundo"),
			},
			ExpectedResult: nil,
			ExpectedError:  FakeError3,
		},
		{
			Name: "When only rejecting promises (more than one)",
			Promises: []*Promise{
				New(func(resolve func(interface{}), reject func(error)) {
					time.Sleep(time.Millisecond * 20)
					reject(FakeError3)
				}),
				New(func(resolve func(interface{}), reject func(error)) {
					reject(FakeError1)
				}),
				New(func(resolve func(interface{}), reject func(error)) {
					time.Sleep(time.Millisecond * 20)
					reject(FakeError2)
				}),
			},
			ExpectedResult: nil,
			ExpectedError:  FakeError1,
		},
	} {
		t.Run(tc.Name, func(t2 *testing.T) {
			p := All(tc.Promises...)
			data, err := p.Await()
			var result []interface{}

			switch data.(type) {
			case []interface{}:
				result = data.([]interface{})
			default:
				result = nil
			}

			subTestName := fmt.Sprintf("Expect result length: %v", tc.ExpectedResult)
			t2.Run(subTestName, func(t3 *testing.T) {
				if len(result) != len(tc.ExpectedResult) {
					t3.Errorf("Expected length %v;  Received %v", tc.ExpectedError, err)
				}
			})

			subTestName = fmt.Sprintf("Expect error: %v", tc.ExpectedError)
			t2.Run(subTestName, func(t3 *testing.T) {
				if err != tc.ExpectedError {
					t3.Errorf("Expected error %v;  Received %v", tc.ExpectedError, err)
				}
			})

			t2.Run("Returned data comparison", func(t3 *testing.T) {
				for i, x := range tc.ExpectedResult {
					testName := fmt.Sprintf("%v === %v", x, result[i])
					t3.Run(testName, func(t4 *testing.T) {
						if err != tc.ExpectedError {
							t3.Errorf("Expected value %v;  Received %v", x, result[i])
						}
					})
				}
			})

		}) // test case run

	} // test cases loop
}
