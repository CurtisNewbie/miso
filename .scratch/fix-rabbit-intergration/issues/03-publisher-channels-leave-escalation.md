# 03: Stop publisher channel death from rebuilding the connection

**What to build:** A dead publisher channel no longer escalates to a connection rebuild. Publisher channels live in the pool as a cache, not as supervised resources. The pool discards a channel that is already closed when it is next borrowed, and the borrow path opens a replacement on demand, confirms it and hands it out. A publish that loses its channel surfaces as a failed confirmation to the caller, and the next publish uses a fresh channel without touching the connection.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] A publisher channel death produces no connection rebuild and no escalation signal.
- [ ] Borrowing from a pool whose channels are all closed opens replacements instead of failing.
- [ ] A failed publish caused by a dead channel surfaces as an error to the caller, and a later publish succeeds without a reconnect.
- [ ] The configured pool size still bounds how many channels are opened, and a refill for a drained borrow opens one channel per borrow attempt.