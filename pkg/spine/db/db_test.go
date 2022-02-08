package db_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"gorm.io/plugin/dbresolver"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/razorpay/trino-gateway/pkg/spine"
	"github.com/razorpay/trino-gateway/pkg/spine/db"
)

type TestModel struct {
	spine.Model
	Name string `json:"name"`
}

func (t *TestModel) EntityName() string {
	return "model"
}

func (t *TestModel) TableName() string {
	return "model"
}

func (t *TestModel) GetID() string {
	return t.ID
}

func (t *TestModel) Validate() error {
	return nil
}

func (t *TestModel) SetDefaults() error {
	return nil
}

func TestGetConnectionPath(t *testing.T) {
	c := getDefaultConfig()
	// Asserts connection string for mysql dialect.
	assert.Equal(t, "user:pass@tcp(localhost:3307)/database?charset=utf8&parseTime=True&loc=Local", c.GetConnectionPath())
	// Asserts connection string for postgres dialect.
	c.Dialect = "postgres"
	assert.Equal(t, "host=localhost port=3307 dbname=database sslmode=require user=user password=pass", c.GetConnectionPath())

	// invalid dialect
	c.Dialect = "invalid"
	assert.Equal(t, "", c.GetConnectionPath())
}

func TestNewDb(t *testing.T) {
	tests := []struct {
		name    string
		err     string
		config  db.IConfigReader
		options func() ([]func(*db.DB) error, func())
	}{
		{
			name:   "success",
			config: getDefaultConfig(),
			options: func() ([]func(*db.DB) error, func()) {
				conn, _, err := sqlmock.New()
				assert.Nil(t, err)
				return []func(*db.DB) error{
					db.Dialector(getGormDialectorForMock(conn)),
				}, func() { _ = conn.Close() }
			},
		},
		{
			name:   "invalid dialect",
			err:    db.ErrorUndefinedDialect.Error(),
			config: &db.Config{ConnectionConfig: db.ConnectionConfig{Dialect: "invalid dialect"}},
			options: func() ([]func(*db.DB) error, func()) {
				return []func(*db.DB) error{}, func() {}
			},
		},
		{
			name:   "connect error: no mock",
			err:    "dial tcp .+:3307: connect: connection refused",
			config: getDefaultConfig(),
			options: func() ([]func(*db.DB) error, func()) {
				return []func(*db.DB) error{}, func() {}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			options, finish := testCase.options()
			defer finish()

			gdb, err := db.NewDb(testCase.config, options...)

			if testCase.err == "" {
				assert.Nil(t, err)
				assert.NotNil(t, gdb)
			} else {
				expr, e := regexp.Compile(testCase.err)
				assert.Nil(t, e)
				assert.NotNil(t, err)
				assert.Nil(t, gdb)
				assert.Regexp(t, expr, err.Error())
			}
		})
	}
}

func TestDB_Replica(t *testing.T) {
	defConn, _, err := sqlmock.New()
	assert.Nil(t, err)
	defer defConn.Close()

	replicaConn, replicaMock, err := sqlmock.New()
	assert.Nil(t, err)
	defer replicaConn.Close()

	sdb, err := db.NewDb(getDefaultConfig(), db.Dialector(getGormDialectorForMock(defConn)), db.GormConfig(&gorm.Config{}))
	assert.Nil(t, err)

	err = sdb.Replicas([]gorm.Dialector{getGormDialectorForMock(replicaConn)}, &db.ConnectionPoolConfig{
		MaxOpenConnections:    5,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 5 * time.Minute,
	})
	assert.Nil(t, err)

	model := TestModel{}

	// 1. Test that with replica select query goes to replica
	replicaMock.
		ExpectQuery(regexp.QuoteMeta("SELECT * FROM `model` WHERE id = ? ORDER BY `model`.`id` LIMIT 1")).
		WithArgs("1").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "name1"))

	tx := sdb.Instance(context.TODO()).Where("id = ?", "1").First(&model)
	assert.Nil(t, tx.Error)
	assert.Equal(t, model.ID, "1")
}

func TestDB_WarmStorageDB(t *testing.T) {
	defConn, _, err := sqlmock.New()
	assert.Nil(t, err)
	defer defConn.Close()

	warmStorageConn, warmStorageMock, err := sqlmock.New()
	assert.Nil(t, err)
	defer warmStorageConn.Close()

	newDB, err := db.NewDb(getDefaultConfig(), db.Dialector(getGormDialectorForMock(defConn)), db.GormConfig(&gorm.Config{}))
	assert.Nil(t, err)

	err = newDB.WarmStorageDB([]gorm.Dialector{getGormDialectorForMock(warmStorageConn)}, &db.ConnectionPoolConfig{
		MaxOpenConnections:    5,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 5 * time.Minute,
	})
	assert.Nil(t, err)

	model := TestModel{}

	// 1. Test that with warm storage select query goes to warm storage
	warmStorageMock.
		ExpectQuery(regexp.QuoteMeta("SELECT * FROM `model` WHERE id = ? ORDER BY `model`.`id` LIMIT 1")).
		WithArgs("1").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "name1"))

	tx := newDB.Instance(context.TODO()).Clauses(dbresolver.Use(db.WarmStorageDBResolverName)).Where("id = ?", "1").First(&model)
	assert.Nil(t, tx.Error)
	assert.Equal(t, model.ID, "1")
}

func TestDb_Alive(t *testing.T) {
	conn, _, err := sqlmock.New()
	assert.Nil(t, err)
	defer conn.Close()

	gdb, err := db.NewDb(getDefaultConfig(), db.Dialector(getGormDialectorForMock(conn)))
	assert.Nil(t, err)
	assert.NotNil(t, gdb)

	err = gdb.Alive()
	assert.Nil(t, err)
}

func TestDB_Instance(t *testing.T) {
	conn, _, err := sqlmock.New()
	assert.Nil(t, err)
	defer conn.Close()

	gdb1, err := db.NewDb(getDefaultConfig(), db.Dialector(getGormDialectorForMock(conn)))
	assert.Nil(t, err)
	assert.NotNil(t, gdb1)

	gdb2, err := db.NewDb(getDefaultConfig(), db.Dialector(getGormDialectorForMock(conn)))
	assert.Nil(t, err)
	assert.NotNil(t, gdb2)

	instance1 := gdb1.Instance(context.TODO())
	assert.NotNil(t, instance1)

	instance2 := gdb2.Instance(context.TODO())
	assert.NotNil(t, instance2)

	ctx := context.WithValue(context.TODO(), db.ContextKeyDatabase, instance2)
	tgdb := gdb1.Instance(ctx)
	assert.Equal(t, instance2, tgdb)
}

func getDefaultConfig() *db.Config {
	return &db.Config{
		ConnectionPoolConfig: db.ConnectionPoolConfig{
			MaxOpenConnections:    5,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 5 * time.Minute,
		},
		ConnectionConfig: db.ConnectionConfig{
			Dialect:  "mysql",
			Protocol: "tcp",
			URL:      "localhost",
			Port:     3307,
			Username: "user",
			Password: "pass",
			SslMode:  "require",
			Name:     "database",
		},
	}
}

func getGormDialectorForMock(conn gorm.ConnPool) gorm.Dialector {
	return mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true})
}
