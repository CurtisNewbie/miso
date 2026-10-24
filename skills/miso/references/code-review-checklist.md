# Code Review Checklist

Read this when reviewing miso code — a diff, a PR, your own changes before handing them over. Each entry is a mistake that recurs here; the rule behind it lives in the linked reference.

**Completion criterion:** every entry walked against the diff, each marked *found at `<file:line>`* or *clear*. A review that reports "looks fine" without walking the list is not a review.

**Table of Contents:**
- rail.Context()
- fmt.Errorf
- return ErrFileNotFound
- GetRedis().Set(...)
- mysql.GetMySQL().Raw(...)
- time.Time
- Panic in business code

## rail.Context()

`Rail` satisfies `context.Context` (`flow/ctx.go:21`), so `f(rail)` is the right argument wherever a `ctx context.Context` is wanted. `Deadline`, `Done`, `Err` and `Value` all delegate to the wrapped context, so passing the Rail loses nothing and keeps the call visibly inside the traced request. Do not unwrap it with `rail.Context()` just to satisfy a `context.Context` parameter — that turns a traced call into an anonymous one at every call site for no gain.

```go
// Wrong — needless unwrap when a ctx context.Context is wanted
err := client.Get(rail.Context(), key)
doWork(rail.Context())

// Right — Rail is a context.Context
err := client.Get(rail, key)
doWork(rail)
```

## fmt.Errorf

Never build returned errors with `fmt.Errorf`. `errs` carries the stack trace and error code; `fmt.Errorf` produces an opaque string the web layer cannot render as a structured response. `errors.New` is not the problem — it appears only for plain static errors — what matters is not hand-formatting a returned error and losing the wrap.

```go
// Wrong
return fmt.Errorf("failed to load user %d: %w", id, err)

// Right
return errs.Wrapf(err, "failed to load user %d", id)
return errs.NewErrf("not found")
return errs.NewErrfCode("USER_NOT_FOUND", "user %d not found", id)
```

The only place `%w` belongs is inside `errs` itself. Details: [error-handling.md](error-handling.md).

## return ErrFileNotFound

An `errs` value declared as a package var is a template, not an error to ship. Its stack trace was captured where the var was declared, and `Wrap`/`Wrapf` does not repair that — the innermost `*MisoErr` keeps it. `.New()` copies the error and re-captures the stack at the return site.

```go
var ErrFileNotFound = errs.NewErrfCode("FILE_NOT_FOUND", "File is not found")

// Wrong — stack points at the var's declaration
return ErrFileNotFound

// Right
return ErrFileNotFound.New()
```

## GetRedis().Set(...)

`GetRedis()` hands back the raw `*redis.Client`. Check whether the package-level API already covers the command before reaching past it: `redis.Set`, `redis.Get`, `redis.HIncrBy`, `redis.BRPopAny` and the `*Json` variants wrap errors in `errs` for you, which a bare client call leaves to every caller.

```go
// Wrong — raw client reached for before checking the wrapper
return redis.GetRedis().Set(ctx, key, val, ttl).Err()

// Right — redis.Set covers it
return redis.Set(rail, key, val, ttl)
```

`GetRedis()` is the escape hatch for a command the wrappers do not expose — and even there, pass `rail` and wrap the error. See [redis.md](redis.md).

## mysql.GetMySQL().Raw(...)

The same mistake one layer down: `dbquery.NewQuery(rail, mysql.GetMySQL())` wraps the same `*gorm.DB` and keeps what a bare `GetMySQL()` call drops — the statement runs on `rail.Context()`, SQL logging carries the trace, and errors come back wrapped in `errs`. Raw SQL is no excuse for skipping it: the builder does `.Raw(...)` too.

```go
// Wrong — bare gorm, no rail on the statement, unwrapped error
mysql.GetMySQL().Raw("select count(id) from file where file_id in ? and status = 'NORMAL'", fileIds).Scan(&cnt)

// Right — builder covers it
cnt, err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table(TableFile).
    In("file_id", fileIds).
    Eq("status", "NORMAL").
    Count()
if err != nil {
    return errs.Wrapf(err, "failed to count files")
}
```

`.Raw(...)` earns its place only for SQL the builder cannot express — multi-table joins, window functions, subqueries — and even then chained onto a `dbquery.NewQuery` so the trace and error wrapping survive. See [database.md](database.md#raw-sql).

## time.Time

`atom.Time` in models, structs, and JSON boundaries — `time.Time` loses miso's JSON, SQL, and parsing behavior. See [util.md](util.md).

## Panic in business code

Return an `errs` error. `panic` earns its place only at startup — misconfiguration and `regexp.MustCompile`. See [golang-coding-practice.md](golang-coding-practice.md).

## Growing this list

A mistake that shows up twice in review belongs here: one entry, wrong/right pair, pointer to the reference that owns the rule.
