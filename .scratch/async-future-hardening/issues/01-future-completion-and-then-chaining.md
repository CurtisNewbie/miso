# 01: Future completion and Then chaining

**What to build:** A future publishes its result exactly once, and every path a reader can take observes that result consistently. A timed get that times out reports no result. Callbacks attach as a chain instead of replacing one another, both before and after completion.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] A timed get that times out returns the zero value and `ErrGetTimeout`, and the existing timeout test passes under the race detector.
- [x] Completion publishes value and error before waking waiters; every reader observes the completion signal or holds the mutex.
- [x] `Then` returns a future, and chained callbacks run in registration order.
- [x] A panicking callback is recovered and logged, and the value and error passed down the chain are unchanged.
- [x] Two `Then` calls on one future both fire, with no second-call panic.
- [x] `Then` on an already completed future fires immediately and returns a completed future, and a future created from a completed value behaves the same.
- [x] Fire-and-forget with a nil logging handle does not panic.
- [x] A timed wait stops its timer instead of leaking it.
- [x] `ErrTaskDropped` exists, and submission can complete a future with it when the pool reports a rejection.
