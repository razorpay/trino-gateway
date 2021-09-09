package spine_test

import (
	"context"
	goErr "errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	// error_module "github.com/razorpay/error-mapping-module"
	"github.com/razorpay/trino-gateway/pkg/errors"
	"github.com/razorpay/trino-gateway/pkg/spine"
	"github.com/razorpay/trino-gateway/pkg/spine/datatype"
	"github.com/razorpay/trino-gateway/pkg/spine/db"
)

func init() {
	errors.InitMapping(error_module.Mapper{}, []string{"pkg/spine"})
}

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

func (t *TestModel) Validate() errors.IError {
	return nil
}

func (t *TestModel) SetDefaults() errors.IError {
	return nil
}

type InvalidModel struct {
	TestModel
}

func (t *InvalidModel) Validate() errors.IError {
	return spine.GetValidationError(goErr.New("error"))
}

type SoftDeleteModel struct {
	spine.SoftDeletableModel
	name string
}

func (t *SoftDeleteModel) EntityName() string {
	return "model"
}

func (t *SoftDeleteModel) TableName() string {
	return "model"
}

func (t *SoftDeleteModel) GetID() string {
	return t.ID
}

func (t *SoftDeleteModel) Validate() errors.IError {
	return nil
}

func (t *SoftDeleteModel) SetDefaults() errors.IError {
	return nil
}

func TestFindByID(t *testing.T) {
	repo, mockdb := createRepo(t)

	mockdb.
		ExpectQuery(regexp.QuoteMeta("SELECT * FROM `model` WHERE id = ? ORDER BY `model`.`id` LIMIT 1")).
		WithArgs("1").
		WillReturnRows(
			sqlmock.
				NewRows([]string{"id", "name"}).
				AddRow(1, "name1"))

	model := TestModel{}
	err := repo.FindByID(context.TODO(), &model, "1")
	assert.Nil(t, err)
	assert.Equal(t, model.Name, "name1")
	assert.Equal(t, model.ID, "1")
}

