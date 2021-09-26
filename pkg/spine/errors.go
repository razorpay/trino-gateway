package spine

import (
	"errors"

	"github.com/go-sql-driver/mysql"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

const (
	PQCodeUniqueViolation    = "unique_violation"
	MySqlCodeUniqueViolation = 1062

	errDBError                   = "db_error"
	errNoRowAffected             = "no_row_affected"
	errRecordNotFound            = "record_not_found"
	errValidationFailure         = "validation_failure"
	errUniqueConstraintViolation = "unique_constraint_violation"
)

var (
	DBError                   = errors.New(errDBError)
	NoRowAffected             = errors.New(errNoRowAffected)
	RecordNotFound            = errors.New(errRecordNotFound)
	ValidationFailure         = errors.New(errValidationFailure)
	UniqueConstraintViolation = errors.New(errUniqueConstraintViolation)
)

// GetDBError accepts db instance and the details
// creates appropriate error based on the type of query result
// if there is no error then returns nil
func GetDBError(db *gorm.DB) error {
	if db.Error == nil {
		return nil
	}

	// check of error is specific to dialect
	if de, ok := DialectError(db); ok {
		// is the specific error is captured then return it
		// else try construct further errors
		if err := de.ConstructError(); err != nil {
			return err
		}
	}

	// Construct error based on type of db operation
	err := func() error {
		switch true {
		case errors.Is(db.Error, gorm.ErrRecordNotFound):
			return RecordNotFound

		default:
			return db.Error
		}
	}()

	// add specific details of error
	return err
}

// GetValidationError wraps the error and returns instance of ValidationError
// if the provided error is nil then it just returns nil
func GetValidationError(err error) error {
	if err != nil {
		return err
	}

	return nil
}

// DialectError returns true if the error is from dialect
func DialectError(d *gorm.DB) (IDialectError, bool) {
	switch d.Error.(type) {
	case *pq.Error:
		return pqError{d.Error.(*pq.Error)}, true
	case *mysql.MySQLError:
		return mysqlError{d.Error.(*mysql.MySQLError)}, true
	default:
		return nil, false
	}
}

// IDialectError interface to handler dialect related errors
type IDialectError interface {
	ConstructError() error
}

// pqError holds the error occurred by postgres
type pqError struct {
	err *pq.Error
}

// ConstructError will create appropriate error based on dialect
func (pqe pqError) ConstructError() error {
	switch pqe.err.Code.Name() {
	case PQCodeUniqueViolation:
		return pqe.err
	default:
		return nil
	}
}

type mysqlError struct {
	err *mysql.MySQLError
}

// ConstructError will create appropriate error based on dialect
func (msqle mysqlError) ConstructError() error {
	switch msqle.err.Number {
	case MySqlCodeUniqueViolation:
		return msqle.err

	default:
		return nil
	}
}
