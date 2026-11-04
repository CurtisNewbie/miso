# 02: Escalate to a connection rebuild after two failed restarts

**What to build:** A slot that cannot restart in place stops trying and asks the connection supervisor to rebuild. A restart fails when opening the channel or subscribing returns an error, or when the new channel dies before staying up for one minute. The second consecutive failure escalates, and a slot that survives a minute resets its count so a later death starts from the first attempt again. Escalation goes through the generation's coalescing signal, so several slots escalating at once, or an escalation racing a connection close, still produce one rebuild.

Logging is bounded by the same rule: one warning per in-place attempt, one error per escalation, and no repeating line while a slot waits.

**Blocked by:** 01.

**Status:** done

- [x] Two consecutive failed restarts produce one escalation and no third attempt.
- [x] A slot that survives the stable window resets its failure count.
- [x] Two slots escalating together produce one generation rebuild.
- [x] An escalation arriving with a connection close produces one generation rebuild.
- [x] Log output carries one warning per attempt and one error per escalation, with no per-second repetition.