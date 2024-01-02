//go:build !excl_mysql
// +build !excl_mysql

package miso

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	// Global handle to the database
	mysqlp = &mysqlHolder{conn: nil}

	// default connection parameters string
	defaultConnParams = []string{
		"charset=utf8mb4",
		"parseTime=True",
		"loc=Local",
		"readTimeout=30s",
		"writeTimeout=30s",
		"timeout=3s",
	}
)

type mysqlHolder struct {
	conn *gorm.DB
	sync.RWMutex
}

func init() {
	SetDefProp(PropMySQLEnabled, false)
	SetDefProp(PropMySQLUser, "root")
	SetDefProp(PropMySQLPassword, "")
	SetDefProp(PropMySQLHost, "localhost")
	SetDefProp(PropMySQLPort, 3306)
	SetDefProp(PropMySQLConnParam, defaultConnParams)
	SetDefProp(PropMySQLMaxOpenConns, 10)
	SetDefProp(PropMySQLMaxIdleConns, 10)

	// Connection max lifetime, hikari recommends 1800000, so we do the same thing
	SetDefProp(PropMySQLConnLifetime, 30)

	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap MySQL",
		Bootstrap: MySQLBootstrap,
		Condition: MySQLBootstrapCondition,
		Order:     BootstrapOrderL1,
	})
}

/*
Check if mysql is enabled

This func looks for following prop:

	"mysql.enabled"
*/
func IsMySqlEnabled() bool {
	return GetPropBool(PropMySQLEnabled)
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
func InitMySQLFromProp(rail Rail) error {
	p := MySQLConnParam{
		User:            GetPropStr(PropMySQLUser),
		Password:        GetPropStr(PropMySQLPassword),
		Schema:          GetPropStr(PropMySQLSchema),
		Host:            GetPropStr(PropMySQLHost),
		Port:            GetPropInt(PropMySQLPort),
		ConnParam:       strings.Join(GetPropStrSlice(PropMySQLConnParam), "&"),
		MaxOpenConns:    GetPropInt(PropMySQLMaxOpenConns),
		MaxIdleConns:    GetPropInt(PropMySQLMaxIdleConns),
		MaxConnLifetime: GetPropDur(PropMySQLConnLifetime, time.Minute),
	}
	return InitMySQL(rail, p)
}

type MySQLConnParam struct {
	User            string
	Password        string
	Schema          string
	Host            string
	Port            int
	ConnParam       string
	MaxConnLifetime time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
}

// Create new MySQL connection
func NewMySQLConn(rail Rail, p MySQLConnParam) (*gorm.DB, error) {
	p.ConnParam = strings.TrimSpace(p.ConnParam)
	if p.ConnParam != "" && !strings.HasPrefix(p.ConnParam, "?") {
		p.ConnParam = "?" + p.ConnParam
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s", p.User, p.Password, p.Host, p.Port, p.Schema, p.ConnParam)
	rail.Infof("Connecting to database '%s:%d/%s' with params: '%s'", p.Host, p.Port, p.Schema, p.ConnParam)

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

	if p.MaxConnLifetime > 0 {
		sqlDb.SetConnMaxLifetime(p.MaxConnLifetime)
	}
	if p.MaxOpenConns > 0 {
		sqlDb.SetMaxOpenConns(p.MaxOpenConns)
	}
	if p.MaxIdleConns > 0 {
		sqlDb.SetMaxIdleConns(p.MaxIdleConns)
	}

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
func InitMySQL(rail Rail, p MySQLConnParam) error {
	mysqlp.Lock()
	if mysqlp.conn != nil {
		mysqlp.Unlock()
		return nil
	}
	defer mysqlp.Unlock()

	if mysqlp.conn != nil {
		return nil
	}

	conn, err := NewMySQLConn(rail, p)
	if err != nil {
		return fmt.Errorf("failed to create mysql connection, %v:%v/%v, %w", p.User, p.Password, p.Schema, err)
	}
	mysqlp.conn = conn
	return nil
}

// Get MySQL Connection.
func GetMySQL() *gorm.DB {
	mysqlp.RLock()
	defer mysqlp.RUnlock()

	if mysqlp.conn == nil {
		panic("MySQL Connection hasn't been initialized yet")
	}

	if IsDebugLevel() {
		return mysqlp.conn.Debug()
	}

	return mysqlp.conn
}

// Check whether mysql client is initialized
func IsMySQLInitialized() bool {
	mysqlp.RLock()
	defer mysqlp.RUnlock()
	return mysqlp.conn != nil
}

func MySQLBootstrap(rail Rail) error {
	if e := InitMySQLFromProp(rail); e != nil {
		return fmt.Errorf("failed to establish connection to MySQL, %w", e)
	}

	AddHealthIndicator(HealthIndicator{
		Name: "MySQL Component",
		CheckHealth: func(rail Rail) bool {
			db, err := GetMySQL().DB()
			if err != nil {
				rail.Errorf("Failed to get MySQL DB, %v", err)
				return false
			}
			err = db.Ping()
			if err != nil {
				rail.Errorf("Failed to ping MySQL, %v", err)
				return false
			}
			return true
		},
	})

	return nil
}

func MySQLBootstrapCondition(rail Rail) (bool, error) {
	return IsMySqlEnabled(), nil
}
