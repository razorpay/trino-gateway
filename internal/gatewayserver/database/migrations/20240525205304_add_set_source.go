package migration

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20240525205304, Down20240525205304)
}

func Up20240525205304(tx *sql.Tx) error {
	var err error

	_, err = tx.Exec("ALTER TABLE `policies` ADD COLUMN `set_request_source` VARCHAR(255) DEFAULT '';")
	if err != nil {
		return err
	}
	return err
}

func Down20240525205304(tx *sql.Tx) error {
	var err error

	_, err = tx.Exec("ALTER TABLE `policies` DROP COLUMN `set_request_source`;")
	if err != nil {
		return err
	}
	return err
}
