// package for async task processing.
//
// Core types in this package are: [AsyncPool], [Future], [AwaitFutures].
//
// [AsyncPool] internally maintain a pool of goroutines. Use [NewAsyncPool] create a new [AsyncPool],
// customize the pool with options like [WithTaskQueue], [FallbackCallerRun] and [FallbackDropTask].
//
// Use [CalcPoolSize] to estimate optimal worker count in the pool.
//
// [Future] represents the result of an async task, similar to Future in Java and Promise in Javascript.
//
// Use [AsyncPool.Run], [Submit] or [Run] to create an async task, and obtain task result through the returned [Future],
// e.g., [Future.Get] and [Future.TimedGet].
//
// For cases where you need to await for a group of Futures, use [NewAwaitFutures]. [AwaitFutures] represent a group of tasks that are triggered at the
// same time and awaited together.
//
// See [BatchTask] if you need to manage a group of task generator and consumers without using a goroutine pool.
//
// See [SignalOnce] for one-time signal based communication.
//
// async package also provides various convenience funcs to run async task until certain condition match, or help you capture potential panic in async task.
// E.g., [RunCancellable], [RunCancellableChan], [RunUntil], [CapturePanic], [CapturePanicErr], [PanicSafeFunc] and so on.
package async
