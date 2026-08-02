# Util Packages

Shared utility packages under `github.com/curtisnewbie/miso/util/`. These are framework-level helpers used across miso itself — prefer them over hand-rolled equivalents.

## Time (atom)

**Rule: always use `atom.Time` instead of `time.Time`** in models, structs, and JSON boundaries. It wraps `time.Time` with miso-friendly JSON, SQL, and parsing behavior.

**Package:** `github.com/curtisnewbie/miso/util/atom`

```go
import "github.com/curtisnewbie/miso/util/atom"

now := atom.Now()          // local time
nowUTC := atom.NowUTC()    // UTC
nowIn := atom.NowIn(8)     // offset hours

t := atom.WrapTime(time.Now()) // wrap an existing time.Time

// Parse from various formats (string, int64 epoch sec/milli, time.Time...)
t, err := atom.ParseTime("2024-01-02 15:04:05")
t = atom.MayParseTime(1700000000000) // ignore parse errors, return zero Time

// Time window helpers
start := t.StartOfDay()
end := t.EndOfDay()
weekStart := t.StartOfWeek(time.Monday)
lastMon := t.LastWeekday(time.Monday)
monthStart := t.StartOfMonth()

// Formatting
s := t.FormatStd()        // 2006-01-02 15:04:05
s = t.FormatClassic()     // 2006-01-02 15:04:05.999999
s = t.FormatDate()        // 2006-01-02
s = t.FormatRFC3339()
```

**Why atom.Time over time.Time:**

- **JSON marshals as epoch millis by default** (framework convention), overridable via `atom.SetTimeMarshalFormat("2006-01-02 15:04:05")`
- Implements `sql.Scanner`/`driver.Valuer` — safe in GORM; accepts RFC3339, `2006-01-02 15:04:05.999999`, `2006-01-02`, and auto-detects epoch **sec** (≤ 9999999999) vs epoch **millis** (> 9999999999)
- Unwraps to plain `time.Time` via `t.Unwrap()` when needed

## JSON

jsoniter-based JSON helpers, prefer over `encoding/json`.

**Package:** `github.com/curtisnewbie/miso/util/json`

```go
// Parse
var user User
err := json.ParseJson(body, &user)
err = json.ParseJsonAs[User](body)      // generic
user, err := json.SParseJsonAs[User](s) // from string

// Write (Marshal/Unmarshal are aliases)
b, err := json.WriteJson(user)
s, err := json.SWriteJson(user)
s = json.TrySWriteJson(user) // "" on failure, no error

// Streams / validation
err = json.DecodeJson(reader, &user)
err = json.EncodeJson(writer, user)
ok := json.IsValidJson(b)

// Indentation
s, err := json.SWriteIndent(user) // 2-space
s = json.Indent(b)                // tab
```

**Notable:** untagged exported struct fields are auto-lowercased (first rune) via the default naming strategy; `LowercaseNamingStrategy` is a ready-made strategy. Error messages include the raw body for debugging.

## strutil

String helpers.

**Package:** `github.com/curtisnewbie/miso/util/strutil`

```go
// Named placeholder formatting: "${key}"
s := strutil.NamedSprintf("Hello ${name}, you are ${age}", map[string]any{"name": "Tom", "age": 30})
s = strutil.NamedSprintfv("Hello ${name}", user)         // struct fields
s = strutil.NamedSprintfkv("${a}-${b}", "a", "1", "b", "2") // k,v,k,v pairs

// Common helpers
b := strutil.IsBlankStr("  ")          // true
s = strutil.TrimSpace(" a ", ',')      // trim + extra runes
parts := strutil.SplitStr("a, b,,c", ",") // split + trim + drop empty
s = strutil.CamelCase("user_name")     // "userName" (camelCase)
s = strutil.MaxLenStr("long string", 4) // rune-based truncation
s = strutil.PadNum(7, 4)               // "0007"
s = strutil.QuoteStr(`"quoted"`)

// Path/API pattern matching (doublestar: * one segment, ** multi)
ok := strutil.MatchPath("/api/**", "/api/v1/users") // true
ok = strutil.MatchApiPattern("GET", "/api/users", "GET:/api/**")

// Case-insensitive prefix/suffix
s, ok := strutil.CutPrefixIgnoreCase("HELLO world", "hello")

// Zero-copy conversions (read-only! mutating the byte slice panics)
b := strutil.UnsafeStr2Byt(s)
s2 := strutil.UnsafeByt2Str(b)
```

## slutil

Generic slice toolkit.

