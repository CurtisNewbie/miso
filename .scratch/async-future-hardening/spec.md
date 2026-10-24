# Async future hardening

Status: ready-for-agent

## Problem Statement

The async utility package ships concurrency defects that surface under load and under the race detector.

A timed get that times out reads the task's result field with no synchronization against the goroutine about to write it. The package's own `TestAsyncOnce` fails under `go test -race` for this reason. `AwaitFutures` appends to its internal slice with no lock, so concurrent submits corrupt the slice, and the slice it hands back is the live one. A task a pool refuses runs never completes its future, so callers of `Get`, await groups, and fire-and-forget error logging block forever. `BoundedAsyncPool` double decrements its wait group on the drop path and can send on the task channel after `Stop` closes it, both of which panic. `StopAndWait` can return while a concurrently submitted task still runs. `SignalOnce` leaks a timer on every timed wait.

## Solution

Make completion the single contract every path honours.

A future completes exactly once. Publishing the result and waking waiters happen in one order that every reader synchronizes with. A timeout reports no result, a dropped task completes its future with `ErrTaskDropped`, and a stopped pool drains what it already accepted. `Then` returns a derived future so callbacks chain by type instead of overwriting one another. Await groups guard their state with a mutex and seal on first await.

## User Stories

1. As a service author, I want a timed get that times out to return no result and no data race, so that I can use timeouts on hot paths with the race detector enabled.
2. As a service author, I want the zero value plus `ErrGetTimeout` on timeout, so that a partial result can never be mistaken for a finished task.
3. As a service author, I want `Then` to return a future, so that I can chain callbacks instead of writing nested closures.
4. As a service author, I want chained callbacks to run in registration order, so that a chain reads the same way it executes.
5. As a service author, I want a panicking callback to be recovered and logged without corrupting the value passed down the chain, so that one bad observer cannot fail the task.
6. As a service author, I want two `Then` calls on the same future to be independent branches that both fire, so that adding an observer never silently replaces an existing one.
7. As a service author, I want `Then` on an already completed future to fire immediately and return a completed future, so that chaining works the same before and after completion.
8. As a service author, I want a future from a completed value to behave exactly like any other future, so that shutdown hooks and pre-resolved results take one code path.
9. As a service author, I want concurrent submits to an await group to be safe, so that fan-out code can submit from several goroutines.
10. As a service author, I want the await group to hand back a copy of its futures, so that my iteration cannot observe a later append.
11. As a service author, I want an await group to seal on first await, so that a late submit fails loudly instead of being silently unawaited.
12. As a service author, I want await to wait for real completion, so that a returned future is already resolved when await returns.
13. As a service author, I want a dropped task to complete its future with a specific error, so that pool pressure shows up as a failed task and not as a permanent hang.
14. As a service author, I want fire-and-forget logging to report a dropped task, so that silent loss stops happening.
15. As a service author, I want a pool that reports whether it accepted a task, so that a future can be completed on rejection without changing the public pool interface.
16. As a service author, I want stopping a pool to block new submissions and drain the queue, so that queued work finishes and nothing is lost.
17. As a service author, I want a submission that arrives after stop to run inline or use its fallback, so that the pool never wedges the caller.
18. As a service author, I want `StopAndWait` to be a true barrier, so that returning means the accepted work finished.
19. As a service author, I want stop and submit to be safe against each other, so that neither panics with a send on a closed channel.
20. As a service author, I want the drop fallback to decrement the wait group exactly once, so that a full pool does not panic with a negative counter.
21. As a service author, I want the default fallback to run the task on the caller instead of discarding it, so that an unconfigured pool never loses work.
22. As a maintainer, I want repeated timed waits to stop allocating timers that outlive the wait, so that high-frequency timeouts do not grow memory.
23. As a maintainer, I want the package documentation and glossary to state the completion contract, so that the next change does not reintroduce a lost-wakeup or a lost task.

## Implementation Decisions

- The async utility package is the only module under change. Its public types, `Future`, `AsyncPool`, `AwaitFutures`, and the constructors, keep their names. `Then` gains a return value, which is source compatible because Go callers may ignore it.
- `TimedGet` on timeout returns the zero value and `ErrGetTimeout`. The result fields are only read after the completion signal has been observed.
- Completion publishes value and error before waking waiters. All readers either observe the completion signal first or hold the same mutex.
- `Then` returns a derived `Future[T]` that completes with the parent's value and error after the callback returns. The callback runs on the completing goroutine. A callback panic is recovered and logged. Chaining stays same-type because Go methods cannot declare their own type parameters. Cross-type mapping is a separate exported function and is not part of this change.
- A second `Then` on the same future is a second, independent branch. There is no single callback slot and no panic. `Then(nil)` keeps panicking.
- A completed future fires its callback synchronously and returns a future completed with the same value and error.
- `AwaitFutures` holds a mutex and a sealed flag. Submit takes the mutex, rejects submissions after seal, and appends the future. Await takes the mutex once, seals, copies the slice, then waits for every future to complete. Repeated awaits return copies of the same sealed set.
- A dropped task completes its future with a new exported sentinel, `ErrTaskDropped`. Submission uses an internal capability that reports whether a pool accepted the task. Pools that only implement the public interface fall back to the old submission call, so external pool implementations still compile; they cannot report drops.
- Fallback callbacks no longer touch the wait group. The pool owns one increment and exactly one decrement per accepted or dropped task, on every path.
- `BoundedAsyncPool` stops by setting a stopped flag under a lock that also guards submission, and by closing a stop channel. The task channel is never closed. Workers drain the queue after stop and exit when it is empty.
- A submission after stop runs inline when the pool blocks on a full queue, and otherwise goes through the configured fallback. `StopAndWait` waits on the same counter the submissions increment.
- `AntsAsyncPool` uses the same stopped flag and wait-group discipline. A failed submit goes through the fallback, and the default fallback runs the task on the caller in both pools.
- `SignalOnce` timed waits use a timer that is stopped when the wait returns.
- Fire-and-forget with a nil logging handle does not panic. A custom runner is documented as owning execution of the wrapped function.

## Testing Decisions

- Tests assert external behaviour of the async package only. No test reaches into private fields or asserts on goroutine counts.
- The single seam is the package's public API, in two groups. Future behaviour covers timeout results, chaining order, callback panic isolation, completed futures, concurrent callbacks, and await-group submit and seal. Pool behaviour covers drain on stop, post-stop submission, drop on a full pool, drop on a stopped pool, the stop barrier, ants after release, and the submit and stop storm.
- Ordering comes from channel handshakes, not sleeps. A test that needs the worker busy closes a channel from inside the task.
- Every test in the seam runs under the race detector, and the existing package suite must pass under `-race`.
- Prior art is the existing async tests, which already exercise `Run`, `Submit`, await groups, and chained callbacks, plus the existing pool lifecycle test.

## Out of Scope

- Cross-type then mapping and any richer chain type.
- Context cancellation, deadlines on `Get`, and future cancellation.
- Changing the `Get` signature or the public `AsyncPool` interface.
- Pool sizing, worker tuning, and benchmark work.
- Migration patches. The changes are source compatible, so no gopatch entry is added.
- Reworking callers outside the async package.

## Further Notes

The work lands on `feat/async-future-hardening` in a worktree cut from `main`, and merges only after review. Existing callers pass `Then` as a statement, so the new return value changes no call site.
