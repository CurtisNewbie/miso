# 01: Client request and byte counters

**What to build:** A miso client that has been opted in with `EnableMetrics(name)` reports three counters for every request it sends. The counters record attempts, request body bytes, and response body bytes, all labeled with the client name, the request target, and the emitting app. A direct URL target labels the host as host and port. A service discovery target labels it as the resolved service name. A client that was never opted in emits nothing.

**Blocked by:** None (can start immediately)

**Status:** done

- [x] `EnableMetrics(clientName)` chains on a client and leaves clients that never call it emitting nothing.
- [x] One request through a metered client increments `miso_client_requests_total` and both byte families once, labeled with the given client name, the target, and the `app.name` property of the process.
- [x] A direct URL target labels `host` with host and port.
- [x] A service discovery target labels `host` with the resolved service name.
- [x] An empty client name produces an empty label value and does not fail the request.
- [x] A request that fails to connect still increments the attempt counter.
- [x] Two metered clients in one process work without a duplicate registration panic.
- [x] The shared default client's transport is the same value after a metered request.
- [x] A public `NewPromCounterVec(name, labels)` exists beside the existing metric constructors.
- [x] Tests drive the public client API against a test HTTP server and assert on gathered metric values, never on wrapper types or internal fields.
- [x] `go test -race -run TestClient ./miso/` passes, and the package suite shows no new failures against the base branch.