**Package:** `github.com/curtisnewbie/miso/util/slutil`

```go
// Filter: in-place (returns subslice of the same array) — use CopyFilter to copy
kept := slutil.Filter(users, func(u User) bool { return u.Age > 18 })
kept = slutil.CopyFilter(users, func(u User) bool { return u.Age > 18 })

// Map
names := slutil.MapTo(users, func(u User) string { return u.Name })
ids, err := slutil.MapToErr(users, func(u User) (string, error) { ... }) // aborts on error

// Distinct: note Distinct is unordered; DistinctStrBy preserves order
ids := slutil.Distinct([]string{"a", "b", "a"})
unique := slutil.DistinctStrBy(users, func(u User) string { return u.ID })

// Lookup / aggregation
first, ok := slutil.First(users)                              // (zero value, false) if empty
u, ok := slutil.FirstMatch(users, func(u User) bool { ... })
byID := slutil.MergeMap(users, func(u User) string { return u.ID }) // last wins

// Sorting (stable, in place; supports string/int/float types)
slutil.Sort(ids)
slutil.SortMapped(users, func(u User) int { return u.Age })

// Batch processing
err := slutil.SplitSubSlices(ids, 100, func(batch []string) error { ... })

// Misc
slutil.Remove(items, 2, 5)
slutil.Prepend(items, first)
flattened := slutil.Flatten([][]string{...})
safe := slutil.SyncSlice[int](0) // concurrent-safe slice with ForEach/Copy
```

## retry

Retry helpers with configurable attempt counts and backoff.

**Package:** `github.com/curtisnewbie/miso/util/retry`

```go
// Up to retryCount+1 attempts (default: retry on any error)
v, err := retry.GetOne(3, func() (string, error) { return call() })

// Retry only on specific errors
v, err = retry.GetOne(3, doWork, func(err error) bool { return errors.Is(err, ErrRetryable) })

// Explicit backoff schedule: sleep backoff[i] after failed attempt i
// (keeps retrying without sleep once the schedule is exhausted)
v, err = retry.GetOneWithBackoff([]time.Duration{100 * time.Millisecond, 500 * time.Millisecond, time.Second}, doWork)

// Dynamic backoff — retries indefinitely; i is the 1-based attempt index
v, err = retry.GetOneDyn(doWork, func(i int, err error) (time.Duration, bool) {
    return time.Duration(i) * time.Second, true
})

// Retry decision can inspect the result too
v, err = retry.GetOneCond(3, doWork, func(v T, err error) bool { return err != nil || !v.Ready })
```

## randutil

Random string/number generation (note: the `testutil` package has no random helpers — these live here).

**Package:** `github.com/curtisnewbie/miso/util/randutil`

```go
s := randutil.RandStr(16)                 // [a-zA-Z0-9]
n := randutil.RandNum(6)                  // digits only
s = randutil.RandUpperAlpha(8)
s = randutil.RandLowerAlphaNumeric(12)
s = randutil.RandLowerAlphaNumeric16()    // fixed 16, pooled

// High-entropy base64 (crypto/rand), output length ≈ len
s = randutil.ERand(16)

// ID-like strings: prefix + random
s = randutil.GenNo("order-")     // prefix + 35 random chars
s = randutil.GenNoL("order-", 12)

// Random pick (incl. weighted)
v := randutil.RandPick(items)
v = randutil.WeightedRandPick(weightedItems) // items implement GetWeight() float64
```

## ID Generation

**Package:** `github.com/curtisnewbie/miso/util/snowflake` and `github.com/curtisnewbie/miso/util/idutil`

```go
import "github.com/curtisnewbie/miso/util/snowflake"
import "github.com/curtisnewbie/miso/util/idutil"

id := snowflake.Id()        // thread-safe, ts+seq + 6-digit machine code, ≤25 chars
id = snowflake.IdPrefix("usr_")
_ = snowflake.SetMachineCode(123) // 0..999999, zero-padded, random by default

id = idutil.New()  // "2" + ULID, 27 chars — sorts after snowflake ids
id = idutil.Id("evt_")
```

## async

Goroutine/async utilities: futures, pools, panic-safe runners.

**Package:** `github.com/curtisnewbie/miso/util/async`

