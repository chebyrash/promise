# Performance measurements

Measured on an Apple M4 Pro (darwin/arm64, GOMAXPROCS=14) using Go 1.27.1.
Values are medians of three runs. The original baseline is commit `43d9f27`,
built with the same toolchain. Its benchmark uses the old package functions;
the current benchmark uses methods with the same callbacks and workload.
Current results include value-returning `Await`, typed `Catch` recovery,
non-nil rejection errors, and synchronous argument validation.
Timings vary by machine and workload; small differences are not statistically
established speedups.

```sh
GOTOOLCHAIN=go1.27.1 go test -run '^$' -bench . -benchmem -count 3
```

## Further improvements after the first Go 1.27 implementation

The previous column is the first implementation from this migration, before
removing compatibility functions and optimizing settlement and waiting.

| Benchmark | Previous ns/op | Current ns/op | Previous allocs/op | Current allocs/op |
| --- | ---: | ---: | ---: | ---: |
| Then | 334.90 | 297.30 | 4 | 3 |
| AwaitSettled | 13.21 | 6.96 | 0 | 0 |
| Combinators/All/100 | 4,302.00 | 1,011.00 | 8 | 5 |
| Combinators/Race/100 | 386.40 | 88.75 | 6 | 4 |
| CombinatorsPending/All/100 | 9,112.00 | 6,663.00 | 210 | 207 |
| CombinatorsPending/Race/100 | 5,308.00 | 5,272.00 | 208 | 206 |

Pending `Race(100)` timing is roughly unchanged in these runs, with fewer
allocations. Improvements to settled-input benchmarks should not be assumed
to apply equally to pending or long-running workloads.

`AwaitSettledCancelable` currently measures 29.66 ns/op with zero allocations.
An earlier separate baseline (five runs at 300 ms each) measured 27.59 ns/op.
The cancellable path does not benefit from the fast path for contexts without
a cancellation channel.

## Compared with the original library

| Benchmark | Original ns/op | Current ns/op | Original B/op | Current B/op | Original allocs/op | Current allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| NewAwait | 317.60 | 301.60 | 248 | 240 | 6 | 5 |
| Then | 356.40 | 297.30 | 296 | 224 | 7 | 3 |
| AwaitSettled | 13.11 | 6.96 | 0 | 0 | 0 | 0 |
| Combinators/All/1 | 1,455.00 | 80.52 | 1,256 | 280 | 27 | 5 |
| Combinators/Race/1 | 1,431.00 | 63.20 | 1,224 | 212 | 26 | 4 |
| Combinators/All/10 | 8,580.00 | 160.80 | 7,464 | 352 | 171 | 5 |
| Combinators/Race/10 | 5,402.00 | 63.33 | 7,192 | 212 | 170 | 4 |
| Combinators/All/100 | 60,245.00 | 1,011.00 | 69,740 | 1,168 | 1611 | 5 |
| Combinators/Race/100 | 51,668.00 | 88.75 | 67,261 | 212 | 1610 | 4 |
| ThenPending | 397.50 | 345.10 | 464 | 384 | 10 | 5 |
| CombinatorsPending/All/10 | 9,125.00 | 975.90 | 9,144 | 2,128 | 201 | 27 |
| CombinatorsPending/Race/10 | 5,883.00 | 847.00 | 8,872 | 1,988 | 200 | 26 |
| CombinatorsPending/All/100 | 69,571.00 | 6,663.00 | 86,543 | 18,176 | 1911 | 207 |
| CombinatorsPending/Race/100 | 56,748.00 | 5,272.00 | 84,068 | 17,209 | 1910 | 206 |

## What the benchmarks measure

- `NewAwait` creates and awaits a resolved promise.
- `Then` and `AwaitSettled` reuse a settled source; source creation is excluded.
  The faster `Await` path applies to contexts whose `Done` channel is nil.
  `AwaitSettledCancelable` separately measures a live cancellable context.
- `Combinators` reuses settled inputs; `All` preserves every result, while
  `Race` can select the first ready input after validating all inputs. Validation
  is O(n), even when a winner is ready. This is its best-case workload.
- `ThenPending` creates an unresolved source, attaches a transformation, then
  resolves it. Source creation is included.
- `CombinatorsPending` creates unresolved inputs, attaches the aggregate, then
  resolves every input. Source creation is included. The coordinator may find
  inputs already settled by the time it runs; these benchmarks model short
  producer work, not permanently blocked or long-running tasks.

## Implementation changes

- Use native generic methods and typed recovery. Package-level chaining functions
  are removed; `Await` returns a value and error, with no shared scalar pointer.
- Store resolved values inside the promise, avoiding a separate allocation.
- Remove a redundant callback closure from `Then` and `Catch` dispatch.
- Wait directly on the settlement channel for contexts that cannot be canceled.
- Read ready aggregate inputs once, instead of selecting again in `Await`.
- Settle default-pool aggregates directly, eliminating a goroutine and its closures.
  Custom pools still dispatch settlement, and default-pool user callbacks remain
  asynchronous.
- Combine aggregation counters into one allocation.
- Remove intermediate promises and result/error channels from aggregation.
- Consume ready inputs together; register pending inputs in a coordinator and
  release remaining waiters after settlement.
- Use one goroutine for waiting and transforming with the default pool. Custom
  pools receive continuations after their dependencies become ready, preventing
  dependency waits from occupying bounded workers.
