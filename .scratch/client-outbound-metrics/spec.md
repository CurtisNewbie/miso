# Client outbound metrics

Status: ready-for-agent

## Problem Statement

A miso service reports its own memory, goroutine count, and inbound traffic, but nothing about the HTTP requests it sends out. When a service calls four dependencies, the operator cannot tell which one takes most of the calls, which one moves the most data, or when a dependency's call volume changed.

The only way to see outbound traffic today is to read logs or add counters by hand at each call site. Hand-added counters drift. Names differ between services, label sets differ, and call sites get missed.

## Solution

Outbound request metrics become an opt-in behavior of the miso client. Calling `EnableMetrics` with a client name on a client turns on three counters for every request that client sends. The counters record attempts, request body bytes, and response body bytes, and all three carry the same three labels, so a dashboard can sum by client, by target, or by app.

QPS is computed from the attempt counter with `rate()`. The framework stores no rate of its own.

## User Stories

1. As a service author, I want to turn on request metrics for one client with a single chained call, so that I do not wrap every call site.
2. As a service author, I want metrics off by default, so that existing services emit nothing new until someone asks for it.
3. As a service author, I want to name the client, so that dashboards separate the clients a service runs.
4. As an operator, I want request counts per client and per target, so that I can compute QPS with `rate()` in a query.
5. As an operator, I want request body bytes per target, so that I can see how much data a service uploads.
6. As an operator, I want response body bytes per target, so that I can see how much data a service downloads.
7. As an operator, I want the target label to read as the service name when service discovery resolves it, so that all instances of that service aggregate under one name.
8. As an operator, I want the target label to read as host and port for a direct URL, so that I can still separate endpoints that share a host.
9. As an operator, I want every sample labeled with the emitting app, so that one dashboard can cover every app in an environment.
10. As a service author, I want failed attempts counted, so that the QPS line includes calls that never got a response.
11. As a service author, I want a redirect hop counted as its own request, so that the byte totals match the transfers that actually happened.
12. As a service author, I want metrics to work whichever order I chain the client options in, so that I do not have to remember a required order.
13. As a maintainer, I want the counters shared by every client, so that a second client with metrics does not panic on duplicate registration.
14. As a maintainer, I want enabling metrics to leave the shared default client untouched, so that unrelated code keeps its transport and connection pool.
15. As a service author, I want an empty client name to stay empty instead of failing, so that a quick enable call does not break a request path.
16. As a service author, I want a public counter vector constructor alongside the existing metric constructors, so that my own metrics can use the same helper style.
17. As a maintainer, I want the label set and the counting rules written down, so that cardinality expectations are explicit.
18. As a maintainer, I want the client metrics described in the miso skill reference, so that a later agent finds the feature without reading the source.

## Implementation Decisions

The feature covers outbound requests sent through the miso client. The server side reverse proxy is not instrumented.

Enabling is a chained builder method on the client, `EnableMetrics(clientName string) *Client`, named to match the existing `EnableServiceDiscovery` and `EnableTracing`. An empty client name is accepted and appears as an empty label value.

Three counters carry the labels.

| Metric | Counts |
| `miso_client_requests_total` | attempts, including connection failures, non-2xx responses, and each redirect hop |
| `miso_client_request_bytes_total` | request body bytes read by the HTTP transport |
| `miso_client_response_bytes_total` | response body bytes read by the caller |

The label set is the same on all three.

| Label | Value |
| `client_name` | the name passed to `EnableMetrics`, empty when none was given |
| `host` | the resolved service name when the client uses service discovery, otherwise the URL host including port |
| `app` | the `app.name` property of the emitting process |

The counters attach by wrapping the `http.Client` used for a request, at the moment the request is sent. Attaching at enable time is not enough. Service discovery and transport replacement both happen later in the chain, and the target is only known per request. Wrapping produces a shallow copy, so the shared default client, its transport, and its connection pool are never mutated.

The three counter families are package level and registered once at package initialization. Registration reuses an already registered collector of the same name instead of panicking, because the existing metric constructors panic on duplicate registration. A public `NewPromCounterVec(name string, labels []string)` joins the existing histogram, histogram vector, and counter constructors and backs these three families.

Every attempt is counted, with no status or error label. A request that fails before its body is fully read contributes only the bytes that were read, and a response body the caller never reads contributes nothing. A caller that streams a response to a file reads the whole body, so its byte count is complete.

Byte counts come from the body streams as the transport and caller read them. They are the payload after any decompression the transport performs, not the bytes on the wire, which matches the counters drone keeps for its proxy.

No duration or latency metric is added.

The miso skill reference for Prometheus metrics gains a section describing these three families, their labels, and the `rate()` query for QPS.

## Testing Decisions

A good test drives the public client API against an `httptest` server and asserts on what the Prometheus default registry reports. It never asserts on wrapper types, field names, or other internals.

The single seam is the client API plus the default gatherer. Tests set up a server, send one or more requests through a client, gather the default registry, and compare the three families against their values before the requests.

Cases in the seam:

- A POST with a request body and a response body increments all three families, with the expected client name, host, and app labels.
- A bodyless GET increments the attempt and response byte counters and leaves the request byte counter alone.
- A client that resolves through service discovery reports the service name in `host`.
- A client on a direct URL reports host and port in `host`.
- An empty client name reports an empty label value.
- A connection to a closed port still increments the attempt counter.
- A client without `EnableMetrics` emits nothing.
- Two metered clients that both enable metrics work, and neither panics on registration.
- The default client's transport is the same value after a metered request.
- A redirect counts one request per hop, and the intermediate response body counts toward response bytes.
- A gzip response counts decompressed bytes, not compressed ones.

Tests read each counter as a delta against its value before the requests, since the package level counters persist across tests in the same process. Prior art is the existing in-package client and proxy tests that run against `httptest` servers. No existing test reads the default gatherer, so its helpers are new to this file.

## Out of Scope

- Duration and latency metrics for outbound calls.
- Instrumentation of the server side reverse proxy.
- A global configuration switch. Enabling stays per client.
- Status and error labels, success rates, and per-response-code breakdowns.
- Bytes on the wire, including headers, TLS overhead, and pre-decompression sizes.
- Transport-internal retries. When the HTTP transport rewinds a request and retries, that attempt never reaches the metrics transport, so neither the attempt nor the replayed request bytes are counted.
- The package level `SendGet` and `SendPost` helpers and any `http.Client` not built through the miso client.
- Recording rules, dashboards, and alert definitions.
- Turning metrics on for existing call sites. The change adds the capability only.

## Further Notes

The reference shape is drone's `proxymetrics` package, which counts proxy payload bytes by wrapping the transport. This design keeps that wrapping technique and replaces the single proxy wide counter with per target and per client labels.

Cardinality is bounded by the number of clients a service enables and the number of targets those clients call. The client name is expected to be a small, stable set chosen in code, not a value built per request.