```go
// Future
fut := async.Run(func() (User, error) { return loadUser() })
user, err := fut.Get()
user, err = fut.TimedGet(3000) // timeout in milliseconds, returns async.ErrGetTimeout
fut.Then(func(u User, err error) { ... })

// Fire-and-forget with rail error logging
async.Fire(rail, func() error { return cleanup() })

// Pool (prefer NewAsyncPool; NewCpuAsyncPool/NewIOAsyncPool/NewBoundedAsyncPool are deprecated)
pool := async.NewAsyncPool(100)
fut = async.Submit(pool, func() (User, error) { return loadUser() })

// Batch futures
await := async.NewAwaitFutures[User](pool)
await.SubmitAsync(func() (User, error) { ... })
users, err := await.AwaitResultAnyErr()

// Panic safety
err := async.CapturePanicErr(func() { ... })
async.PanicSafeRun(func() { ... })
err = async.PanicSafeRunErr(func() error { ... })

// Pool sizing helper: multi * GOMAXPROCS clamped to [min, max], max=-1 = unlimited
size := async.CalcPoolSize(12, 128, 1024)

// Misc: RunUntil (timeout with result), TickRunner (note: constructor is NewTickRuner),
// SignalOnce, BatchTask (parallel map-reduce over tasks), DoneWatcher
```

## osutil

Filesystem helpers.

**Package:** `github.com/curtisnewbie/miso/util/osutil`

```go
ok, err := osutil.FileExists("conf.yml")
ok = osutil.TryFileExists("conf.yml") // error-swallowing

data, err := osutil.ReadFileAll("data.json")
err = osutil.WriteFileStr("out.txt", "hello") // truncates
f, err := osutil.OpenRWFile("data.json")      // creates if absent by default
f, err = osutil.OpenAppendFile("log.txt")

err = osutil.MkdirAll("/tmp/a/b/c")
err = osutil.MkdirParentAll("/tmp/a/b/file.txt")

// Recursive walk, optionally filtered by suffix
files, err := osutil.WalkDir("./assets", ".go", ".md") // []WalkFsFile{Path, File}

// Temp files
path, err := osutil.NewTmpFilePath()
f, err := osutil.NewTmpFileWith(".json")
path, err = osutil.SaveTmpFile("/tmp/uploads", reader)

// Filename suffixes (ext without dot, case-insensitive)
osutil.FileAddSuffix("a.txt", "bak")        // a.bak.txt
osutil.FileCutSuffix("a.bak.txt", "bak")    // (a.txt, true)
osutil.FileReplaceSuffix("a.bak.txt", "zip")// a.zip

// Size units
osutil.KbUnit // 1024, also MbUnit/GbUnit
```

## testutil

Test helpers for locating test data and config files.

**Package:** `github.com/curtisnewbie/miso/util/testutil`

```go
// Walks up from cwd looking for a testdata/ dir (stops at go.mod root)
data, err := testutil.FindTestdataPath("fixtures/user.json")
data = testutil.FindTestdata(t, "fixtures/user.json") // t.Fatal on error

// Project-root-relative path via go.mod
confPath, err := testutil.FindTestConfPath("conf.yml")
```

## flags

CLI flag parsing with required-flag enforcement.

**Package:** `github.com/curtisnewbie/miso/util/flags`

```go
var (
    port    = flags.Int("port", 8080, "server port", true)      // required
    verbose = flags.Bool("verbose", false, "verbose output", false)
    names   = flags.StrSlice("name", "repeatable names", false)
)

func main() {
    flags.Parse() // os.Exit(2) if a required flag is missing
}
```

`BoolVal(name, value, usage, required)` requires an explicit `-flag true`/`false` argument (bare `-flag` errors). Usage text can be customized via `flags.WithDescription(s)`/`flags.WithExtra(s)`.

## excel

Excel read/write via excelize. **Both functions take `miso.Rail`.**

**Package:** `github.com/curtisnewbie/miso/util/excel`

```go
// Read: whole file loaded into memory
sheets, err := excel.ReadExcel(rail, "report.xlsx",
    excel.OverwriteMergeCell(), // write merge-cell content to every covered row
    excel.WithHyperlink(),      // append hyperlinks to cell text
)
for _, s := range sheets {
    for _, row := range s.Records {
        // row = []string
    }
}

// Write
err = excel.Write(rail, "out.xlsx", "Sheet1", [][]string{{"a", "b"}, {"1", "2"}})
```

## copyutil

Struct copying via jinzhu/copier (field-name matching).

**Package:** `github.com/curtisnewbie/miso/util/copyutil`

```go
// Note: Copy returns nothing — copier errors are logged internally
copyutil.Copy(from, &to)

to := copyutil.CopyNew[User](from) // allocates and returns a copy
```
