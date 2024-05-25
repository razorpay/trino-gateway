package migration

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20240524205304, Down20240524205304)
}

func Up20240524205304(tx *sql.Tx) error {
	var err error

	_, err = tx.Exec("ALTER TABLE `policies` ADD COLUMN `is_auth_delegated` BOOL DEFAULT false;")
	if err != nil {
		return err
	}
	return err
}

func Down20240524205304(tx *sql.Tx) error {
	var err error

	_, err = tx.Exec("ALTER TABLE `policies` DROP COLUMN `is_auth_delegated`;")
	if err != nil {
		return err
	}
	return err
}
