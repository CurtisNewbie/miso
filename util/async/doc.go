// Package for async task processing.
//
// Core types in this package are: [AsyncPool], [Future], [AwaitFutures].
//
// [Future] represents the result of an async task, similar to Future in Java and Promise in Javascript.
// A Future completes exactly once: its value and error are published before every waiter is woken, so
// no reader observes a partially published result. [Future.TimedGet] returns the zero value and
// [ErrGetTimeout] when it gives up waiting; the task may still complete afterwards.
//
// [Future.Then] registers a callback and returns a derived Future that completes with the same value and
// error once the callback returns, so callbacks chain in registration order. A panicking callback is
// recovered and logged, and does not fail the task or stop the next callback. Two Then calls on the same
// Future are independent branches and both fire.
//
// Use [AsyncPool.Run], [Submit] or [Run] to create an async task, and obtain task result through the
// returned [Future], e.g., [Future.Get] and [Future.TimedGet]. A task a pool refuses to run completes its
// Future with [ErrTaskDropped] instead of leaving the caller waiting forever.
//
// [AsyncPool] internally maintain a pool of goroutines. Use [NewAsyncPool] create a new [AsyncPool],
// customize the pool with options like [WithTaskQueue], [FallbackCallerRun], [FallbackDropTask] and [FallbackNewGoroutine].
// Stopping a pool blocks new submissions; accepted work drains in the background and [AsyncPool.StopAndWait]
// returns once it finishes. A task submitted after stop runs inline or goes through the configured fallback.
//
// Use [CalcPoolSize] to estimate optimal worker count in the pool.
//
// For cases where you need to await for a group of Futures, use [NewAwaitFutures]. [AwaitFutures] represent a group of tasks that are triggered at the
// same time and awaited together. A group seals on the first await: [AwaitFutures.Await] returns a copy of
// the group's Futures, and submitting to a sealed group panics.
//
// See [BatchTask] if you need to manage a group of task generator and consumers without using a goroutine pool.
//
// See [SignalOnce] for one-time signal based communication.
//
// async package also provides various convenience funcs to run async task until certain condition match, or help you capture potential panic in async task.
// E.g., [RunCancellable], [RunCancellableChan], [RunUntil], [CapturePanic], [CapturePanicErr], [PanicSafeFunc] and so on.
package async
