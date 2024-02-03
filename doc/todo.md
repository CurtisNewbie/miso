# TODO

- [x] Consul registration includes metadata.
- [x] Support customized server selection logic in client.go.
- [x] Implement AsyncPool.
- [x] Update consul server list in real time using watch.
- [x] Refactor consul.go, move service discovery related stuff to a discovery.go.
- [x] Filter consul services that are not `passing`.
- [x] Add an *almost* complete configuration example.
- [x] Support grouping the already grouped subpaths.
- [x] Finish RabbitMQ Doc.
- [x] ~~Import sourcegraph/conc when it's stable.~~ we can just use *miso.AsyncPool and *miso.AwaitFutures for now, should be enough.
- [x] ~~Provide a demo project.~~ Not really needed.
- [ ] Document service discovery and load balancing.
- [ ] Document MySQL and Redis client.
- [ ] Support nacos (maybe).
- [ ] Support named database connections (maybe).
- [ ] Support etcd (maybe).

