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

	// misoconfig-prop: host | `localhost`
	PropMySQLHost = "mysql.host"

	// misoconfig-prop: port | 3306
	PropMySQLPort = "mysql.port"

	// misoconfig-prop: connection parameters (slices of strings) | "charset=utf8mb4"<br>"parseTime=True"<br>"loc=Local"<br>"readTimeout=30s"<br>"writeTimeout=30s"<br>"timeout=3s"
	PropMySQLConnParam = "mysql.connection.parameters"

	// misoconfig-prop: connection lifetime in minutes | 30
	PropMySQLConnLifetime = "mysql.connection.lifetime"

	// misoconfig-prop: max number of open connections | 10
	PropMySQLMaxOpenConns = "mysql.connection.open.max"

	// misoconfig-prop: max number of idle connections | 10
	PropMySQLMaxIdleConns = "mysql.connection.idle.max"
)

func init() {
	miso.SetDefProp(PropMySQLEnabled, false)
	miso.SetDefProp(PropMySQLUser, "root")
	miso.SetDefProp(PropMySQLPassword, "")
	miso.SetDefProp(PropMySQLHost, "localhost")
	miso.SetDefProp(PropMySQLPort, 3306)
	miso.SetDefProp(PropMySQLConnParam, []string{
		"charset=utf8mb4",
		"parseTime=True",
		"loc=Local",
		"readTimeout=30s",
		"writeTimeout=30s",
		"timeout=3s",
	})
	miso.SetDefProp(PropMySQLMaxOpenConns, 10)
	miso.SetDefProp(PropMySQLMaxIdleConns, 10)

	// Connection max lifetime, hikari recommends 1800000, so we do the same thing (30 minutes)
	miso.SetDefProp(PropMySQLConnLifetime, 30)
}
