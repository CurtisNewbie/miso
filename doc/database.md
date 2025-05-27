# Database

miso currently supports MySQL and SQLite. Integration is internally backed by `gorm`, so essentially, miso focuses on managing these gorm instances for you, and maybe provide you with some tools to query database more easily.

To use MySQL or SQLite, you need to follow the configuration guide [config.md](./config.md), and enable the related middleware.

E.g.,

```yaml
mysql:
  enabled: true
  host: "localhost"
  port: "3306"
  user: "root"
  password: ""
  database: "mydb"
  connection:
    parameters:
      - "charset=utf8mb4"
      - "parseTime=True"
      - "loc=Local"
      - "readTimeout=30s"
      - "writeTimeout=30s"
      - "timeout=3s"
```

In your application code, you need to import the related go middleware package provided by miso.

E.g.,

```go
import (
    _ "github.com/curtisnewbie/miso/middleware/mysql" // for MySQL
    _ "github.com/curtisnewbie/miso/middleware/sqlite" // for SQLite
)
```

If you can, you should try your best to only use `dbquery` package not the implementation package, as it encapsulates which database your are connecting to. After all, it's still a `*gorm.DB` instance.

```go
import (
	"github.com/curtisnewbie/miso/middleware/dbquery"
)

func getMyDB() {
	var db *gorm.DB = dbquery.GetDB()
}
```

If you are not familiar with gorm, you are free to learn it from it's official website. It's an ORM framework.

miso provides you with some extra utility methods to query the database more efficiently. Feel free to have a look at `miso/middleware/dbquery` package.

For example,

To scan a row, we can use `dbquery.NewQuery()`:

```go
var t DroneTask
n, err := dbquery.NewQuery(db).
    Table("task").
    Eq("task_id", taskId).
    Select("task_id,user_no,status,dir_file_key,temp_dir,platform,url,attempt,file_count,temp_dir_cleaned").
    Scan(&t)
if err != nil {
    return t, err
}
if n < 1 {
    return t, ErrTaskNotFound
}
```

To update a row, we can use `dbquery.NewQuery()`:

```go
_, err := dbquery.NewQuery(db).
    Table("task").
    Set("status", TaskStatusCancelled).
    Set("updated_by", user.UserNo).
    Eq("task_id", req.TaskId).
    Update()
if err != nil {
    panic(err) // demo only
}
```

To query a page of data, we can use `dbquery.NewPagedQuery()`:

```go
func ListSitePasswords(rail miso.Rail, req ListSitePasswordReq, user common.User, db *gorm.DB) (miso.PageRes[ListSitePasswordRes], error) {
	return dbquery.NewPagedQuery[ListSitePasswordRes](db).
		WithBaseQuery(func(q *dbquery.Query) *dbquery.Query {
			return q.Table("site_password").
				Eq("user_no", user.UserNo).
				LikeIf(req.Alias != "", "alias", req.Alias).
				LikeIf(req.Site != "", "site", req.Site).
				LikeIf(req.Username != "", "username", req.Username)
		}).
		WithSelectQuery(func(q *dbquery.Query) *dbquery.Query {
			return q.Select("record_id,site,alias,username,create_time")
		}).
		Scan(rail, req.Paging)
}
```

`dbquery` also supports method to iterate all rows that match the given conditions:

```go
err := dbquery.NewPagedQuery[ScrapingTask](db).
    WithBaseQuery(func(q *dbquery.Query) *dbquery.Query {
        return q.Table("task").
            Eq("status", TaskStatusPending)
    }).
    WithSelectQuery(func(q *dbquery.Query) *dbquery.Query {
        return q.Select("task_id").Order("id ASC")
    }).
    IterateAll(rail, dbquery.IteratePageParam{Limit: 10}, func(v ScrapingTask) (stop bool, err error) {
        // do something for each row
        doScrape(v)
        return false, nil
    })
```
