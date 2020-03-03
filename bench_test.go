package promise

import (
	"errors"
	"testing"
	"time"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		New(func(resolve func(interface{}), reject func(error)) {
			resolve(nil)
		})
	}
}

func BenchmarkThen(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var promise = New(func(resolve func(interface{}), reject func(error)) {
			resolve(1 + 1)
		})

		promise.
			Then(func(data interface{}) interface{} {
				return data.(int) + 1
			}).
			Then(func(data interface{}) interface{} {
				return nil
			}).
			Catch(func(err error) error {
				return nil
			})

		promise.Await()
	}
}

func BenchmarkCatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var promise = New(func(resolve func(interface{}), reject func(error)) {
			reject(errors.New("very serious err"))
		})

		promise.
			Catch(func(err error) error {
				return nil
			}).
			Catch(func(err error) error {
				return nil
			})

		promise.Then(func(data interface{}) interface{} {
			return nil
		})

		promise.Await()
	}
}

func BenchmarkAwait(b *testing.B) {
	var promises = make([]*Promise, 10)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
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
			p.Await()
		}

		promise1.Await()
		promise2.Await()
	}
}

func BenchmarkResolve(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var promise = Resolve(123).
			Then(func(data interface{}) interface{} {
				return data.(int) + 1
			}).
			Then(func(data interface{}) interface{} {
				return nil
			})

		promise.Await()
	}
}

func BenchmarkReject(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var promise = Reject(errors.New("rejected")).
			Then(func(data interface{}) interface{} {
				return data.(int) + 1
			}).
			Catch(func(err error) error {
				return nil
			})

		promise.Await()
	}
}

func BenchmarkAll(b *testing.B) {
	var promises = make([]*Promise, 10)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for x := 0; x < 10; x++ {
			if x == 8 {
				promises[x] = Reject(errors.New("bad promise"))
				continue
			}

			promises[x] = Resolve("All Good")
		}

		All(promises...).Await()
	}
}

func BenchmarkAllSettled(b *testing.B) {
	var promises = make([]*Promise, 10)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for x := 0; x < 10; x++ {
			if x == 8 {
				promises[x] = Reject(errors.New("bad promise"))
				continue
			}

			promises[x] = Resolve("All Good")
		}

		AllSettled(promises...).Await()
	}
}

func BenchmarkRace(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var p1 = Resolve("Promise 1")
		var p2 = Resolve("Promise 2")

		Race(p1, p2).Await()
	}
}
