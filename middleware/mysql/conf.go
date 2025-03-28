package mysql

import "github.com/curtisnewbie/miso/miso"

const (
	/*
		------------------------------------

		Prop for MySQL

		------------------------------------
	*/
	PropMySQLEnabled      = "mysql.enabled"
	PropMySQLUser         = "mysql.user"
	PropMySQLPassword     = "mysql.password"
	PropMySQLSchema       = "mysql.database"
	PropMySQLHost         = "mysql.host"
	PropMySQLPort         = "mysql.port"
	PropMySQLConnParam    = "mysql.connection.parameters"
	PropMySQLConnLifetime = "mysql.connection.lifetime"
	PropMySQLMaxOpenConns = "mysql.connection.open.max"
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
