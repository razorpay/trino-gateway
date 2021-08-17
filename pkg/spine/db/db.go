// Package db has specific primitives for database config & connections.
//
// Usage:
// -    E.g. db.NewDb(&c), where c must implement IConfigReader and default use case is to just use Config struct.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm/logger"

	// tracingIntegration "github.com/razorpay/goutils/tracing/integrations"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

const (
	// PostgresConnectionDSNFormat is postgres connection on path format for gorm.
	// E.g. host=localhost:3306 dbname=app sslmode=require user=app password=password
	PostgresConnectionDSNFormat           = "host=%s port=%d dbname=%s sslmode=%s user=%s password=%s"
	PostgresConnectionDSNFormatWithSchema = "host=%s port=%d dbname=%s sslmode=%s user=%s password=%s search_path=%s"

	// MysqlConnectionDSNFormat is mysql connection path format for gorm.
	// E.g. app:password@tcp(localhost:3306)/app?charset=utf8&parseTime=True&loc=Local
	MysqlConnectionDSNFormat = "%s:%s@%s(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local"

	// WarmStorageDBResolverName string used by db resolver to use warm storage db instance
	WarmStorageDBResolverName = "warm_storage"
)

const (
	DialectMySQL    string = "mysql"
	DialectPostgres string = "postgres"
)

type contextKey int

const (
	// used to set the db instance in context in case of transactions
	ContextKeyDatabase contextKey = iota
)

var (
	ErrorUndefinedDialect = errors.New("dialect for the db is not defined")
)

// ConnectionReader has methods required to open a new connection to a database.
// It answers questions for - What to connect to ?.
type IConnectionReader interface {
	GetDialect() string
	GetConnectionPath() string
}

// IConnectionPoolConfigReader has methods to manage connection/connections with a database.
// It answers questions for - How to manage connection/s with a database ?.
type IConnectionPoolConfigReader interface {
	GetMaxIdleConnections() int
	GetMaxOpenConnections() int
	GetConnMaxLifetime() time.Duration
}

// IConfigReader interface has methods to read various DB configurations.
type IConfigReader interface {
	IConnectionReader
	IConnectionPoolConfigReader
	IsDebugMode() bool
}

// ConnectionConfig implements ConnectionReader.
type ConnectionConfig struct {
	Dialect  string
	Protocol string
	URL      string
	Port     int
	Username string
	Password string
	SslMode  string
	Name     string
	Schema   string //can be used in Postgres (optional)
}

// GetDialect returns a dialect identifier
func (c *ConnectionConfig) GetDialect() string {
	return c.Dialect
}

// GetConnectionPath returns connection string to be used by gorm basis dialect.
func (c *ConnectionConfig) GetConnectionPath() string {
	switch c.Dialect {
	case DialectPostgres:
		if c.Schema == "" {
			return fmt.Sprintf(PostgresConnectionDSNFormat, c.URL, c.Port, c.Name, c.SslMode, c.Username, c.Password)
		}
		return fmt.Sprintf(PostgresConnectionDSNFormatWithSchema, c.URL, c.Port, c.Name, c.SslMode, c.Username, c.Password, c.Schema)
	case DialectMySQL:
		return fmt.Sprintf(MysqlConnectionDSNFormat, c.Username, c.Password, c.Protocol, c.URL, c.Port, c.Name)
	default:
		return ""
	}
}

// ConnectionPoolConfig implements IConnectionPoolConfigReader
type ConnectionPoolConfig struct {
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
}

// GetMaxOpenConnections returns max open connections for the db.
func (c *ConnectionPoolConfig) GetMaxOpenConnections() int {
	return c.MaxOpenConnections
}

// GetMaxIdleConnections returns max idle connections for the db.
func (c *ConnectionPoolConfig) GetMaxIdleConnections() int {
	return c.MaxIdleConnections
}

// GetConnMaxLifetime returns configurable max lifetime of any connection of db.
func (c *ConnectionPoolConfig) GetConnMaxLifetime() time.Duration {
	return c.ConnectionMaxLifetime
}

// Config implements IConfigReader and holds configuration for the DB.
type Config struct {
	ConnectionConfig
	ConnectionPoolConfig
	Debug bool
}

// IsDebugMode returns true if the debug logs for the DB are to be enabled
func (c *Config) IsDebugMode() bool {
	return c.Debug
}

// DB is the specific wrapper holding gorm db instance.
type DB struct {
	configReader IConfigReader
	dialector    gorm.Dialector
	gormConfig   *gorm.Config
	instance     *gorm.DB
}

// GormConfig if set, will override the default DB.gormConfig used
// while opening connection with gorm.
func GormConfig(c *gorm.Config) func(*DB) error {
	return func(db *DB) error {
		db.gormConfig = c
		return nil
	}
}

// Dialector if set, will skip the default initialisation of the DB.dialector.
func Dialector(gd gorm.Dialector) func(*DB) error {
	return func(db *DB) error {
		db.dialector = gd
		return nil
	}
}

