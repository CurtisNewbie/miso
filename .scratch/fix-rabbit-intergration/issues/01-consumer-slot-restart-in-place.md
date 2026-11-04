# 01: Restart a dead consumer channel in place

**What to build:** When a consumer's channel dies, that consumer restarts on the connection the client already holds, while every other consumer and the publisher pool keep running. The slot that owns the channel handles a channel close and a broker cancel the same way: brief backoff, open a channel, re-declare the queue and the bindings that target it, apply the quality of service, subscribe again, watch the new channel. A generation being retired closes its done signal before it closes the connection, so no slot opens a channel on a connection already being replaced, and the supervisor waits a bounded time for its slots to exit.

Every generation holds one slot per registered listener. The slot loop takes its open, start and watch operations as injected functions, so the restart cycle is covered without a broker.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] One consumer channel death opens exactly one replacement channel on the same connection.
- [ ] A consumer channel death does not rebuild the connection.
- [ ] The restart re-declares the listener's queue and the bindings that target it before subscribing.
- [ ] Retiring a generation mid-backoff opens no channel, escalates nothing, and leaves no slot goroutine running.
- [ ] The broker-free suite covers the restart cycle through the injected seam.