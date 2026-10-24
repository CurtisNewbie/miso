# 02: Await group safety

**What to build:** An await group can be filled from several goroutines, reads back a stable set, and seals once awaited. Await returns only after every future in the group has completed, so callers never received a half-finished future.

**Blocked by:** 01.

**Status:** done

- [x] Concurrent submits from many goroutines pass under the race detector.
- [x] Await returns a copy of the group's futures, so a later append cannot change what the caller holds.
- [x] A submit after the first await fails loudly with a clear message instead of being silently unawaited.
- [x] Await returns only after every future has completed, not merely after each task function returned.
- [x] Repeated awaits return the same sealed set.
