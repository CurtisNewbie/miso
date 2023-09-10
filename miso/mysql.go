package miso

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	CONN_MAX_LIFE_TIME = time.Minute * 30 // Connection max lifetime, hikari recommends 1800000, so we do the same thing
	MAX_OPEN_CONNS     = 10               // Max num of open conns
	MAX_IDLE_CONNS     = MAX_OPEN_CONNS   // max num of idle conns, recommended to be the same as the maxOpenConns
)

var (
	// Global handle to the database
	mysqlp = &mysqlHolder{mysql: nil}

	// default connection parameters string
	defaultConnParams = strings.Join([]string{
		"charset=utf8mb4", "parseTime=True", "loc=Local", "readTimeout=30s", "writeTimeout=30s", "timeout=3s",
	}, "&")
)

type mysqlHolder struct {
	mysql *gorm.DB
	mu    sync.RWMutex
}

func init() {
	SetDefProp(PROP_MYSQL_ENABLED, false)
	SetDefProp(PROP_MYSQL_USER, "root")
	SetDefProp(PROP_MYSQL_PASSWORD, "")
	SetDefProp(PROP_MYSQL_HOST, "localhost")
	SetDefProp(PROP_MYSQL_PORT, 3306)
	SetDefProp(PROP_MYSQL_CONN_PARAM, defaultConnParams)
}

/*
Check if mysql is enabled

This func looks for following prop:

	"mysql.enabled"
*/
func IsMySqlEnabled() bool {
	return GetPropBool(PROP_MYSQL_ENABLED)
}

/*
Init connection to mysql

If mysql client has been initialized, current func call will be ignored.

This func looks for following props:

	"mysql.user"
	"mysql.password"
	"mysql.database"
	"mysql.host"
	"mysql.port"
	"mysql.connection.parameters"
*/
func InitMySQLFromProp() error {
	return InitMySQL(GetPropStr(PROP_MYSQL_USER),
		GetPropStr(PROP_MYSQL_PASSWORD),
		GetPropStr(PROP_MYSQL_DATABASE),
		GetPropStr(PROP_MYSQL_HOST),
		GetPropStr(PROP_MYSQL_PORT),
		GetPropStr(PROP_MYSQL_CONN_PARAM))
}

// Create new MySQL connection
func NewMySQLConn(user string, password string, dbname string, host string, port string, connParam string) (*gorm.DB, error) {
	rail := EmptyRail()
	connParam = strings.TrimSpace(connParam)
	if connParam != "" && !strings.HasPrefix(connParam, "?") {
		connParam = "?" + connParam
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s%s", user, password, host, port, dbname, connParam)
	rail.Infof("Connecting to database '%s:%s/%s' with params: '%s'", host, port, dbname, connParam)

	conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		rail.Infof("Failed to connect to MySQL, err: %v", err)
		return nil, err
	}

	sqlDb, err := conn.DB()
	if err != nil {
		rail.Infof("Failed to obtain MySQL conn from gorm, %v", err)
		return nil, err
	}

	sqlDb.SetConnMaxLifetime(CONN_MAX_LIFE_TIME)
	sqlDb.SetMaxOpenConns(MAX_OPEN_CONNS)
	sqlDb.SetMaxIdleConns(MAX_IDLE_CONNS)

	err = sqlDb.Ping() // make sure the handle is actually connected
	if err != nil {
		rail.Infof("Ping DB Error, %v, connection may not be established", err)
		return nil, err
	}

	rail.Infof("MySQL connection established")
	return conn, nil
}

/*
Init Handle to the database

If mysql client has been initialized, current func call will be ignored.
*/
func InitMySQL(user string, password string, dbname string, host string, port string, connParam string) error {
	if IsMySQLInitialized() {
		return nil
	}

	mysqlp.mu.Lock()
	defer mysqlp.mu.Unlock()

	if mysqlp.mysql != nil {
		return nil
	}

	conn, enc := NewMySQLConn(user, password, dbname, host, port, connParam)
	if enc != nil {
		return TraceErrf(enc, "failed to create mysql connection, %v:%v/%v", user, password, dbname)
	}
	mysqlp.mysql = conn
	return nil
}

/*
Get MySQL Connection.

If client is not yet created, func InitMySqlFromProp(...) is called to initialize a new one. For any error occurred, it panics.
*/
func GetMySQL() *gorm.DB {
	mysqlp.mu.RLock()
	defer mysqlp.mu.RUnlock()

	if mysqlp.mysql == nil {
		if e := InitMySQLFromProp(); e != nil {
			panic(fmt.Sprintf("MySQL Connection hasn't been initialized, even failed to initialize one with func InitMySqlFromProp(), no choice but to panic, %v", e))
		}
	}

	if IsDebugLevel() {
		return mysqlp.mysql.Debug()
	}

	return mysqlp.mysql
}

// Check whether mysql client is initialized
func IsMySQLInitialized() bool {
	mysqlp.mu.RLock()
	defer mysqlp.mu.RUnlock()
	return mysqlp.mysql != nil
}
