package mysql

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/middleware/dbquery"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/pair"
	"github.com/curtisnewbie/miso/util/strutil"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	minimumConnParam = "parseTime=true&loc=Local"
)

func init() {
	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Bootstrap MySQL",
		Bootstrap: mysqlBootstrap,
		Condition: mysqlBootstrapCondition,
		Order:     miso.BootstrapOrderL1,
	})
}

var (
	slowThreshold = 500 * time.Millisecond
	dbLogger      = dbquery.NewGormLogger(logger.Config{SlowThreshold: slowThreshold, LogLevel: logger.Warn})
	module        = miso.InitAppModuleFunc(func() *mysqlModule {
		return &mysqlModule{
			mu: &sync.RWMutex{},
		}
	})
)

type MySQLBootstrapCallback func(rail miso.Rail, db *gorm.DB) error

type mysqlModule struct {
	mu                 *sync.RWMutex
	conn               *gorm.DB
	bootstrapCallbacks []MySQLBootstrapCallback
	managed            map[string]*gorm.DB
}

func (m *mysqlModule) getAllManaged() []pair.Pair[string, *gorm.DB] {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clt := []pair.Pair[string, *gorm.DB]{}
	if len(m.managed) < 1 {
		return clt
	}

	debug := miso.IsDebugLevel() || !miso.IsProdMode() || miso.GetPropBool(PropMySQLLogSQL)
	for k, v := range m.managed {
		cp := v
		if debug {
			cp = v.Debug()
		}
		clt = append(clt, pair.New(k, cp))
	}
	return clt
}

func (m *mysqlModule) getManaged(name string) *gorm.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.managed[name]
	if !ok {
		return nil
	}

	if miso.IsDebugLevel() || !miso.IsProdMode() || miso.GetPropBool(PropMySQLLogSQL) {
		return v.Debug()
	}

	return v
}

func (m *mysqlModule) initManaged(rail miso.Rail) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.managed = map[string]*gorm.DB{}
	for _, n := range miso.GetPropChild("mysql.managed") {
		n = strings.ToLower(n)
		pm := map[string]any{
			"name": n,
		}
		prepareStmt := true
		{
			k := strutil.NamedSprintf(PropMySQLManagedPrepareStmt, pm)
			if miso.HasProp(k) {
				prepareStmt = miso.GetPropBool(k)
			}
		}

		p := MySQLConnParam{
			User:            miso.GetPropStr(strutil.NamedSprintf(PropMySQLManagedUser, pm)),
			Password:        miso.GetPropStr(strutil.NamedSprintf(PropMySQLManagedPassword, pm)),
			Schema:          miso.GetPropStr(strutil.NamedSprintf(PropMySQLManagedSchema, pm)),
			Host:            miso.GetPropStr(strutil.NamedSprintf(PropMySQLManagedHost, pm)),
			Port:            miso.GetPropInt(strutil.NamedSprintf(PropMySQLManagedPort, pm)),
			ConnParam:       strings.Join(miso.GetPropStrSlice(PropMySQLConnParam), "&"),
			MaxOpenConns:    miso.GetPropInt(PropMySQLMaxOpenConns),
			MaxIdleConns:    miso.GetPropInt(PropMySQLMaxIdleConns),
			MaxConnLifetime: miso.GetPropDur(PropMySQLConnLifetime, time.Minute),
			NotPrepareStmt:  !prepareStmt,
		}
		if p.ConnParam == "" {
			p.ConnParam = minimumConnParam
		}
		if p.Host == "" {
			p.ConnParam = "localhost"
		}

		conn, err := NewMySQLConn(rail, p)
		if err != nil {
			return errs.WrapErrf(err, "failed to create mysql connection for '%v', %v:***/%v", n, p.User, p.Schema)
		}
		m.managed[n] = conn
		rail.Infof("Initialized managed MySQL connection '%v'", n)
	}

	return nil
}

func (m *mysqlModule) initPrimary(rail miso.Rail, p MySQLConnParam) error {
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
		return errs.WrapErrf(err, "failed to create mysql connection, %v:***/%v", p.User, p.Schema)
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

	if miso.IsDebugLevel() || !miso.IsProdMode() || miso.GetPropBool(PropMySQLLogSQL) {
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
		User:            miso.GetPropStr(PropMySQLUser),
		Password:        miso.GetPropStr(PropMySQLPassword),
		Schema:          miso.GetPropStr(PropMySQLSchema),
		Host:            miso.GetPropStr(PropMySQLHost),
		Port:            miso.GetPropInt(PropMySQLPort),
		ConnParam:       strings.Join(miso.GetPropStrSlice(PropMySQLConnParam), "&"),
		MaxOpenConns:    miso.GetPropInt(PropMySQLMaxOpenConns),
		MaxIdleConns:    miso.GetPropInt(PropMySQLMaxIdleConns),
		MaxConnLifetime: miso.GetPropDur(PropMySQLConnLifetime, time.Minute),
		NotPrepareStmt:  !miso.GetPropBool(PropMySQLPrepareStmt),
	}
	return m.initPrimary(rail, p)
}

