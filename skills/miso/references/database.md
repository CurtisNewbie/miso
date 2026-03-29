# Database

Database operations with GORM integration in miso framework.

miso framework provides two database middleware packages:

- **MySQL**: `github.com/curtisnewbie/miso/middleware/mysql`
- **SQLite**: `github.com/curtisnewbie/miso/middleware/sqlite`

Both packages return `*gorm.DB` instances with the same interface, so you can use either one seamlessly.

## Getting Database Instance

```go
import (
    "github.com/curtisnewbie/miso/middleware/mysql"
    "github.com/curtisnewbie/miso/middleware/sqlite"
)

// Get MySQL connection
db := mysql.GetMySQL()

// Get SQLite connection
db := sqlite.GetDB()
```

## Model Definition

```go
type User struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    Name      string    `json:"name" gorm:"not null;size:100"`
    Email     string    `json:"email" gorm:"uniqueIndex;not null;size:255"`
    Age       int       `json:"age" gorm:"default:0"`
    Status    string    `json:"status" gorm:"default:active;size:20"`
    CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
    UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

// Specify table name
func (User) TableName() string {
    return "users"
}
```

**Including Audit Fields:**

For automatic audit tracking, add these fields to your models:

```go
type User struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    Name      string    `json:"name" gorm:"not null;size:100"`
    Email     string    `json:"email" gorm:"uniqueIndex;not null;size:255"`

    CreatedBy string    `json:"createdBy" gorm:"size:100"`
    UpdatedBy string    `json:"updatedBy" gorm:"size:100"`
    TraceId   string    `json:"traceId" gorm:"size:64"`

    CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
    UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}
```

## Automatic Audit Fields

`dbquery` provides hooks to automatically set audit fields (`created_by`, `updated_by`, `trace_id`) on INSERT and UPDATE operations.

**Setup (call before miso bootstraps):**

```go
import "github.com/curtisnewbie/miso/middleware/dbquery"

func init() {
    // Enable automatic audit fields for all tables
    dbquery.PrepareCreateModelHook()
    dbquery.PrepareUpdateModelHook()
}
```

**Optional: Enable for specific tables only:**

```go
func init() {
    // Only enable for tables that have these columns
    dbquery.PrepareCreateModelHook(func(table string) bool {
        return table == "users" || table == "orders"
    })
    dbquery.PrepareUpdateModelHook(func(table string) bool {
        return table == "users" || table == "orders"
    })
}
```

**How it works:**

- `created_by`: Automatically set from `rail.Username()` on INSERT (if empty)
- `updated_by`: Automatically set from `rail.Username()` on UPDATE (if not already set)
- `trace_id`: Automatically set from `rail.TraceId()` on all INSERT/UPDATE operations

```go
// INSERT: created_by and trace_id are automatically set
err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Insert(user)

// UPDATE: updated_by and trace_id are automatically set
err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Eq("id", userID).
    Set("name", "New Name").
    Update()
```

## Bootstrap Database

The database middleware packages automatically bootstrap when configured. No manual initialization is required.

### MySQL Configuration

```yaml
mysql:
  enabled: true
  user: root
  password: secret
  database: mydb
  host: localhost
  port: 3306
```

