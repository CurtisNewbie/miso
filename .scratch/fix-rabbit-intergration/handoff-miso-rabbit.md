# Handoff — miso RabbitMQ reconnect fix + vfm disconnect investigation

Read this, then **wait for the user's instructions. Do not ask clarifying questions on the basis of this document.**

## Goal

Two objectives, in order:

1. **Explain and fix why `vfm` lost its RabbitMQ client entirely** during the incident of 2026-11-03 ~23:22 (UTC+8): vfm missing from the broker's connection list, a queue sitting at ~176k ready with `Deliver ≈ 0` for over an hour, its log spamming `Failed to start Consumer 'event.bus.vfm.file.moved', restarting, rabbitmq connection is closed` once a second, recovery only by process restart.
2. **Improve the miso framework's RabbitMQ lifecycle** so this class of failure cannot recur, then (as a separate follow-up) redesign channel-level failure isolation.

Objective 1 is **not closed**: the implemented fix is a defensible improvement, but the empirical repro **refuted** the causal story that motivated it. Do not treat the current diff as "the fix for the incident" until the gap in §4 is closed.

## 1. Repo state (uncommitted work)

| Location | State |
|---|---|
| `/Users/zhuangyongjie/dev/git/miso` (main) | **Uncommitted** fix, 4 files: `middleware/rabbit/rabbitmq.go`, `middleware/rabbit/rabbitmq_test.go`, `util/pool/pool.go`, `util/pool/pool_test.go`. Nothing committed, no tag. |
| `/Users/zhuangyongjie/dev/git/miso-bugverify` | Detached-HEAD worktree (`3480645`, pre-fix code) holding the reproducer `middleware/rabbit/rabbit_wedge_repro_test.go` (untracked) plus one uncommitted control edit (`rabbitmq.go:464`, chan cap 1 — restore with `git checkout -- middleware/rabbit/rabbitmq.go`). |
| `/Users/zhuangyongjie/dev/git/moon-monorepo` | Clean. `backend/mini-fstore` commit `43adac4` fixes an unrelated unzip temp-dir permission bug (0755 vs 0666). An earlier experiment that replaced the unzip listener with an empty one was **reverted**; `event.go` is back to `UnzipPipeline.Listen(1, OnUnzipFileEvent)`. |

Read the diff rather than trusting prose: `git -C /Users/zhuangyongjie/dev/git/miso diff`.

## 2. What the uncommitted miso fix does

Single-owner lifecycle: one supervisor goroutine owns a `rabbitGeneration` (conn + cap-1 connClose + cap-1 coalescing signal + publisher pool + `startedAt`); on connection close or any channel death it retires the generation (`once`-guarded `conn.Close()`), backs off 1s→30s (reset only after a 60s-stable generation), re-dials, re-declares and re-`Consume`s every listener. `rabbitManagedChannel`/`retryStart`/`channel()`/`onNotifyCancelClose()` are gone, replaced by `startManagedChannel` + non-deciding `watchChannel`. `NotifyClose` is cap-1 and registered inside `dialRabbit`. `establish` uses a local `gen` + `installed` flag so every error path retires the half-built connection (a real leak in the first draft, found by the integration tests). Publisher leasing carries the originating pool; drained pools refill lazily through `gen.publisher.open`; `NewEphPool` finally assigns its `filterFunc`; `RabbitConnected()` checks `IsClosed()`.

Broker-free tests: `TestSupervisor*`, `TestPublishChannel*`. Live-broker tests: five `TestIntegration*` funcs, **commented out** at `middleware/rabbit/rabbitmq_test.go:433` (all five passed before being disabled). The package's older broker tests (`TestInitClient`, `TestPublishMessage`, `TestDeclareEventBus`, …) still dial the broker, so the package as a whole is **not** broker-free yet.

## 3. Evidence (reproducer verdicts, old code)

Reproducer logs: `/tmp/final_A_old.log`, `/tmp/final_B_old.log`, `/tmp/final_C_old.log`; cap-1 controls: `/tmp/final_A_control.log`, `/tmp/final_B_control.log`.

