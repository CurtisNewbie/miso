package mysql

import "github.com/curtisnewbie/miso/miso"

// misoapi-config-section: MySQL Configuration
const (

	// misoapi-config: enable MySQL client | false
	PropMySQLEnabled = "mysql.enabled"

	// misoapi-config: username | root
	PropMySQLUser = "mysql.user"

	// misoapi-config: password
	PropMySQLPassword = "mysql.password"

	// misoapi-config: database
	PropMySQLSchema = "mysql.database"

	// misoapi-config: host | `localhost`
	PropMySQLHost = "mysql.host"

	// misoapi-config: port | 3306
	PropMySQLPort = "mysql.port"

	// misoapi-config: connection parameters (slices of strings) | "charset=utf8mb4"<br>"parseTime=True"<br>"loc=Local"<br>"readTimeout=30s"<br>"writeTimeout=30s"<br>"timeout=3s"
	PropMySQLConnParam = "mysql.connection.parameters"

	// misoapi-config: connection lifetime in minutes | 30
	PropMySQLConnLifetime = "mysql.connection.lifetime"

	// misoapi-config: max number of open connections | 10
	PropMySQLMaxOpenConns = "mysql.connection.open.max"

	// misoapi-config: max number of idle connections | 10
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