func (m *mysqlModule) runBootstrapCallbacks(rail miso.Rail) error {
	if len(m.bootstrapCallbacks) > 0 {
		db := GetMySQL()
		for _, cbk := range m.bootstrapCallbacks {
			if err := cbk(rail, db); err != nil {
				return errs.WrapErrf(err, "failed to execute MySQLBootstrapCallback")
			}
		}
	}
	return nil
}

func (m *mysqlModule) registerHealthIndicator() {
	miso.AddHealthIndicator(miso.HealthIndicator{
		Name: "MySQL Component",
		CheckHealth: func(rail miso.Rail) bool {
			dbs := []pair.Pair[string, *gorm.DB]{}
			dbs = append(dbs, pair.New("primary", m.mysql()))
			dbs = append(dbs, m.getAllManaged()...)

			for _, d := range dbs {
				db, err := d.Right.DB()
				if err != nil {
					rail.Errorf("Failed to get MySQL DB (%v), %v", d.Left, err)
					return false
				}
				err = db.Ping()
				if err != nil {
					rail.Errorf("Failed to ping MySQL (%v), %v", d.Left, err)
					return false
				}
				rail.Debugf("MySQL %v Healthcheck passed", d.Left)
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
	NotPrepareStmt  bool
}

// Create new MySQL connection
func NewMySQLConn(rail miso.Rail, p MySQLConnParam) (*gorm.DB, error) {
	p.ConnParam = strings.TrimSpace(p.ConnParam)
	if p.ConnParam != "" && !strings.HasPrefix(p.ConnParam, "?") {
		p.ConnParam = "?" + p.ConnParam
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s", p.User, p.Password, p.Host, p.Port, p.Schema, p.ConnParam)
	rail.Infof("Connecting to database '%s:%d/%s' with params: '%s' (MaxLifetime: %v, MaxOpen: %v, MaxIdle: %v, PrepareStmt: %v)", p.Host, p.Port, p.Schema, p.ConnParam, p.MaxConnLifetime, p.MaxOpenConns, p.MaxIdleConns, !p.NotPrepareStmt)

	cfg := &gorm.Config{
		PrepareStmt: !p.NotPrepareStmt, CreateBatchSize: 100,
		Logger: dbLogger,
	}
	conn, err := gorm.Open(mysql.Open(dsn), cfg)
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
	return module().initPrimary(rail, p)
}

// Get MySQL Connection.
func GetMySQL() *gorm.DB {
	return module().mysql()
}

// Get Managed MySQL Connection.
func GetManaged(name string) *gorm.DB {
	return module().getManaged(strings.ToLower(name))
}

// Check whether mysql client is initialized
func IsMySQLInitialized() bool {
	return module().initialized()
}

func mysqlBootstrap(rail miso.Rail) error {
	m := module()

	if e := InitMySQLFromProp(rail); e != nil {
		return errs.WrapErrf(e, "failed to establish connection to MySQL")
	}
	if e := m.initManaged(rail); e != nil {
		return errs.WrapErrf(e, "failed to establish connection to MySQL")
	}

	// run bootstrap callbacks
	m.runBootstrapCallbacks(rail)

	// register health indicator
	m.registerHealthIndicator()

	dbquery.ImplGetPrimaryDBFunc(func() *gorm.DB { return GetMySQL() })

	if logSql() {
		colorful := false
		if miso.GetPropStrTrimmed(miso.PropLoggingRollingFile) == "" {
			colorful = true
		}
		dbLogger.UpdateConfig(logger.Config{SlowThreshold: slowThreshold, LogLevel: logger.Info, Colorful: colorful})
	}

	return nil
}

func mysqlBootstrapCondition(rail miso.Rail) (bool, error) {
	return miso.GetPropBool(PropMySQLEnabled), nil
}

func AddMySQLBootstrapCallback(cbk MySQLBootstrapCallback) {
	module().addMySQLBootstrapCallback(cbk)
}

func logSql() bool {
	return miso.IsDebugLevel() || !miso.IsProdMode() || miso.GetPropBool(PropMySQLLogSQL)
}

func ShowGrants(rail miso.Rail, db *gorm.DB) ([]string, error) {
	var grants []string
	_, err := dbquery.NewQueryRail(rail, db).Raw(`SHOW GRANTS`).Scan(&grants)
	if err != nil {
		return nil, err
	}
	return grants, nil
}

func LogShowGrants(rail miso.Rail, db *gorm.DB) {
	grants, err := ShowGrants(rail, db)
	if err != nil {
		rail.Warnf("SHOW GRANTS failed, %v", err)
		return
	}
	sb := strings.Builder{}
	for _, g := range grants {
		if sb.Len() > 0 {
			sb.WriteRune('\n')
		}
		sb.WriteString("- " + g)
	}
	rail.Infof("SHOW GRANTS:\n%v", sb.String())
}