- **CONFIRMED** — `rabbitmq.go:464` registered an unbuffered `NotifyClose` and error paths abandoned it; the abandoned receiver permanently wedges `amqp091.Connection.shutdown` (goroutine stuck in `chan send` at `connection.go:609`, client FD leaked in `(CLOSED)` state). Cap-1 control: 0 wedges, FD released. Cause attribution solid.
- **REFUTED** — `conn.Close()` never returning: it returns `nil` on a healthy conn and `Exception (504)` after the wedge (`Close()` short-circuits on `IsClosed()`); 0/8 race offsets hung.
- **REFUTED** — "only recovery is a process restart": with a successful re-declare, the old module-level reconnect rebuilt consumers on a new connection and messages flowed again (Test C). Consumers stayed dead in Test B only because that test's own re-declare returned 406 — identical with a buffered listener.
- **CONFIRMED** — `retryStart` loops forever on the captured `*amqp.Connection` (31 retries/14s), i.e. log spam + leaked goroutines, not a dead client.
- **CONFIRMED (incidental)** — `-race` flags a genuine data race on `m.publisher` (`:529` write vs `:644` read) in the old code; the new code removes it.

## 4. The open question (why nothing was committed)

The reproduced defect is a socket/goroutine leak, **not** a permanent reconnect stall. So it does not by itself explain vfm's total disappearance from the broker with no recovery. To close objective 1, establish what actually happened to vfm at 23:22:

- Did the broker refuse vfm's reconnect (dial error), or did the re-declare fail (e.g. 406/404)?
- Did the vfm process restart, get OOM-killed, or stay up?
- Did the module-level reconnect goroutine run at all, and if not, where was it blocked?

The deciding artifact is vfm's own logs after 23:22. The `es-access` skill is **not configured** here (`~/.esaccess/es.json` missing); it needs the Kibana base URL from the user. Alternatively, container logs/state where the services run.

## 5. Decisions the user has not made

- Whether to commit the current miso fix at all, given §4 (it fixes real defects — buffered-listener invariant, leak, `m.publisher` race, log spam — but is not proven to be the incident's cause).
- Version/tag for miso if committed, then the module bump + redeploy of vfm (and other services).
- The follow-up design the user leaned toward: keep the connection-level supervisor, add a **per-channel supervisor** that restarts only the dead channel (AMQP-correct isolation), escalating to a full connection rebuild after two failed in-place restarts. The user asked for a spec for this **separately from the code commit**; the spec was never written because the work stopped.
- Known small defect still unfixed: `retire()` on an already-closed connection returns `ErrClosed`, which the supervisor wrapper logs as `WARN Failed to close RabbitMQ connection` on every genuine reconnect (`rabbitmq.go` ~:558).

## 6. Useful facts

- Local RabbitMQ: `localhost:5672`, default local guest credentials, management API on `:15672`; `rabbitmqctl` is **not** installed here — use the management API. Dials against this broker take ~5s, which slows reconnect tests and masks backoff timing.
- `basic.cancel` (e.g. queue deleted under a consumer) → `watchChannel` cancel branch → `signal` → full generation rebuild; observed live (`consumer-cancelled=true`).
- Delete `misorepro-*` queues / stray connections if any repro leftovers are found (`/api/queues`, `/api/connections`).

## 7. Suggested skills to load

- `miso` (repo-local `skills/miso/SKILL.md`) — mandatory before writing miso Go code; plus `skills/miso/references/code-review-checklist.md` when reviewing.
- `dev-flow` — when writing the per-channel spec/tickets (`.scratch/<slug>/` in the miso repo).
- `es-access` — only if the user provides a Kibana URL for vfm's incident logs.
- `offloaded-skill` — to load `handoff` or other offloaded skills by name.

## 8. Files worth reading first

- `/Users/zhuangyongjie/dev/git/miso/middleware/rabbit/rabbitmq.go` (the uncommitted rewrite; `superviseRabbit`, `establish`, `dialRabbit`, `watchChannel`, `borrowPubChan`).
- `/Users/zhuangyongjie/dev/git/miso/middleware/rabbit/rabbitmq_test.go:433` — the commented integration block, including the leak assertion and the broker-path recording.
- `/Users/zhuangyongjie/dev/git/miso-bugverify/middleware/rabbit/rabbit_wedge_repro_test.go` — the reproducer that produced §3.
