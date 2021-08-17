package spine

import (
	"context"

	"github.com/razorpay/trino-gateway/pkg/spine/db"
	"gorm.io/plugin/dbresolver"

	"gorm.io/gorm"
)

const updatedAtField = "updated_at"

type Repo struct {
	Db *db.DB
}

// FindByID fetches the record which matches the ID provided from the entity defined by receiver
// and the result will be loaded into receiver
func (repo Repo) FindByID(ctx context.Context, receiver IModel, id string) error {
	q := repo.DBInstance(ctx).Where("id = ?", id).First(receiver)

	return GetDBError(q)
}

// FindByIDs fetches the all the records which matches the IDs provided from the entity defined by receivers
// and the result will be loaded into receivers
func (repo Repo) FindByIDs(ctx context.Context, receivers interface{}, ids []string) error {
	q := repo.DBInstance(ctx).Where(AttributeID+" in (?)", ids).Find(receivers)

	return GetDBError(q)
}

// Create inserts a new record in the entity defined by the receiver
// all data filled in the receiver will inserted
func (repo Repo) Create(ctx context.Context, receiver IModel) error {
	if err := receiver.SetDefaults(); err != nil {
		return err
	}

	if err := receiver.Validate(); err != nil {
		return err
	}

	q := repo.DBInstance(ctx).Create(receiver)

	return GetDBError(q)
}

// CreateInBatches insert the value in batches into database
func (repo Repo) CreateInBatches(ctx context.Context, receivers interface{}, batchSize int) error {
	q := repo.DBInstance(ctx).CreateInBatches(receivers, batchSize)

	return GetDBError(q)
}

// Update will update the given receiver model with respect to primary key / id available in it.
// If selective list is non empty, only those fields which are present in the list will be updated.
// Note: When using selectiveList `updated_at` field need not be passed in the list.
func (repo Repo) Update(ctx context.Context, receiver IModel, selectiveList ...string) error {
	if len(selectiveList) > 0 {
		selectiveList = append(selectiveList, updatedAtField)
	}
	return repo.updateSelective(ctx, receiver, selectiveList...)
}

// Delete deletes the given model
// Soft or hard delete of model depends on the models implementation
// if the model composites SoftDeletableModel then it'll be soft deleted
func (repo Repo) Delete(ctx context.Context, receiver IModel) error {
	q := repo.DBInstance(ctx).Delete(receiver)

	return GetDBError(q)
}

// FineMany will fetch multiple records form the entity defined by receiver which matched the condition provided
// note: this wont work for in clause. can be used only for `=` conditions
func (repo Repo) FindMany(
	ctx context.Context,
	receivers interface{},
	condition map[string]interface{}) error {

	q := repo.DBInstance(ctx).Where(condition).Find(receivers)

	return GetDBError(q)
}

// Preload preload associations with given conditions
// repo.Preload(ctx, "Orders", "state NOT IN (?)", "cancelled").FindMany(ctx, &users)
func (repo Repo) Preload(ctx context.Context, query string, args ...interface{}) *Repo {
	return &Repo{
		Db: repo.Db.Preload(ctx, query, args),
	}
}

// Transaction will manage the execution inside a transactions
// adds the txn db in the context for downstream use case
func (repo Repo) Transaction(ctx context.Context, fc func(ctx context.Context) error) error {
	var err = repo.DBInstance(ctx).Transaction(func(tx *gorm.DB) error {

		// This will ensure that when db.Instance(context) we return the txn on the context
		// & all repo queries are done on this txn. Refer usage in test.
		if err := fc(context.WithValue(ctx, db.ContextKeyDatabase, tx)); err != nil {
			return err
		}

		return GetDBError(tx)
	})

	if err == nil {
		return nil
	}

	// tx.Commit can throw an error which will not be an IError
	if iErr, ok := err.(error); ok {
		return iErr
	}

	// use the default code and wrap err in internal
	return err
}

// IsTransactionActive returns true if a transaction is active
func (repo Repo) IsTransactionActive(ctx context.Context) bool {
	_, ok := ctx.Value(db.ContextKeyDatabase).(*gorm.DB)
	return ok
}

// DBInstance returns gorm instance.
// If replicas are specified, for Query, Row callback, will use replicas, unless Write mode specified.
// For Raw callback, statements are considered read-only and will use replicas if the SQL starts with SELECT.
//
func (repo Repo) DBInstance(ctx context.Context) *gorm.DB {
	return repo.Db.Instance(ctx)
}

// WriteDBInstance returns a gorm instance of source/primary db connection.
func (repo Repo) WriteDBInstance(ctx context.Context) *gorm.DB {
	return repo.DBInstance(ctx).Clauses(dbresolver.Write)
}

// WarmStorageDBInstance returns gorm instance of source/primary db connection.
func (repo Repo) WarmStorageDBInstance(ctx context.Context) *gorm.DB {
	return repo.DBInstance(ctx).Clauses(dbresolver.Use(db.WarmStorageDBResolverName))
}

// updateSelective will update the given receiver model with respect to primary key / id available in it.
// If selective list is non empty, only those fields which are present in the list will be updated.
// Note: When using selectiveList `updated_at` field also needs to be explicitly passed in the selectiveList.
func (repo Repo) updateSelective(ctx context.Context, receiver IModel, selectiveList ...string) error {
	q := repo.DBInstance(ctx).Model(receiver)

	if len(selectiveList) > 0 {
		q = q.Select(selectiveList)
	}

	q = q.Updates(receiver)

	if q.RowsAffected == 0 {
		return NoRowAffected
	}

	return GetDBError(q)
}
