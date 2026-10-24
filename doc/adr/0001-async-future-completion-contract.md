# Async future completion contract

A future completes exactly once, and every path that cannot run its task must still complete the future. We chose this over silent drops and blocking waits because a future that never completes turns a pool-pressure event into a permanent hang. The contract has four parts. `TimedGet` reports a timeout with a zero value and never exposes a partially published result. A task a pool refuses to run completes its future with `ErrTaskDropped`. Stopping a pool blocks new submissions and drains the queue before `StopAndWait` returns. An await group seals on first await, so concurrent submissions cannot be appended to a slice already being read.

## Consequences

`FallbackDropTask` remains available and now reports `ErrTaskDropped` instead of leaving futures pending. Callers that supply their own `AsyncPool` implementation still work, but a pool that cannot report a drop keeps the old hang risk. `Then` returns a derived future for same-type chaining, because Go methods cannot declare type parameters of their own.
