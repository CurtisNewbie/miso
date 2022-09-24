package config

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	// hikari recommends 1800000, so we do the same thing
	connMaxLifeTime = time.Minute * 30
	maxOpenConns    = 10
	maxIdleConns    = maxOpenConns // recommended to be the same as the maxOpenConns
)

var (
	// Global handle to the database
	dbHandle *gorm.DB

	// Global handle to the redis
	redisHandle *redis.Client
)

// Init handle to redis
func InitRedisFromConfig(config *RedisConfig) *redis.Client {
	return InitRedis(config.Address, config.Port, config.Username, config.Password, config.Database)
}

// Init handle to redis
func InitRedis(address string, port string, username string, password string, db int) *redis.Client {
	log.Printf("Connecting to redis '%v:%v', database: %v", address, port, db)
	var rdb *redis.Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", address, port),
		Password: password,
		DB:       db,
	})

	redisHandle = rdb
	log.Println("Redis Handle initialized")

	return rdb
}

/* Init Handle to the database */
func InitDBFromConfig(config *DBConfig) error {
	return InitDB(config.User, config.Password, config.Database, config.Host, config.Port)
}

/* Init Handle to the database */
func InitDB(user string, password string, dbname string, host string, port string) error {

	params := "charset=utf8mb4&parseTime=True&loc=Local&readTimeout=30s&writeTimeout=30s&timeout=3s"
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?%v", user, password, host, port, dbname, params)
	log.Printf("Connecting to database '%v:%v' with params: '%v'", host, port, params)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to Open DB Handle, err: %v", err)
		return err
	}

	sqlDb, err := db.DB()
	if err != nil {
		log.Printf("Get DB Handle from gorm failed, %v", err)
		return err
	}

	sqlDb.SetConnMaxLifetime(connMaxLifeTime)
	sqlDb.SetMaxOpenConns(maxOpenConns)
	sqlDb.SetMaxIdleConns(maxIdleConns)

	err = sqlDb.Ping() // make sure the handle is actually connected
	if err != nil {
		log.Printf("Ping DB Error, %v, connection may not be established", err)
		return err
	}

	log.Println("DB Handle initialized")

	dbHandle = db

	return nil
}

// Get DB Handle, must call InitDB(...) method before this method
func GetDB() *gorm.DB {
	if dbHandle == nil {
		panic("GetDB is called prior to the DB Handle initialization, this is illegal, see InitDB(...) method")
	}

	if IsProdMode() {
		return dbHandle
	}

	// not debug mode, enable debugging for printing SQLs
	return dbHandle.Debug()
}

// Get Redis Handle, must call InitRedis(...) method before this method
func GetRedis() *redis.Client {
	if redisHandle == nil {
		panic("GetRedis is called prior to the Redis Handle initialization, this is illegal, see InitRedis(...) method")
	}
	return redisHandle
}
