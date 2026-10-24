# 04: Race verification and docs

**What to build:** The async package and its dependents are race clean, and the package documentation states the completion contract so the next change cannot quietly reintroduce a lost wakeup or a lost task.

**Blocked by:** 01, 02, 03.

**Status:** done

- [x] The async package suite passes under the race detector, including the existing timeout test.
- [x] Race tests for the dependents that use futures and await groups pass (`middleware/redis`, `cmd/misoapi`, `miso`).
- [x] Package documentation and doc comments describe completion, chaining, dropped-task errors, the stop barrier, and group sealing.
- [x] No exported API is added beyond `ErrTaskDropped` and the return value of `Then`.

## Comments

`middleware/redis` passes under `-race`; `cmd/misoapi` has no test files. The `miso` package run reports no data race, but six tests fail: five consul tests need a missing `../testdata/conf_dev.yml` fixture and `TestWrapMisoErr` asserts the wrong error code. The same six fail on `main` with the same commands, so they are pre-existing and unrelated to this change.

