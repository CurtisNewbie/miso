# 04: Race verification and docs

**What to build:** The async package and its dependents are race clean, and the package documentation states the completion contract so the next change cannot quietly reintroduce a lost wakeup or a lost task.

**Blocked by:** 01, 02, 03.

**Status:** ready-for-agent

- [ ] The async package suite passes under the race detector, including the existing timeout test.
- [ ] Race tests for the dependents that use futures and await groups pass (`middleware/redis`, `cmd/misoapi`, `miso`).
- [ ] Package documentation and doc comments describe completion, chaining, dropped-task errors, the stop barrier, and group sealing.
- [ ] No exported API is added beyond `ErrTaskDropped` and the return value of `Then`.
