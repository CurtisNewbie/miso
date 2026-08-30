# Testing

Writing tests that exercise the full miso framework (config, database, middleware) instead of mocking everything.

**Table of Contents:**
- PrepareTestEnv
- Test Configuration (conf.yml)
- Importing Middleware Packages
- InTestEnv
- Database Testing
- Test Data Files (testutil)

## PrepareTestEnv

`miso.PrepareTestEnv(t)` bootstraps the framework inside a Go test — config loading, logging, and all registered component bootstrap callbacks (database, redis, etc.) run exactly as they would during server startup, but **without starting the web server**.

```go
import (
    "testing"

    "github.com/curtisnewbie/miso"
)

func TestSomething(t *testing.T) {
    rail := miso.PrepareTestEnv(t)

    // framework is fully bootstrapped here:
    // config loaded, mysql/redis/etc. connected, callbacks executed
}
```

What it does:

- Sets the test-env flag (`app.test-env`), which disables the web server bootstrap
- Loads config from `conf.yml` (see below)
- Configures logging
- Runs pre-server bootstrap, component bootstrap, and post-server bootstrap callbacks
- Marks the app as fully bootstrapped
- Returns a `Rail` you can use for logging and tracing

Call it once per test (or in a `TestMain`/setup helper if you want to share the bootstrapped env across tests). It is not free — each call re-runs the full bootstrap, so prefer bootstrapping once per package when tests share the same config.

## Test Configuration (conf.yml)

`PrepareTestEnv` looks for `conf.yml` by walking up from the current working directory until it finds the project root (`go.mod`). Put a `conf.yml` at the project root (or in a directory above the test file) with the config your tests need:

```yaml
mysql:
  enabled: true
  user: root
  password: secret
  database: mydb_test
  host: localhost
  port: 3306
```

If no `conf.yml` is found, it logs a warning and continues with defaults — components that require config (e.g., mysql) will fail to bootstrap.

## Importing Middleware Packages

Middleware packages register their bootstrap callbacks via `init()`. **Import them in the test file** (or in the package under test) so they are registered before `PrepareTestEnv` runs:

```go
import (
    "testing"

    "github.com/curtisnewbie/miso"
    _ "github.com/curtisnewbie/miso/middleware/mysql" // registers mysql bootstrap
    _ "github.com/curtisnewbie/miso/middleware/redis" // registers redis bootstrap
)
```

If a middleware package is not imported, its component never bootstraps and `mysql.GetMySQL()` / `redis.GetRedis()` will return nil or fail.

## InTestEnv

`miso.InTestEnv()` reports whether the current process was bootstrapped by `PrepareTestEnv`. Useful for code that should behave differently under test (e.g., skip external side effects):

```go
if miso.InTestEnv() {
    // skip sending real notifications
}
```

## Database Testing

After `PrepareTestEnv`, the database middleware is connected and usable via the normal APIs:

```go
db := mysql.GetMySQL() // *gorm.DB
dbquery.NewQuery(rail, db).Table("users").Where("id = ?", id).Scan(&user)
```

### Transaction + Rollback for Cleanup

When a test writes data, an easy way to keep the database clean is to run the test inside a transaction and roll it back at the end — no manual DELETE cleanup needed.

```go
func TestCreateUser(t *testing.T) {
    rail := miso.PrepareTestEnv(t)

    tx := mysql.GetMySQL().Begin()
    defer tx.Rollback() // discard everything the test wrote

    // run the code under test against the transaction
    err := CreateUser(rail, tx, User{Name: "alice"})
    if err != nil {
        t.Fatal(err)
    }

    // assert within the transaction
    var u User
    if err := tx.Where("name = ?", "alice").First(&u).Error; err != nil {
        t.Fatal(err)
    }
}
```

**Important:** the rollback only covers writes that go through the same transaction. The code under test must accept the `*gorm.DB` (or `*dbquery.Query`) as a parameter and use it — if it calls `mysql.GetMySQL()` internally, its writes bypass the transaction and will not be rolled back. Design functions to take the db handle as a parameter to make them testable this way.

For code that uses `dbquery`, the same pattern works with `dbquery.RunTransaction` or by passing the tx's `*gorm.DB`:

```go
err := dbquery.RunTransaction(rail, mysql.GetMySQL(), func(qry func() *dbquery.Query) error {
    // qry() returns a *dbquery.Query bound to the transaction
    _, err := qry().Table("users").Create(&user)
    return err
})
```

### Alternative: Dedicated Test Database

If the code under test cannot take a db handle (e.g., it calls `mysql.GetMySQL()` directly), use a dedicated test database (e.g., `mydb_test`) and clean up with explicit DELETEs or by truncating tables between tests. The transaction approach is preferable when possible because it needs no cleanup logic and leaves no residue on failure.

## Test Data Files (testutil)

`github.com/curtisnewbie/miso/util/testutil` locates test fixtures and config files relative to the project root:

```go
import "github.com/curtisnewbie/miso/util/testutil"

// walks up from cwd looking for a testdata/ dir (stops at go.mod root)
data, err := testutil.FindTestdataPath("fixtures/user.json")
data = testutil.FindTestdata(t, "fixtures/user.json") // t.Fatal on error

// project-root-relative path via go.mod
confPath, err := testutil.FindTestConfPath("conf.yml")
```