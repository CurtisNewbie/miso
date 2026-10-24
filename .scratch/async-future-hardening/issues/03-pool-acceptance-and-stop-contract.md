# 03: Pool acceptance and stop contract

**What to build:** A pool accounts for every task it accepts exactly once. Stopping blocks new work, drains the queue, and only then releases the stop barrier. A refused task completes its future with `ErrTaskDropped` instead of leaving the caller waiting forever.

**Blocked by:** 01.

**Status:** done

- [x] Every path that handles a task, accepted, dropped, run inline, fallback, or stopped, increments and decrements the wait group exactly once.
- [x] A full pool with the drop fallback neither panics nor hangs; the future completes with `ErrTaskDropped` and fire-and-forget reports a failure.
- [x] Stop blocks new submissions, drains queued tasks, and `StopAndWait` returns only after the drained work finishes.
- [x] A concurrent stop and submit storm passes under the race detector with no send on a closed channel.
- [x] A submission after stop runs inline when the pool blocks on a full queue, and otherwise goes through the configured fallback.
- [x] The ants pool reaches parity, including a failed submit, a submission after stop, and a correct stop barrier.
- [x] The default fallback on both pools runs the task on the caller rather than discarding it.
- [x] A pool that only implements the public interface still compiles and still accepts work, with no drop reporting.
