# Code Review Checklist

Read this when reviewing miso code — a diff, a PR, your own changes before handing them over. Each entry is a mistake that recurs here; the rule behind it lives in the linked reference.

**Completion criterion:** every entry walked against the diff, each marked *found at `<file:line>`* or *clear*. A review that reports "looks fine" without walking the list is not a review.

**Table of Contents:**
- Rail-as-ctx
- fmt.Errorf / errors.New
- return ErrFileNotFound
- GetRedis().Set(...)
- mysql.GetMySQL().Raw(...)
- time.Time
- Panic in business code

## Rail-as-ctx

`Rail` satisfies `context.Context` (`flow/ctx.go:21`), so `f(rail)` compiles wherever a `ctx context.Context` is wanted. It still is the wrong argument: the framework draws the boundary deliberately — `miso.Rail` where logging and tracing happen, `rail.Context()` at stdlib and middleware boundaries. Converting at the call site keeps that boundary visible; handing the Rail over smuggles the logging wrapper into code typed to take a plain context.

```go
// Wrong — Rail passed as ctx
err := client.Get(rail, key)
doWork(rail)

// Right
err := client.Get(rail.Context(), key)
doWork(rail.Context())
```

## fmt.Errorf / errors.New

Never build errors with the standard library. `errs` carries the stack trace and error code; `fmt.Errorf` and `errors.New` produce opaque strings the web layer cannot render as a structured response.

```go
// Wrong
return fmt.Errorf("failed to load user %d: %w", id, err)
return errors.New("not found")

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

`GetRedis()` hands back the raw `*redis.Client`. Check whether the package-level API already covers the command before reaching past it: `redis.Set`, `redis.Get`, `redis.HIncrBy`, `redis.BRPopAny` and the `*Json` variants convert `rail.Context()` and wrap errors in `errs` for you, which a bare client call leaves to every caller.

```go
// Wrong — raw client reached for before checking the wrapper
return redis.GetRedis().Set(ctx, key, val, ttl).Err()

// Right — redis.Set covers it
return redis.Set(rail, key, val, ttl)
```

`GetRedis()` is the escape hatch for a command the wrappers do not expose — and even there, pass `rail.Context()` and wrap the error. See [redis.md](redis.md).

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