// NewDb instantiates Db and connects to database.
//
// Use options to set gorm.Config and gorm.Dialector.
// Dialector can be set by using db.Dialector() in the options.
// Similarly gorm config can be set by using db.GormConfig in the options.
//
// If gorm.Config is not set in the options, default will be used.
//
// If the gorm.Dialector is not set in the options, a default dialector
// will be created based on dialect & connection from IConfigReader.GetDialect() &
// IConfigReader.GetConnectionPath()
func NewDb(cr IConfigReader, options ...func(*DB) error) (*DB, error) {
	if cr == nil {
		cr = &Config{}
	}

	db := &DB{configReader: cr}

	for _, option := range options {
		if err := option(db); err != nil {
			return nil, err
		}
	}

	if db.dialector == nil {
		if err := db.initDialector(); err != nil {
			return nil, err
		}
	}

	if db.gormConfig == nil {
		db.gormConfig = &gorm.Config{
			AllowGlobalUpdate:      false,
			SkipDefaultTransaction: true,
			PrepareStmt:            true,
			// Set log level based on debug mode
			Logger: logger.Default.LogMode(getLogLevelByDebugMode(cr.IsDebugMode())),
		}
	}

	if err := db.connect(); err != nil {
		return nil, err
	}

	return db, nil
}

// Replicas provide the capability to set read replicas. An array of replicas
// i.e gorm.Dialector can be passed. The replica will be chosen randomly.
// For the replicas, configuration for the connection also need to be passed to
// set connection properties such as idle connections, max connections etc..
func (db *DB) Replicas(dls []gorm.Dialector, ccfr IConnectionPoolConfigReader) error {
	return db.instance.Use(
		dbresolver.Register(
			dbresolver.Config{
				Replicas: dls,
				Policy:   dbresolver.RandomPolicy{},
			}).
			SetMaxIdleConns(ccfr.GetMaxIdleConnections()).
			SetConnMaxLifetime(ccfr.GetConnMaxLifetime()).
			SetMaxOpenConns(ccfr.GetMaxOpenConnections()),
	)
}

// WarmStorageDB provide the capability to set warm storage db instances.
func (db *DB) WarmStorageDB(dls []gorm.Dialector, ccfr IConnectionPoolConfigReader) error {
	return db.instance.Use(
		dbresolver.Register(
			dbresolver.Config{
				Sources: dls,
				Policy:  dbresolver.RandomPolicy{},
			},
			WarmStorageDBResolverName,
		).
			SetMaxIdleConns(ccfr.GetMaxIdleConnections()).
			SetConnMaxLifetime(ccfr.GetConnMaxLifetime()).
			SetMaxOpenConns(ccfr.GetMaxOpenConnections()),
	)
}

// copy returns a copy of the instance of DB.
// Note: It is not a deep copy.
func (db *DB) copy() *DB {
	return &DB{
		instance: db.instance,
	}
}

// Preload preloads associations with given conditions &
// creates a new instance of DB which can be set in the Repo.
// newDB := db.Preload(ctx, "Orders", "state NOT IN (?)", "cancelled")
// repo := Repo{DB : newDB}
func (db *DB) Preload(ctx context.Context, query string, args ...interface{}) *DB {
	tx := db.copy()
	tx.instance = db.instance.Preload(query, args)
	return tx
}

// Instance returns underlying instance of gorm db.
// If the transaction/session in progress then it'll return
// the *gorm.DB from the context.
func (db *DB) Instance(ctx context.Context) *gorm.DB {
	if instance, ok := ctx.Value(ContextKeyDatabase).(*gorm.DB); ok {
		return instance
	}
	return db.instance
}

// Session creates a new session with the session and
// returns a new new instance of DB.
// newDB := DB.Session(session)
// repo := Repo{DB: newDB}
func (db *DB) Session(session *gorm.Session) *DB {
	newDB := db.copy()
	newDB.instance = db.instance.Session(session)
	return newDB
}

// Alive executes a select query and checks if connection exists and is alive.
func (db *DB) Alive() error {
	if dbi, err := db.instance.DB(); err != nil {
		return err
	} else {
		return dbi.Ping()
	}
}

func (db *DB) Dialector(ctx context.Context) gorm.Dialector {
	return db.dialector
}

// initDialector initializes a new dialector for the DB using the connReader
func (db *DB) initDialector() (err error) {
	var d gorm.Dialector
	if d, err = getDialector(db.configReader); err == nil {
		db.dialector = d
	}
	return
}

// connect opens a gorm connection and configures other connection details.
func (db *DB) connect() error {
	var err error

	if db.instance, err = gorm.Open(db.dialector, db.gormConfig); err != nil {
		return err
	}

	var dbConn *sql.DB
	if dbConn, err = db.instance.DB(); err != nil {
		return err
	}
	dbConn.SetMaxIdleConns(db.configReader.GetMaxIdleConnections())
	dbConn.SetMaxOpenConns(db.configReader.GetMaxOpenConnections())
	dbConn.SetConnMaxLifetime(db.configReader.GetConnMaxLifetime() * time.Second)

	return nil
}

func getDialector(connReader IConnectionReader) (gorm.Dialector, error) {
	switch connReader.GetDialect() {
	case DialectMySQL:
		return mysql.Open(connReader.GetConnectionPath()), nil
	case DialectPostgres:
		return postgres.Open(connReader.GetConnectionPath()), nil
	default:
		return nil, ErrorUndefinedDialect
	}
}

// GetDialectors is a utility to create a gorm.Dialector from ConnectionReader
func GetDialectors(connections []IConnectionReader) ([]gorm.Dialector, error) {
	var err error
	var d gorm.Dialector
	var gds = make([]gorm.Dialector, 0, len(connections))
	for _, c := range connections {
		if d, err = getDialector(c); err != nil {
			return nil, err
		} else {
			gds = append(gds, d)
		}
	}
	return gds, nil
}

// getLogLevelByDebugMode return logger log level based on debug mode.
// If app db is in debug mode, make log level as info
// Default log level for gorm db is warning, overriding that by this method.
func getLogLevelByDebugMode(debug bool) logger.LogLevel {
	if debug == false {
		return logger.Silent
	} else {
		return logger.Info
	}
}
