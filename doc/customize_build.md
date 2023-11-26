# Customize Build

The following components can be excluded from the build by adding following tags:

- MySQL: `excl_mysql`
- Consul: `excl_consul`
- SQLite: `excl_sqlite`

E.g.,

```
go build -tags=excl_sqlite
```