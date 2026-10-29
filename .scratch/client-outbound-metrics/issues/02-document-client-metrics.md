# 02: Document client metrics in the miso skill reference

**What to build:** The miso skill's Prometheus reference gains a section that describes the client metrics, so a later reader finds the feature without reading the source.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] The reference names `miso_client_requests_total`, `miso_client_request_bytes_total`, and `miso_client_response_bytes_total`.
- [ ] It states the three labels, `client_name`, `host`, and `app`, and what value each carries.
- [ ] It shows the query that derives QPS from the attempt counter with `rate()`.
- [ ] It states that enabling is per client and off by default.
- [ ] It states that byte counts are body bytes after transport decompression, and that a response body the caller never reads is not counted.
- [ ] It states the public `NewPromCounterVec` constructor alongside the other metric constructors.