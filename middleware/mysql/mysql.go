package mysql

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/miso"
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

	minimumConnParam = "parseTime=True&loc=Local"

	mysqlBootstrapCallbacks = []MySQLBootstrapCallback{}
)

type mysqlHolder struct {
	conn *gorm.DB
	sync.RWMutex
}

func init() {
	miso.SetDefProp(PropMySQLEnabled, false)
	miso.SetDefProp(PropMySQLUser, "root")
	miso.SetDefProp(PropMySQLPassword, "")
	miso.SetDefProp(PropMySQLHost, "localhost")
	miso.SetDefProp(PropMySQLPort, 3306)
	miso.SetDefProp(PropMySQLConnParam, defaultConnParams)
	miso.SetDefProp(PropMySQLMaxOpenConns, 10)
	miso.SetDefProp(PropMySQLMaxIdleConns, 10)

	// Connection max lifetime, hikari recommends 1800000, so we do the same thing
	miso.SetDefProp(PropMySQLConnLifetime, 30)

	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Bootstrap MySQL",
		Bootstrap: MySQLBootstrap,
		Condition: MySQLBootstrapCondition,
		Order:     miso.BootstrapOrderL1,
	})
}

/*
Check if mysql is enabled

This func looks for following prop:

	"mysql.enabled"
*/
func IsMySqlEnabled() bool {
	return miso.GetPropBool(PropMySQLEnabled)
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
func InitMySQLFromProp(rail miso.Rail) error {
	p := MySQLConnParam{
		User:            miso.GetPropStr(PropMySQLUser),
		Password:        miso.GetPropStr(PropMySQLPassword),
		Schema:          miso.GetPropStr(PropMySQLSchema),
		Host:            miso.GetPropStr(PropMySQLHost),
		Port:            miso.GetPropInt(PropMySQLPort),
		ConnParam:       strings.Join(miso.GetPropStrSlice(PropMySQLConnParam), "&"),
		MaxOpenConns:    miso.GetPropInt(PropMySQLMaxOpenConns),
		MaxIdleConns:    miso.GetPropInt(PropMySQLMaxIdleConns),
		MaxConnLifetime: miso.GetPropDur(PropMySQLConnLifetime, time.Minute),
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
func NewMySQLConn(rail miso.Rail, p MySQLConnParam) (*gorm.DB, error) {
	p.ConnParam = strings.TrimSpace(p.ConnParam)
	if p.ConnParam != "" && !strings.HasPrefix(p.ConnParam, "?") {
		p.ConnParam = "?" + p.ConnParam
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s", p.User, p.Password, p.Host, p.Port, p.Schema, p.ConnParam)
	rail.Infof("Connecting to database '%s:%d/%s' with params: '%s'", p.Host, p.Port, p.Schema, p.ConnParam)

	conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{PrepareStmt: true})
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
func InitMySQL(rail miso.Rail, p MySQLConnParam) error {
	if p.ConnParam == "" {
		p.ConnParam = minimumConnParam
	}
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

	if miso.IsDebugLevel() {
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

func MySQLBootstrap(app *miso.MisoApp, rail miso.Rail) error {
	if e := InitMySQLFromProp(rail); e != nil {
		return fmt.Errorf("failed to establish connection to MySQL, %w", e)
	}

	if len(mysqlBootstrapCallbacks) > 0 {
		db := GetMySQL()
		for _, cbk := range mysqlBootstrapCallbacks {
			if err := cbk(rail, db); err != nil {
				return fmt.Errorf("failed to execute MySQLBootstrapCallback, %w", err)
			}
		}
	}

	miso.AddHealthIndicator(miso.HealthIndicator{
		Name: "MySQL Component",
		CheckHealth: func(rail miso.Rail) bool {
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

func MySQLBootstrapCondition(app *miso.MisoApp, rail miso.Rail) (bool, error) {
	return IsMySqlEnabled(), nil
}

type MySQLBootstrapCallback func(rail miso.Rail, db *gorm.DB) error

func AddMySQLBootstrapCallback(cbk MySQLBootstrapCallback) {
	mysqlBootstrapCallbacks = append(mysqlBootstrapCallbacks, cbk)
}