func TestFindByIDs(t *testing.T) {
	repo, mockdb := createRepo(t)

	mockdb.
		ExpectQuery(regexp.QuoteMeta("SELECT * FROM `model` WHERE id in (?,?)")).
		WithArgs("1", "2").
		WillReturnRows(
			sqlmock.
				NewRows([]string{"id", "name"}).
				AddRow("1", "name1").
				AddRow("2", "name2"))

	var models []TestModel
	err := repo.FindByIDs(context.TODO(), &models, []string{"1", "2"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(models))
}

func TestCreate(t *testing.T) {
	repo, mockdb := createRepo(t)

	staticTime := time.Date(2020, time.May, 19, 1, 2, 3, 4, time.UTC)

	mockdb.ExpectBegin()
	mockdb.
		ExpectExec(
			regexp.QuoteMeta("INSERT INTO `model` (`id`,`created_at`,`updated_at`,`name`) VALUES (?,?,?,?)")).
		WithArgs(sqlmock.AnyArg(), staticTime.Unix(), staticTime.Unix(), "test").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockdb.ExpectCommit()

	model := TestModel{
		Name: "test",
	}

	pg := monkey.Patch(time.Now, func() time.Time { return staticTime })
	defer pg.Unpatch()

	err := repo.Create(context.TODO(), &model)
	assert.Nil(t, err)
	assert.Equal(t, "test", model.Name)
	assert.Equal(t, staticTime.Unix(), model.GetCreatedAt())
	assert.Equal(t, staticTime.Unix(), model.GetUpdatedAt())
	assert.Nil(t, datatype.IsRZPID(model.ID))
}

func TestCreate_ValidationFailure(t *testing.T) {
	repo, _ := createRepo(t)

	model := InvalidModel{}

	err := repo.Create(context.TODO(), &model)
	assert.NotNil(t, err)
	assert.Equal(t, "validation_failure: error", err.Error())
}

func TestCreateInBatches(t *testing.T) {
	repo, mockdb := createRepo(t)

	staticTime := time.Date(2020, time.May, 19, 1, 2, 3, 4, time.UTC)

	mockdb.ExpectBegin()
	mockdb.
		ExpectExec(
			regexp.QuoteMeta("INSERT INTO `model` (`id`,`created_at`,`updated_at`,`name`) VALUES (?,?,?,?)")).
		WithArgs(sqlmock.AnyArg(), staticTime.Unix(), staticTime.Unix(), "test1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockdb.ExpectCommit()

	model1 := TestModel{
		Name: "test1",
	}

	models := []TestModel{model1}

	pg := monkey.Patch(time.Now, func() time.Time { return staticTime })
	defer pg.Unpatch()

	err := repo.CreateInBatches(context.TODO(), &models, 1)
	assert.Nil(t, err)
	model := models[0]
	assert.Equal(t, "test1", model.Name)
}

func TestTransactionCommit(t *testing.T) {
	repo, mockdb := createRepo(t)
	mockdb.ExpectBegin()
	mockdb.ExpectCommit()

	err := repo.Transaction(context.TODO(), func(ctx context.Context) errors.IError {
		return nil
	})

	assert.Nil(t, err)
}

func TestTransactionRollback(t *testing.T) {
	repo, mockdb := createRepo(t)
	mockdb.ExpectBegin()
	mockdb.ExpectRollback()

	err := repo.Transaction(context.TODO(), func(ctx context.Context) errors.IError {
		return spine.DBError.New("failed to execute query")
	})

	assert.Equal(t, string("db_error: default"), err.Error())
}

func TestUpdate(t *testing.T) {
	repo, mockdb := createRepo(t)

	staticTime := time.Date(2020, time.May, 19, 1, 2, 3, 4, time.UTC)

	mockdb.ExpectBegin()
	mockdb.
		ExpectExec(regexp.QuoteMeta("UPDATE `model` SET `created_at`=?,`updated_at`=?,`name`=? WHERE `id` = ?")).
		WithArgs(int64(123), 1589850123, "test", "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockdb.ExpectCommit()

	model := TestModel{
		Model: spine.Model{
			ID:        "1",
			CreatedAt: int64(123),
		},
		Name: "test",
	}

	pg := monkey.Patch(time.Now, func() time.Time { return staticTime })
	defer pg.Unpatch()

	err := repo.Update(context.TODO(), &model)
	assert.Nil(t, err)
	assert.Equal(t, "test", model.Name)
	assert.Equal(t, int64(123), model.CreatedAt)
	assert.Equal(t, staticTime.Unix(), model.UpdatedAt)
}

func TestDelete_HardDelete(t *testing.T) {
	repo, mockdb := createRepo(t)

	mockdb.ExpectBegin()
	mockdb.
		ExpectExec(regexp.QuoteMeta("DELETE FROM `model` WHERE `model`.`id` = ?")).
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockdb.ExpectCommit()

	model := TestModel{
		Model: spine.Model{
			ID:        "1",
			CreatedAt: int64(123),
		},
		Name: "test",
	}

	err := repo.Delete(context.TODO(), &model)
	assert.Nil(t, err)
}

func TestDelete_SoftDelete(t *testing.T) {
	repo, mockdb := createRepo(t)
	staticTime := time.Date(2020, time.May, 19, 1, 2, 3, 4, time.Local)

	mockdb.ExpectBegin()
	mockdb.
		ExpectExec(regexp.QuoteMeta("UPDATE `model` SET `deleted_at`=? WHERE `model`.`id` = ? AND `model`.`deleted_at` IS NULL")).
		WithArgs(staticTime.Unix(), "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockdb.ExpectCommit()

	model := SoftDeleteModel{
		SoftDeletableModel: spine.SoftDeletableModel{
			Model: spine.Model{
				ID: "1",
			},
		},
	}

	pg := monkey.Patch(time.Now, func() time.Time { return staticTime })
	defer pg.Unpatch()

	err := repo.Delete(context.TODO(), &model)
	assert.Nil(t, err)
}

// TestUpdate_NoID verifies that if the id of the model is empty,
// update should not take place based on the other `where` conditions
func TestUpdate_NoID(t *testing.T) {
	repo, mockdb := createRepo(t)

	staticTime := time.Date(2020, time.May, 19, 1, 2, 3, 4, time.UTC)

	pg := monkey.Patch(time.Now, func() time.Time { return staticTime })
	defer pg.Unpatch()

	model := TestModel{}
	model.ID = ""
	model.Name = "test"
	model.CreatedAt = int64(123)

	mockdb.ExpectBegin()
	mockdb.ExpectRollback()
	err := repo.Update(context.TODO(), &model)
	assert.NotNil(t, err)
	assert.Equal(t, spine.NoRowAffected, err.Class())
	assert.Equal(t, err.Unwrap().Error(), "WHERE conditions required")

	mockdb.ExpectBegin()
	mockdb.ExpectRollback()
	err = repo.Update(context.TODO(), &model, "updated_at", "name")
	assert.NotNil(t, err)
	assert.Equal(t, spine.NoRowAffected, err.Class())
	assert.Equal(t, err.Unwrap().Error(), "WHERE conditions required")
}

func TestUpdateNoRowAffected(t *testing.T) {
	repo, mockdb := createRepo(t)

	staticTime := time.Date(2020, time.May, 19, 1, 2, 3, 4, time.UTC)

	mockdb.ExpectBegin()
	mockdb.
		ExpectExec(regexp.QuoteMeta("UPDATE `model` SET `created_at`=?,`updated_at`=?,`name`=? WHERE `id` = ?")).
		WithArgs(int64(123), 1589850123, "test", "1")
	mockdb.ExpectCommit()

	model := TestModel{
		Model: spine.Model{
			ID:        "1",
			CreatedAt: int64(123),
		},
		Name: "test",
	}

	pg := monkey.Patch(time.Now, func() time.Time { return staticTime })
	defer pg.Unpatch()

	err := repo.Update(context.TODO(), &model)
	assert.NotNil(t, err)
	assert.Equal(t, spine.NoRowAffected, err.Class())
}

// updated_at not passed explicitly but still should be updated.
func TestUpdate_WithSelectiveList(t *testing.T) {
	repo, mockdb := createRepo(t)

	staticTime := time.Date(2020, time.May, 19, 1, 2, 3, 4, time.UTC)
	pg := monkey.Patch(time.Now, func() time.Time { return staticTime })
	defer pg.Unpatch()

	model := TestModel{
		Model: spine.Model{
			ID:        "1",
			CreatedAt: int64(1234567),
			UpdatedAt: int64(2345678),
		},
		Name: "new-name",
	}

	mockdb.ExpectBegin()
	mockdb.ExpectExec(regexp.QuoteMeta("UPDATE `model` SET `updated_at`=?,`name`=? WHERE `id` = ?")).
		WithArgs(1589850123, "new-name", "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockdb.ExpectCommit()

	err := repo.Update(context.TODO(), &model, "name")
	assert.Nil(t, err)
	assert.Equal(t, "new-name", model.Name)
	assert.Equal(t, int64(1234567), model.CreatedAt)
	assert.Equal(t, int64(1589850123), model.UpdatedAt)
}

// explicitly passing updated_at field in the list of fields to update.
func TestUpdateSelective_WithUpdatedAtField(t *testing.T) {
	repo, mockdb := createRepo(t)

	model := TestModel{
		Model: spine.Model{
			ID:        "1",
			CreatedAt: int64(1234567),
			UpdatedAt: int64(2345678),
		},
		Name: "new-name",
	}

	staticTime := time.Date(2020, time.April, 16, 1, 2, 3, 4, time.UTC)

	pg := monkey.Patch(time.Now, func() time.Time { return staticTime })
	defer pg.Unpatch()

	mockdb.ExpectBegin()
	mockdb.ExpectExec(regexp.QuoteMeta("UPDATE `model` SET `updated_at`=?,`name`=? WHERE `id` = ?")).
		WithArgs(staticTime.Unix(), "new-name", "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockdb.ExpectCommit()

	err := repo.Update(context.TODO(), &model, "name", "updated_at")

	assert.Nil(t, err)
	assert.Equal(t, "new-name", model.Name)
	assert.Equal(t, staticTime.Unix(), model.UpdatedAt)
}

func TestFindMany(t *testing.T) {
	repo, mockdb := createRepo(t)

	mockdb.
		ExpectQuery(regexp.QuoteMeta("SELECT * FROM `model`")).
		WillReturnRows(
			sqlmock.
				NewRows([]string{"id", "name"}).
				AddRow("1", "test").
				AddRow("2", "test2"))

	var models []TestModel

	err := repo.FindMany(context.TODO(), &models, map[string]interface{}{})
	assert.Nil(t, err)
	model := models[0]
	assert.Equal(t, "1", model.ID)
	model = models[1]
	assert.Equal(t, "2", model.ID)
}

func TestRepo_TransactionNestedCreateModel(t *testing.T) {
	repo, mockdb := createRepo(t)

	model := TestModel{
		Name: "test",
	}

	mockdb.ExpectBegin()
	mockdb.ExpectExec("SAVEPOINT \\w").WithArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	mockdb.ExpectExec(regexp.QuoteMeta("INSERT INTO `model` (`id`,`created_at`,`updated_at`,`name`) VALUES (?,?,?,?)")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "test").WillReturnResult(sqlmock.NewResult(1, 1))
	mockdb.ExpectCommit()

	err := repo.Transaction(context.TODO(), func(ctx context.Context) errors.IError {
		return repo.Transaction(ctx, func(ctx context.Context) errors.IError {
			return repo.Create(ctx, &model)
		})
	})

	// we make sure that all expectations were met
	if err := mockdb.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Nil(t, err)
}

func TestRepo_TransactionNestedCommitFail(t *testing.T) {
	repo, mockdb := createRepo(t)

	model := TestModel{
		Name: "test",
	}

	var internalErr = "unable to commit"

	mockdb.ExpectBegin()
	mockdb.ExpectExec("SAVEPOINT \\w").WithArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	mockdb.ExpectExec("SAVEPOINT \\w").WithArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	mockdb.ExpectExec(regexp.QuoteMeta("INSERT INTO `model` (`id`,`created_at`,`updated_at`,`name`) VALUES (?,?,?,?)")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "test").WillReturnResult(sqlmock.NewResult(1, 1))
	mockdb.ExpectCommit().WillReturnError(fmt.Errorf(internalErr))

	err := repo.Transaction(context.TODO(), func(ctx context.Context) errors.IError {
		return repo.Transaction(ctx, func(ctx context.Context) errors.IError {
			return repo.Transaction(ctx, func(ctx context.Context) errors.IError {
				return repo.Create(ctx, &model)
			})
		})
	})

	// we make sure that all expectations were met
	if err := mockdb.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NotNil(t, err)
	assert.Equal(t, err.Internal().Error(), internalErr)
	assert.Equal(t, err.Internal().Code(), errors.ErrorCode("default"))
	assert.Equal(t, err.Public().Error(), "The server encountered an error. The incident has been reported to admins.")
}

func TestRepo_TransactionNestedRollbacks(t *testing.T) {
	repo, mockdb := createRepo(t)

	model := TestModel{
		Name: "test",
	}

	var errStr = "error from inside a transaction"

	// Create query will throw an error
	mockdb.ExpectBegin()
	mockdb.ExpectExec("SAVEPOINT \\w").WithArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	mockdb.ExpectExec("SAVEPOINT \\w").WithArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	mockdb.ExpectExec(regexp.QuoteMeta("INSERT INTO `model` (`id`,`created_at`,`updated_at`,`name`) VALUES (?,?,?,?)")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "test").
		WillReturnError(fmt.Errorf(errStr)) // returning error leading to rollbacks
	mockdb.ExpectExec("ROLLBACK TO SAVEPOINT \\w").WithArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	mockdb.ExpectExec("ROLLBACK TO SAVEPOINT \\w").WithArgs().WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Transaction(context.TODO(), func(ctx context.Context) errors.IError {
		return repo.Transaction(ctx, func(ctx context.Context) errors.IError {
			return repo.Transaction(ctx, func(ctx context.Context) errors.IError {
				return repo.Create(ctx, &model)
			})
		})
	})

	// we make sure that all expectations were met
	if err := mockdb.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NotNil(t, err)
	assert.Equal(t, err.Internal().Error(), errStr)
	assert.Equal(t, err.Internal().Code(), errors.ErrorCode("DB_ERROR"))
	assert.Equal(t, err.Public().Error(), "The server encountered an error. The incident has been reported to admins.")
}

func createRepo(t *testing.T) (spine.Repo, sqlmock.Sqlmock) {
	conn, mockdb, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	gdb, err := db.NewDb(getDefaultConfig(), db.Dialector(getGormDialectorForMock(conn)), db.GormConfig(&gorm.Config{}))
	assert.Nil(t, err)
	assert.NotNil(t, gdb)

	return spine.Repo{Db: gdb}, mockdb
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
