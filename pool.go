package promise

import (
	"sync/atomic"

	"github.com/panjf2000/ants/v2"
	conc "github.com/sourcegraph/conc/pool"
)

var defaultPool atomic.Pointer[poolHolder]

type poolHolder struct{ pool Pool }

func init() { SetDefaultPool(newDefaultPool()) }

func getDefaultPool() Pool { return defaultPool.Load().pool }

// Pool schedules promise work. Go must run f exactly once or panic if submission
// fails; promises convert submission panics to rejections.
type Pool interface {
	Go(f func())
}

// SetDefaultPool changes the pool for subsequently created promises and chains.
// It is safe to call concurrently with promise operations. p must not be nil.
func SetDefaultPool(p Pool) {
	if p == nil {
		panic("pool is nil")
	}
	defaultPool.Store(&poolHolder{pool: p})
}

type wrapFunc func(f func())

func (wf wrapFunc) Go(f func()) {
	wf(f)
}

type goroutinePool struct{}

func (goroutinePool) Go(f func()) { go f() }

func newDefaultPool() Pool { return goroutinePool{} }

// FromConcPool adapts a conc pool. A nil pool panics synchronously.
func FromConcPool(p *conc.Pool) Pool {
	if p == nil {
		panic("conc pool is nil")
	}
	return wrapFunc(p.Go)
}

// FromAntsPool adapts an ants pool. Failed submissions reject the promise.
// A nil pool panics synchronously.
func FromAntsPool(p *ants.Pool) Pool {
	if p == nil {
		panic("ants pool is nil")
	}
	return wrapFunc(func(f func()) {
		if err := p.Submit(f); err != nil {
			panic(err)
		}
	})
}
