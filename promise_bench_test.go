package promise

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkNewAwait(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		p := New(func(resolve func(int), _ func(error)) { resolve(42) })
		if _, err := p.Await(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkThen(b *testing.B) {
	ctx := context.Background()
	p := New(func(resolve func(int), _ func(error)) { resolve(42) })
	_, _ = p.Await(ctx)
	b.ReportAllocs()
	for b.Loop() {
		q := p.Then(ctx, func(v int) (int, error) { return v + 1, nil })
		if _, err := q.Await(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAwaitSettled(b *testing.B) {
	ctx := context.Background()
	p := New(func(resolve func(int), _ func(error)) { resolve(42) })
	_, _ = p.Await(ctx)
	b.ReportAllocs()
	for b.Loop() {
		_, _ = p.Await(ctx)
	}
}

func BenchmarkCombinators(b *testing.B) {
	ctx := context.Background()
	for _, n := range []int{1, 10, 100} {
		promises := make([]*Promise[int], n)
		for i := range promises {
			promises[i] = New(func(resolve func(int), _ func(error)) { resolve(42) })
			_, _ = promises[i].Await(ctx)
		}
		for _, name := range []string{"All", "Race"} {
			b.Run(fmt.Sprintf("%s/%d", name, n), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if name == "All" {
						if _, err := All(ctx, promises...).Await(ctx); err != nil {
							b.Fatal(err)
						}
					} else {
						if _, err := Race(ctx, promises...).Await(ctx); err != nil {
							b.Fatal(err)
						}
					}
				}
			})
		}
	}
}

func BenchmarkThenPending(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		p := &Promise[int]{ch: make(chan struct{})}
		q := p.Then(ctx, func(n int) (int, error) { return n + 1, nil })
		p.resolve(42)
		if _, err := q.Await(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCombinatorsPending(b *testing.B) {
	ctx := context.Background()
	for _, n := range []int{10, 100} {
		for _, name := range []string{"All", "Race"} {
			b.Run(fmt.Sprintf("%s/%d", name, n), func(b *testing.B) {
				promises := make([]*Promise[int], n)
				b.ReportAllocs()
				for b.Loop() {
					for i := range promises {
						promises[i] = &Promise[int]{ch: make(chan struct{})}
					}
					if name == "All" {
						q := All(ctx, promises...)
						for _, p := range promises {
							p.resolve(42)
						}
						if _, err := q.Await(ctx); err != nil {
							b.Fatal(err)
						}
					} else {
						q := Race(ctx, promises...)
						for _, p := range promises {
							p.resolve(42)
						}
						if _, err := q.Await(ctx); err != nil {
							b.Fatal(err)
						}
					}
				}
			})
		}
	}
}

func BenchmarkAwaitSettledCancelable(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := New(func(resolve func(int), _ func(error)) { resolve(42) })
	_, _ = p.Await(ctx)
	b.ReportAllocs()
	for b.Loop() {
		_, _ = p.Await(ctx)
	}
}
