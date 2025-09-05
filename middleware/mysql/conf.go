package mysql

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: MySQL Configuration
const (

	// misoconfig-prop: enable MySQL client | false
	PropMySQLEnabled = "mysql.enabled"

	// misoconfig-prop: username | root
	PropMySQLUser = "mysql.user"

	// misoconfig-prop: password
	PropMySQLPassword = "mysql.password"

	// misoconfig-prop: database
	PropMySQLSchema = "mysql.database"

	// misoconfig-prop: host | localhost
	PropMySQLHost = "mysql.host"

	// misoconfig-prop: port | 3306
	PropMySQLPort = "mysql.port"

	// misoconfig-prop: log sql statements | false
	PropMySQLLogSQL = "mysql.log-sql"

	// misoconfig-prop: enable prepared statement | true
	PropMySQLPrepareStmt = "mysql.prepare-statement"

	// misoconfig-prop: connection parameters (slices of strings) (see [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql?tab=readme-ov-file#dsn-data-source-name)) | `[]string{"charset=utf8mb4", "parseTime=true", "loc=Local", "readTimeout=30s", "writeTimeout=30s", "timeout=3s", "collation=utf8mb4_general_ci"}`
	PropMySQLConnParam = "mysql.connection.parameters"

	// misoconfig-prop: connection lifetime in minutes (hikari recommends 1800000, so we do the same thing) | 30
	PropMySQLConnLifetime = "mysql.connection.lifetime"

	// misoconfig-prop: max number of open connections | 10
	PropMySQLMaxOpenConns = "mysql.connection.open.max"

	// misoconfig-prop: max number of idle connections | 10
	PropMySQLMaxIdleConns = "mysql.connection.idle.max"

	// misoconfig-prop: managed connection username | root
	// misoconfig-doc-only
	PropMySQLManagedUser = "mysql.managed.${name}.user"

	// misoconfig-prop: managed connection password
	// misoconfig-doc-only
	PropMySQLManagedPassword = "mysql.managed.${name}.password"

	// misoconfig-prop: managed connection database
	// misoconfig-doc-only
	PropMySQLManagedSchema = "mysql.managed.${name}.database"

	// misoconfig-prop: managed connection host | localhost
	// misoconfig-doc-only
	PropMySQLManagedHost = "mysql.managed.${name}.host"

	// misoconfig-prop: managed connection port | 3306
	// misoconfig-doc-only
	PropMySQLManagedPort = "mysql.managed.${name}.port"

	// misoconfig-prop: managed connection enable prepared statement | true
	PropMySQLManagedPrepareStmt = "mysql.managed.${name}.prepare-statement"
)

// misoconfig-default-start
func init() {
	miso.SetDefProp(PropMySQLEnabled, false)
	miso.SetDefProp(PropMySQLUser, "root")
	miso.SetDefProp(PropMySQLHost, "localhost")
	miso.SetDefProp(PropMySQLPort, 3306)
	miso.SetDefProp(PropMySQLLogSQL, false)
	miso.SetDefProp(PropMySQLPrepareStmt, true)
	miso.SetDefProp(PropMySQLConnParam, []string{"charset=utf8mb4", "parseTime=true", "loc=Local", "readTimeout=30s", "writeTimeout=30s", "timeout=3s", "collation=utf8mb4_general_ci"})
	miso.SetDefProp(PropMySQLConnLifetime, 30)
	miso.SetDefProp(PropMySQLMaxOpenConns, 10)
	miso.SetDefProp(PropMySQLMaxIdleConns, 10)
	miso.SetDefProp(PropMySQLManagedPrepareStmt, true)
}

// misoconfig-default-end
