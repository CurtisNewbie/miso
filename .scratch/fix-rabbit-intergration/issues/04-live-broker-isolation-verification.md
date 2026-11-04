# 04: Prove channel isolation against a live broker

**What to build:** The broker-gated suite proves the isolation end to end, not just through the injected seam. Deleting the queue behind one consumer must leave the connection generation untouched while that consumer comes back and drains the re-declared queue, and a second queue must keep consuming throughout. Killing the connection must still rebuild the generation exactly once and restart every consumer on it. Repeated queue deletion past the escalation limit must produce exactly one connection rebuild. Logging stays bounded to one warning per in-place attempt and one error per escalation.

**Blocked by:** 01, 02, 03.

**Status:** done

- [x] Deleting one consumer's queue leaves the generation pointer unchanged, and that consumer drains the re-created queue afterwards.
- [x] A message published to a second queue is consumed across the affected consumer's restart, with no gap.
- [x] Killing the connection rebuilds the generation exactly once and every consumer resumes on it.
- [x] Deleting one consumer's queue past the escalation limit produces exactly one connection rebuild.
- [x] The log stream carries one warning per in-place attempt and one error per escalation, with no repeating line while waiting.
- [x] The suite runs with `MISO_RABBIT_BROKER_TESTS=1` and skips cleanly without it.