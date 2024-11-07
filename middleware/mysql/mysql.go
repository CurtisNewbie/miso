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

const (
	minimumConnParam = "parseTime=True&loc=Local"
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

	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Bootstrap MySQL",
		Bootstrap: mysqlBootstrap,
		Condition: mysqlBootstrapCondition,
		Order:     miso.BootstrapOrderL1,
	})
}

var appModule, module = miso.InitAppModuleFunc(func(app *miso.MisoApp) *mysqlModule {
	return &mysqlModule{
		mu:   &sync.RWMutex{},
		conf: app.Config(),
	}
})

type MySQLBootstrapCallback func(rail miso.Rail, db *gorm.DB) error

type mysqlModule struct {
	mu                 *sync.RWMutex
	conn               *gorm.DB
	bootstrapCallbacks []MySQLBootstrapCallback
	conf               *miso.AppConfig
}

func (m *mysqlModule) init(rail miso.Rail, p MySQLConnParam) error {
	if p.ConnParam == "" {
		p.ConnParam = minimumConnParam
	}
	m.mu.RLock()
	if m.conn != nil {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn != nil {
		return nil
	}

	conn, err := NewMySQLConn(rail, p)
	if err != nil {
		return fmt.Errorf("failed to create mysql connection, %v:***/%v, %w", p.User, p.Schema, err)
	}
	m.conn = conn
	return nil
}

func (m *mysqlModule) mysql() *gorm.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.conn == nil {
		panic("MySQL Connection hasn't been initialized yet")
	}

	if miso.IsDebugLevel() {
		return m.conn.Debug()
	}

	return m.conn
}

func (m *mysqlModule) initialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conn != nil
}

func (m *mysqlModule) addMySQLBootstrapCallback(cbk MySQLBootstrapCallback) {
	m.bootstrapCallbacks = append(m.bootstrapCallbacks, cbk)
}

func (m *mysqlModule) initFromProp(rail miso.Rail) error {
	p := MySQLConnParam{
		User:            m.conf.GetPropStr(PropMySQLUser),
		Password:        m.conf.GetPropStr(PropMySQLPassword),
		Schema:          m.conf.GetPropStr(PropMySQLSchema),
		Host:            m.conf.GetPropStr(PropMySQLHost),
		Port:            m.conf.GetPropInt(PropMySQLPort),
		ConnParam:       strings.Join(m.conf.GetPropStrSlice(PropMySQLConnParam), "&"),
		MaxOpenConns:    m.conf.GetPropInt(PropMySQLMaxOpenConns),
		MaxIdleConns:    m.conf.GetPropInt(PropMySQLMaxIdleConns),
		MaxConnLifetime: m.conf.GetPropDur(PropMySQLConnLifetime, time.Minute),
	}
	return m.init(rail, p)
}

func (m *mysqlModule) runBootstrapCallbacks(rail miso.Rail) error {
	if len(m.bootstrapCallbacks) > 0 {
		db := GetMySQL()
		for _, cbk := range m.bootstrapCallbacks {
			if err := cbk(rail, db); err != nil {
				return fmt.Errorf("failed to execute MySQLBootstrapCallback, %w", err)
			}
		}
	}
	return nil
}

func (m *mysqlModule) registerHealthIndicator() {
	miso.AddHealthIndicator(miso.HealthIndicator{
		Name: "MySQL Component",
		CheckHealth: func(rail miso.Rail) bool {
			db, err := m.mysql().DB()
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
	return module().initFromProp(rail)
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
	return module().init(rail, p)
}

// Get MySQL Connection.
func GetMySQL() *gorm.DB {
	return module().mysql()
}

// Check whether mysql client is initialized
func IsMySQLInitialized() bool {
	return module().initialized()
}

func mysqlBootstrap(app *miso.MisoApp, rail miso.Rail) error {
	m := appModule(app)

	if e := InitMySQLFromProp(rail); e != nil {
		return fmt.Errorf("failed to establish connection to MySQL, %w", e)
	}

	// run bootstrap callbacks
	m.runBootstrapCallbacks(rail)

	// register health indicator
	m.registerHealthIndicator()

	return nil
}

func mysqlBootstrapCondition(app *miso.MisoApp, rail miso.Rail) (bool, error) {
	return app.Config().GetPropBool(PropMySQLEnabled), nil
}

func AddMySQLBootstrapCallback(cbk MySQLBootstrapCallback) {
	module().addMySQLBootstrapCallback(cbk)
}
