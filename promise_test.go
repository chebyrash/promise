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
		Name        string
		Promises    []*Promise
		Expected    []interface{}
	}

	// Truthy test cases (no error expectations)
	for _, tc := range []TestAllTestCase{
		{
			Name:     "With nil promise list",
			Promises: nil,
			Expected: nil,
		},
		{
			Name:     "With 1 \"already\" resolved promise",
			Promises: []*Promise{
				Resolve(99),
			},
			Expected: []interface{}{99},
		},
		{
			Name:     "With only \"already\" resolved promises",
			Promises: []*Promise{
				Resolve(99),
				Resolve(100),
				Resolve(101),
			},
			Expected: []interface{}{99, 100, 101},
		},
		{
			Name:     "With more than one \"already\" resolved promises",
			Promises: []*Promise{
				Resolve(99),
				New(func (resolve func(interface{}), reject func(error)) {
					resolve (100)
				}),
				Resolve(101),
			},
			Expected: []interface{}{99, 100, 101},
		},
	} {
		t.Run(tc.Name, func(t2 *testing.T) {
			var p *Promise = All(tc.Promises) // Marking as var (with type) ensures
											  // correct type is returned (upfront)
		    // Await promise
			data, err := p.Await()

			t2.Logf("Received: result: %v;  err: %v", data, err)

			// If error fatal-out since there shouldn't be any errors in this test-case set
			if err != nil {
				t2.Fatalf("Error occurred for test case \"%s\": %v", tc.Name, err)
			}

			result := data.([]interface{})

			// Check result 'len' vs expected 'len'
			ExpectEqual(
				t2,
				"Returned data len",
				len(result),
				len(tc.Expected),
			)

			// Compare received data to expected
			t2.Run("Returned data comparison", func(t3 *testing.T) {
				for i, x := range tc.Expected {
					testName := fmt.Sprintf("%v === %v", x, result[i])
					t3.Run(testName, func(t4 *testing.T) {
						ExpectEqual(t4, testName, x, result[i])
					})
				}
			})

		}) // test suite

	} // for loop

	// Falsy test cases
	func () {
		type TestCheck func (*testing.T, error)

		type TestAllFailingTestCase struct {
			Name        string
			Promises    []*Promise
			ExpectCheck TestCheck
		}

		FakeError1 := errors.New("FakeError1")
		FakeError2 := errors.New("FakeError2")
		FakeError3 := errors.New("FakeError3")

		GetExpectRejectionErrorCheck := func (e2 error) TestCheck {
			return func (tx *testing.T, e1 error) {
				tx.Run(fmt.Sprintf("Expect %v", e2), func(t2 *testing.T) {
					ExpectEqual(t2, "Rejection error", e1, e2)
				})
			}
		}

		GetExpectOneOfErrors := func (errs []error) TestCheck {
			return func(tx *testing.T, e error) {
				tx.Run("Expect one of our defined errors", func(t2 *testing.T) {
					for _, e2 := range errs {
						if e == e2 {
							return
						}
					}
					t2.Errorf("Expected one of our defined errors.  Got %v", e)
				})
			}
		}

		ExpectOneOfDefinedErrors := GetExpectOneOfErrors([]error{
			FakeError1, FakeError2, FakeError3,
		})

		// Falsy test cases (error expectations)
		for _, tc := range []TestAllFailingTestCase{
			{
				Name: "When one promise",
				Promises: []*Promise{Reject(FakeError1)},
				ExpectCheck: GetExpectRejectionErrorCheck(FakeError1),
			},
			{
				Name: "When more than one promise (and one 'rejecting' promise)",
				Promises: []*Promise{
					Resolve("Hello World"),
					Reject(FakeError1),
					Resolve("Hola mundo"),
				},
				ExpectCheck: GetExpectRejectionErrorCheck(FakeError1),
			},
			{
				Name: "When more than one promise (and one 'rejecting' promise)",
				Promises: []*Promise{
					Resolve("Hello World"),
					Resolve("Hello World"),
					Resolve("Hello World"),
					Reject(FakeError2),
					Resolve("Hola mundo"),
					Resolve("Hola mundo"),
					Resolve("Hola mundo"),
				},
				ExpectCheck: GetExpectRejectionErrorCheck(FakeError2),
			},
			{
				Name: "When more than one rejecting promise",
				Promises: []*Promise{
					Resolve("Hello World"),
					Reject(FakeError1),
					Resolve("Ola mundo?"),
					Resolve(FakeError2),
					Resolve("Hola mundo"),
				},
				ExpectCheck: ExpectOneOfDefinedErrors,
			},
			{
				Name: "When only rejecting promises (more than one)",
				Promises: []*Promise{
					Reject(FakeError3),
					Reject(FakeError1),
					Reject(FakeError2),
				},
				ExpectCheck: ExpectOneOfDefinedErrors,
			},
		} {
			t.Run(tc.Name, func(t2 *testing.T) {
				p := All(tc.Promises)

				result, err := p.Await()

				ExpectEqual(t2, "Result", result, nil)

				// Check error expectations
				tc.ExpectCheck(t2, err)
			})
		}
	}()
}

func ExpectEqual(t *testing.T, prefix string, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("%s;  Expected %v;  Got %v", prefix, b, a)
	}
}
