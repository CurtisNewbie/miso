# Handoff: miso RabbitMQ reconnection and channel isolation

Read this, then wait for the user's instructions. Do not ask clarifying questions on the basis of this document.

## Goal

Two objectives.

1. Explain why `vfm` lost its RabbitMQ client during the incident of 2026-11-03 around 23:22 (UTC+8): vfm absent from the broker's connection list, one queue at roughly 176k ready with `Deliver` near zero for over an hour, logs repeating `Failed to start Consumer 'event.bus.vfm.file.moved', restarting, rabbitmq connection is closed` once a second, recovery only by restarting the process.
2. Make the miso RabbitMQ lifecycle resilient enough that this class of failure cannot recur, then isolate channel-level failures from connection-level ones.

Objective 1 stays open on the incident's root cause. The evidence that would settle it is gone, so the code fix is not claimed as that root cause. See the section on the incident below.

## Repo state

| Location | State |
|---|---|
| `/Users/photon/dev/git/miso` | Branch `fix-rabbitit`. `eed839f` commits the supervisor rewrite. The current working tree adds the fixes listed under uncommitted work. Nothing tagged. |
| Dependent services | Not touched in this session. `vfm` lives at `/Users/photon/dev/git/vfm`, the monorepo at `/Users/photon/dev/git/moon-monorepo`. `moon-monorepo` is clean. |

Read the diff instead of trusting prose: `git -C /Users/photon/dev/git/miso log -1 --stat` and `git -C /Users/photon/dev/git/miso diff`.

The reproducer worktree and the logs that produced the defect verdicts no longer exist on this machine. Do not look for them.

## What `eed839f` does

Single-owner connection lifecycle. One supervisor goroutine owns a generation: the connection, a cap-1 close listener, a cap-1 coalescing signal, the publisher pool and its start time. On a connection close or any channel death it retires the generation with a once guard, backs off from 1 second to 30 seconds, resets the backoff only after a generation stayed up 60 seconds, then dials, re-declares every component and re-subscribes every listener.

`establish` builds a local generation with an installed flag, so every error path retires a half-built connection instead of leaking the socket. Notifications are registered inside the dial function, so no path creates a connection without a receiver for its close event. Both the connection listener and the per-channel listeners are buffered, because amqp091 delivers those notifications while holding a lock, and an unread unbuffered listener wedges shutdown.

Publisher leasing carries the originating pool. A drained pool refills on demand through the generation's open function, and a lazily opened publisher channel is confirmed and watched like a bootstrap one.

## Uncommitted work in the tree

- `rabbitGeneration.retire()` normalizes `amqp.ErrClosed` to nil. An already-closed connection is the normal retire path, so every genuine reconnect used to log a bogus warning from the supervisor and from the half-built cleanup.
- The RabbitMQ test package is broker-free by default. A `requireRabbitBroker` gate skips every broker-backed test unless `MISO_RABBIT_BROKER_TESTS=1`. The five `TestIntegration*` functions and their helpers are restored and gated the same way, so they can be run on demand instead of sitting commented out.
- `.scratch/fix-rabbit-intergration/spec.md` writes the per-channel supervisor design: a channel restarts on its own connection, two consecutive failed restarts escalate to a connection rebuild, publisher channels stay pool-managed.

Verification: `go test ./middleware/rabbit/ ./util/pool/` passes without a broker, and the broker-backed tests skip. Five packages elsewhere in the repo fail for environmental reasons only: `middleware/redis` and `middleware/task` need a local Redis, `miso/docgen` trips a `go.work` version mismatch, `util/osutil` uses a cwd-relative path, and `miso` fails `TestWrapMisoErr`. None of them touch RabbitMQ.

## The vfm incident

The reproduced defect on the old code was a leak, not a reconnect stall. `NotifyClose` was unbuffered and abandoned on error paths, so the abandoned receiver wedged the connection's shutdown with a goroutine stuck on a channel send and a client socket left in the closed state. Separately, the old retry loop kept retrying against a captured dead connection, which is the once-a-second log spam.

That set of defects does not explain vfm disappearing from the broker's connection list with no recovery. The artifacts that would answer the remaining questions are gone: vfm's own logs after 23:22, the container state, whether the process restarted or was killed. Treat the root cause as unknown. The code fix stands on its own merits: unbuffered listeners, the socket leak, a data race on the publisher field, log spam.

## Decisions still pending

- Version and tag for miso. The latest tag is `v0.4.24`.
- The module bump and redeploy of `vfm` and the other services once it is tagged.
- Whether to implement the per-channel supervisor spec, and whether to write its tickets.

## Running things

```bash
go test ./middleware/rabbit/ ./util/pool/ -count=1                  # broker-free, default
MISO_RABBIT_BROKER_TESTS=1 go test ./middleware/rabbit/ -count=1    # needs localhost:5672
```

No broker and no Redis run on this machine right now; port 5672 and the management API on 15672 are both closed, and 6379 refuses connections. Dialing a missing broker takes about 5 seconds, which stretches the broker-backed tests and hides backoff timing.

`rabbitmqctl` is not installed. Use the management API on `localhost:15672` when a broker is up. Delete leftover `misorepro-*` and `it-*` queues and stray connections after a run.

## Files worth reading first

- `middleware/rabbit/rabbitmq.go`: `superviseRabbit`, `establish`, `dialRabbit`, `watchChannel`, `borrowPubChan`, `retire`.
- `middleware/rabbit/rabbitmq_test.go`: supervisor and publisher unit tests first, then the gated integration block at the end.
- `.scratch/fix-rabbit-intergration/spec.md`: the channel isolation design.

## Skills

- `miso` (`skills/miso/SKILL.md`) before writing miso Go code, plus `skills/miso/references/code-review-checklist.md` when reviewing.
- `dev-flow` for specs and tickets under `.scratch/<slug>/`.