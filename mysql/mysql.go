package mysql

import (
	"fmt"
	"sync"
	"time"

	. "github.com/curtisnewbie/gocommon/common"
	"github.com/sirupsen/logrus"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)
const (
	// Connection max lifetime, hikari recommends 1800000, so we do the same thing
	CONN_MAX_LIFE_TIME = time.Minute * 30

	// Max num of open conns
	MAX_OPEN_CONNS = 10

	// Max num of idle conns
	MAX_IDLE_CONNS = MAX_OPEN_CONNS // recommended to be the same as the maxOpenConns
)

var (
	// Global handle to the database
	mysqlp = &mysqlHolder{mysql: nil}
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
	Init connection to mysql, if failed, panic

	If mysql client has been initialized, current func call will be ignored.

	This func looks for following props:

		"mysql.user"
		"mysql.password"
		"mysql.database"
		"mysql.host"
		"mysql.port"
	
	This func is essentially the same as: 
		InitMySqlFromProp
*/
func MustInitMySqlFromProp() {
	e := InitMySqlFromProp()
	if e != nil {
		panic(e)
	}
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
*/
func InitMySqlFromProp() error {
	return InitMySql(GetPropStr(PROP_MYSQL_USER),
		GetPropStr(PROP_MYSQL_PASSWORD),
		GetPropStr(PROP_MYSQL_DATABASE),
		GetPropStr(PROP_MYSQL_HOST),
		GetPropStr(PROP_MYSQL_PORT))
}

/*
	Init Handle to the database

	If mysql client has been initialized, current func call will be ignored.
*/
func InitMySql(user string, password string, dbname string, host string, port string) error {
	if IsMySqlInitialized() {
		return nil
	}

	mysqlp.mu.Lock()
	defer mysqlp.mu.Unlock()

	if mysqlp.mysql != nil {
		return nil
	}

	params := "charset=utf8mb4&parseTime=True&loc=Local&readTimeout=30s&writeTimeout=30s&timeout=3s"
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?%v", user, password, host, port, dbname, params)
	logrus.Infof("Connecting to database '%v:%v' with params: '%v'", host, port, params)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Infof("Failed to connect to MySQL, err: %v", err)
		return err
	}

	sqlDb, err := db.DB()
	if err != nil {
		logrus.Infof("Failed to obtain MySQL conn from gorm, %v", err)
		return err
	}

	sqlDb.SetConnMaxLifetime(CONN_MAX_LIFE_TIME)
	sqlDb.SetMaxOpenConns(MAX_OPEN_CONNS)
	sqlDb.SetMaxIdleConns(MAX_IDLE_CONNS)

	err = sqlDb.Ping() // make sure the handle is actually connected
	if err != nil {
		logrus.Infof("Ping DB Error, %v, connection may not be established", err)
		return err
	}

	logrus.Infof("MySQL conn initialized")
	mysqlp.mysql = db

	return nil
}

/*
	Get mysql client

	Must call InitMysql method before this method.
*/
func GetMySql() *gorm.DB {
	mysqlp.mu.RLock()
	defer mysqlp.mu.RUnlock()

	if mysqlp.mysql == nil {
		panic("MySQL Connection hasn't been initialized yet")
	}

	if IsProdMode() {
		return mysqlp.mysql
	}

	// not prod mode, enable debugging for printing SQLs
	return mysqlp.mysql.Debug()
}

// Check whether mysql client is initialized
func IsMySqlInitialized() bool {
	mysqlp.mu.RLock()
	defer mysqlp.mu.RUnlock()
	return mysqlp.mysql != nil
}
