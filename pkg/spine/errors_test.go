package spine_test

import (
	"context"
	goerr "errors"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	error_module "github.com/razorpay/error-mapping-module"
	"github.com/razorpay/trino-gateway/pkg/errors"
	"github.com/razorpay/trino-gateway/pkg/spine"
	"github.com/razorpay/trino-gateway/pkg/spine/db"
)

func init() {
	errors.InitMapping(error_module.Mapper{}, []string{"pkg/spine"})
}

func TestQueryFailure(t *testing.T) {
	repo, mockdb := createRepo(t)

	mockdb.
		ExpectQuery(regexp.QuoteMeta("SELECT * FROM `model`")).
		WillReturnError(goerr.New("failed to execute db query"))

	var models []TestModel

	dberr := repo.FindMany(context.TODO(), &models, map[string]interface{}{})
	assert.NotNil(t, dberr)
	assert.Equal(t, spine.DBError, dberr.Class())
	assert.Equal(t, goerr.New("failed to execute db query"), dberr.Unwrap())
	assert.Equal(t, errors.ErrorCode("DB_ERROR"), dberr.Internal().Code())
}

func TestNoRecordError(t *testing.T) {
	mockdriver, _, err := sqlmock.New()

	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	dbInstance, err := db.NewDb(getDefaultConfig(), db.Dialector(mysql.New(mysql.Config{Conn: mockdriver, SkipInitializeWithVersion: true})))
	if err != nil {
		t.Errorf("error expected")
	}
	assert.Nil(t, err)

	di := dbInstance.Instance(context.TODO())
	di.Error = gorm.ErrRecordNotFound

	dberr := spine.GetDBError(di)

	assert.NotNil(t, dberr)
	assert.Equal(t, spine.RecordNotFound, dberr.Class())

	de, ok := spine.DialectError(di)
	assert.False(t, ok)
	assert.Nil(t, de)
}

func TestNewValidationError(t *testing.T) {
	verr := spine.GetValidationError(nil)

	assert.Nil(t, verr)

	err := goerr.New("validation failure")

	verr = spine.GetValidationError(err)

	assert.Equal(t, spine.ValidationFailure, verr.Class())
	assert.Equal(t, err, verr.Unwrap())
	assert.Equal(t, errors.ErrorCode("VALIDATION_FAILURE"), verr.Internal().Code())
}

func TestDialectErrorDefault(t *testing.T) {
	mockdriver, _, err := sqlmock.New()

	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	dbInstance, err := db.NewDb(getDefaultConfig(), db.Dialector(mysql.New(mysql.Config{Conn: mockdriver, SkipInitializeWithVersion: true})))
	if err != nil {
		t.Errorf("error expected")
	}
	assert.Nil(t, err)

	di := dbInstance.Instance(context.TODO())
	di.Error = &pq.Error{}

	de, ok := spine.DialectError(di)
	assert.True(t, ok)
	assert.Equal(t, "spine.pqError", reflect.TypeOf(de).String())

	err = de.ConstructError()
	assert.Nil(t, err)
}

func TestDialectError(t *testing.T) {
	mockdriver, _, err := sqlmock.New()

	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	dbInstance, err := db.NewDb(getDefaultConfig(), db.Dialector(mysql.New(mysql.Config{Conn: mockdriver, SkipInitializeWithVersion: true})))
	if err != nil {
		t.Errorf("error expected")
	}
	assert.Nil(t, err)

	di := dbInstance.Instance(context.TODO())
	di.Error = &pq.Error{
		Code: pq.ErrorCode("23505"),
	}

	de, ok := spine.DialectError(di)
	assert.True(t, ok)
	assert.Equal(t, "spine.pqError", reflect.TypeOf(de).String())

	ierr := de.ConstructError()
	assert.NotNil(t, ierr)
	assert.Equal(t, spine.UniqueConstraintViolation, ierr.Class())

	ierr = spine.GetDBError(di)
	assert.NotNil(t, ierr)
	assert.Equal(t, spine.UniqueConstraintViolation, ierr.Class())
}