For complete configuration options, see [config.md](https://github.com/CurtisNewbie/miso/blob/main/doc/config.md).

### SQLite Configuration

```yaml
sqlite:
  file: /path/to/database.db
```

For complete configuration options, see [config.md](https://github.com/CurtisNewbie/miso/blob/main/doc/config.md).

### Custom Bootstrap Callback

If you need to perform database operations during bootstrap:

```go
import (
    "github.com/curtisnewbie/miso/errs"
    "github.com/curtisnewbie/miso/middleware/mysql"
    "gorm.io/gorm"
)

func init() {
    mysql.AddMySQLBootstrapCallback(func(rail miso.Rail, db *gorm.DB) error {
        // Seed initial data
        if db.Where("name = ?", "admin").First(&User{}).Error == gorm.ErrRecordNotFound {
            admin := User{Name: "admin", Email: "admin@example.com"}
            if err := db.Create(&admin).Error; err != nil {
                return errs.WrapErr(err, "failed to seed admin user")
            }
        }

        return nil
    })
}
```

## Query Operations

### Find All

```go
import (
    "github.com/curtisnewbie/miso/middleware/mysql"
    "github.com/curtisnewbie/miso/middleware/dbquery"
)

var users []User
if err := dbquery.NewQuery(rail, mysql.GetMySQL()).Table("users").Scan(&users); err != nil {
    return errs.WrapErr(err, "failed to fetch users")
}
```

### Find by ID

```go
var user User
err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("id = ?", userID).
    Scan(&user)

if err != nil {
    if errs.IsNoneErr(err) {
        return errs.NewErrfCode("USER_NOT_FOUND", "User does not exist")
    }
    return errs.WrapErr(err, "failed to fetch user")
}
```

### Find by Conditions

```go
var users []User
dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("status = ?", "active").
    Scan(&users)

var user User
dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("email = ?", "user@example.com").
    Scan(&user)
```

### Complex Queries

```go
var users []User
dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("age >= ?", 18).
    Where("status IN ?", []string{"active", "pending"}).
    OrderDesc("created_at").
    Limit(10).
    Offset(0).
    Scan(&users)
```

### Query with OR

```go
dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("name = ?", "John").
    Or("email = ?", "john@example.com").
    Scan(&user)
```

### Query with Not

```go
dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Not("status = ?", "deleted").
    Scan(&users)
```

## Create Operations

### Create Single Record

```go
import (
    "github.com/curtisnewbie/miso/middleware/mysql"
    "github.com/curtisnewbie/miso/middleware/dbquery"
    "github.com/curtisnewbie/miso/errs"
)

user := User{
    ID:    uuid.New().String(),
    Name:  "John Doe",
    Email: "john@example.com",
    Age:   30,
}

if _, err := dbquery.NewQuery(rail, mysql.GetMySQL()).Table("users").Create(&user); err != nil {
    return errs.WrapErr(err, "failed to create user")
}
```

### Create Multiple Records

```go
users := []User{
    {ID: uuid.New().String(), Name: "John", Email: "john@example.com"},
    {ID: uuid.New().String(), Name: "Jane", Email: "jane@example.com"},
}

if _, err := dbquery.NewQuery(rail, mysql.GetMySQL()).Table("users").Create(&users); err != nil {
    return errs.WrapErr(err, "failed to create users")
}
```

## Update Operations

### Update Single Field

```go
import (
    "github.com/curtisnewbie/miso/middleware/mysql"
    "github.com/curtisnewbie/miso/middleware/dbquery"
    "github.com/curtisnewbie/miso/errs"
)

if _, err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("id = ?", userID).
    Update("email", "new@example.com"); err != nil {
    return errs.WrapErr(err, "failed to update user email")
}
```

### Update Multiple Fields

```go
if _, err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("id = ?", userID).
    Update(map[string]interface{}{
        "name":  "John Updated",
        "email": "john.updated@example.com",
    }); err != nil {
    return errs.WrapErr(err, "failed to update user")
}
```

### Update with Conditions

```go
dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("status = ?", "active").
    Update("last_login", time.Now())
```

### Updates from Struct

```go
type UpdateUserReq struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

updates := UpdateUserReq{Name: "New Name", Email: "new@example.com"}
dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("id = ?", userID).
    Update(updates)
```

## Delete Operations

### Soft Delete

Soft delete is **strongly recommended** over hard delete in most scenarios (99% of cases). Your application should define how soft delete is achieved, typically by adding fields like `deleted` (bool) or `deleted_at` (timestamp) to your tables.

```go
import (
    "time"
    "github.com/curtisnewbie/miso/middleware/mysql"
    "github.com/curtisnewbie/miso/middleware/dbquery"
)

// Define your model with soft delete field
type User struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    Name      string    `json:"name"`
    DeletedAt *time.Time `json:"deletedAt"` // soft delete marker (application-defined)
    Deleted   bool      `json:"deleted"`    // alternative soft delete marker
}

// Soft delete by setting both deleted_at and deleted
if _, err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("id = ?", userID).
    Set("deleted_at", time.Now()).
    Set("deleted", true).
    Update(); err != nil {
    return errs.WrapErr(err, "failed to soft delete user")
}

// When querying, exclude soft-deleted records
var users []User
if err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    IsNull("deleted_at").
    Ne("deleted", true).
    Scan(&users); err != nil {
    return errs.WrapErr(err, "failed to fetch users")
}
```

### Hard Delete

Hard delete permanently removes records from the database. Use with caution as it cannot be undone. Only use hard delete for temporary data, cache entries, or when explicitly required by compliance/policy.

```go
// Hard delete (actually remove from database)
if _, err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("id = ?", userID).
    Delete(); err != nil {
    return errs.WrapErr(err, "failed to hard delete user")
}
```

## Transactions

```go
import (
    "github.com/curtisnewbie/miso/middleware/mysql"
    "github.com/curtisnewbie/miso/middleware/dbquery"
    "github.com/curtisnewbie/miso/errs"
)

err := dbquery.NewQuery(rail, mysql.GetMySQL()).Transaction(func(tx *dbquery.Query) error {
    // Transaction operations
    if _, err := tx.Table("users").Create(&user); err != nil {
        return err
    }

    if _, err := tx.Table("profiles").Create(&profile); err != nil {
        return err
    }

    return nil
})

if err != nil {
    return errs.WrapErr(err, "transaction failed")
}
```

### Manual Transaction

```go
import (
    "github.com/curtisnewbie/miso/middleware/mysql"
    "github.com/curtisnewbie/miso/middleware/dbquery"
    "github.com/curtisnewbie/miso/errs"
)

tx := dbquery.NewQuery(rail, mysql.GetMySQL()).Begin()
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
    }
}()

if _, err := tx.Table("users").Create(&user); err != nil {
    tx.Rollback()
    return errs.WrapErr(err, "failed to create user")
}

if _, err := tx.Table("profiles").Create(&profile); err != nil {
    tx.Rollback()
    return errs.WrapErr(err, "failed to create profile")
}

if err := tx.Commit(); err != nil {
    return errs.WrapErr(err, "failed to commit transaction")
}
```

## Raw SQL

```go
import (
    "github.com/curtisnewbie/miso/middleware/mysql"
    "github.com/curtisnewbie/miso/middleware/dbquery"
)

var users []User
dbquery.NewQuery(rail, mysql.GetMySQL()).
    Raw("SELECT * FROM users WHERE age > ?", 18).
    Scan(&users)

var count int64
dbquery.NewQuery(rail, mysql.GetMySQL()).
    Raw("SELECT COUNT(*) FROM users").
    Scan(&count)
```

### Execute Raw SQL

```go
dbquery.NewQuery(rail, mysql.GetMySQL()).
    Raw("UPDATE users SET status = ? WHERE id = ?", "active", userID)
```

## Error Handling

```go
import (
    "github.com/curtisnewbie/miso/middleware/mysql"
    "github.com/curtisnewbie/miso/middleware/dbquery"
)

var user User
err := dbquery.NewQuery(rail, mysql.GetMySQL()).
    Table("users").
    Where("id = ?", userID).
    Scan(&user)

if err != nil {
    if errs.IsNoneErr(err) {
        return errs.NewErrfCode("USER_NOT_FOUND", "User does not exist")
    }
    return errs.WrapErr(err, "database query failed")
}
```

## Pagination

```go
import (
    "github.com/curtisnewbie/miso/middleware/mysql"
    "github.com/curtisnewbie/miso/middleware/dbquery"
)

type ListUsersReq struct {
    miso.Paging `json:",inline"`
    Name        string `json:"name"`
    Status      string `json:"status"`
}

type ListUsersRes struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func ListUsers(rail miso.Rail, req ListUsersReq) (miso.PageRes[ListUsersRes], error) {
    return dbquery.NewPagedQuery[ListUsersRes](mysql.GetMySQL()).
        WithBaseQuery(func(q *dbquery.Query) *dbquery.Query {
            return q.Table("users").
                LikeIf(req.Name != "", "name", req.Name).
                EqIf(req.Status != "", "status", req.Status)
        }).
        WithSelectQuery(func(q *dbquery.Query) *dbquery.Query {
            return q.Select("id,name,email").OrderDesc("created_at")
        }).
        Scan(rail, req.Paging)
}
```

### IterateAllByOffset

For large datasets, use `IterateAllByOffset1` or `IterateAllByOffset2` to iterate through records efficiently.

**Why not `LIMIT/OFFSET`?**

`LIMIT/OFFSET` becomes inefficient and unreliable with large datasets:

**Performance issue:**
- MySQL must scan and skip `OFFSET` rows, causing performance degradation as pagination depth increases
- Example: `LIMIT 100 OFFSET 1000000` scans 1,010,000 rows but returns only 100

**Data consistency issue:**
- If the dataset is modified during iteration (inserts/deletes), `LIMIT/OFFSET` causes duplicates or missed records
- Example: Deleting rows before current offset causes later queries to return different rows
- Example: Inserting rows at the beginning causes some records to be skipped
- **Critical:** When modifying data during iteration, records may not update properly and `LIMIT/OFFSET` can query the same records repeatedly
  - Scenario: Query `WHERE status = 'pending'` with `LIMIT 100 OFFSET 0`, process first 100 records and update to `status = 'processed'`
  - Next query `LIMIT 100 OFFSET 100`: If some records didn't update or new pending records were inserted, this query may return previously processed records
  - Result: Same records get processed multiple times, causing data corruption

**How `IterateAllByOffset` works:**

Uses indexed `WHERE column > last_value` instead:
- Leverages index on offset columns, performance remains constant regardless of pagination depth
- Tracks the last seen value from each page and uses it as filter for the next
- Safe from insertions/deletions as long as offset column values remain stable
- **Best practice:** Use immutable columns as offset columns (e.g., auto-increment ID)

**Single Offset Column:**

```go
import (
    "github.com/curtisnewbie/miso/middleware/mysql"
    "github.com/curtisnewbie/miso/middleware/dbquery"
)

type Record struct {
    RecTime time.Time `json:"rec_time"`
    RecId   string    `json:"rec_id"`
    Data    string    `json:"data"`
}

func ListRecords(rail miso.Rail, forEachPage func(v []Record) error) error {
    return dbquery.IterateAllByOffset1(rail, mysql.GetMySQL(), dbquery.IterateByOffset1Param[Record, time.Time]{
        OffsetCol: "rec_time",
        Limit:     100,
        BuildQuery: func(rail miso.Rail, q *dbquery.Query) *dbquery.Query {
            return q.Table("my_table").
                Eq("status", "active").
                SelectCols(Record{})
        },
        ForEachPage: func(p []Record) (stop bool, err error) {
            return false, forEachPage(p)
        },
        GetOffset: func(v Record) time.Time {
            return v.RecTime
        },
    })
}
```

**Two Offset Columns:**

```go
func ListRecords(rail miso.Rail, forEachPage func(v []Record) error) error {
    return dbquery.IterateAllByOffset2(rail, mysql.GetMySQL(), dbquery.IterateByOffset2Param[Record, time.Time, string]{
        OffsetCol1: "rec_time",
        OffsetCol2: "rec_id",
        Limit:      100,
        BuildQuery: func(rail miso.Rail, q *dbquery.Query) *dbquery.Query {
            return q.Table("my_table").
                Eq("status", "active").
                SelectCols(Record{})
        },
        ForEachPage: func(p []Record) (stop bool, err error) {
            return false, forEachPage(p)
        },
        GetOffset: func(v Record) (time.Time, string) {
            return v.RecTime, v.RecId
        },
    })
}
```

## Database Configuration Properties

### MySQL Configuration

```yaml
mysql:
  enabled: true
  user: root
  password: secret
  database: mydb
  host: localhost
  port: 3306
```

For complete configuration options, see [config.md](https://github.com/CurtisNewbie/miso/blob/main/doc/config.md).

### SQLite Configuration

```yaml
sqlite:
  file: /path/to/database.db
```

For complete configuration options, see [config.md](https://github.com/CurtisNewbie/miso/blob/main/doc/config.md).

### Managed MySQL Connections

You can configure multiple MySQL connections:

```yaml
mysql:
  enabled: true
  user: root
  password: secret
  database: main_db
  host: localhost
  port: 3306
  managed:
    read-replica:
      host: replica.example.com
      user: readonly
      password: secret
      database: main_db
      port: 3306
    analytics:
      host: analytics.example.com
      user: analyst
      password: secret
      database: analytics_db
      port: 3306
```

```go
import "github.com/curtisnewbie/miso/middleware/mysql"

// Get primary connection
db := mysql.GetMySQL()

// Get managed connection
replicaDB := mysql.GetManaged("read-replica")
analyticsDB := mysql.GetManaged("analytics")
```