package migration

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20211203205304, Down20211203205304)
}

func Up20211203205304(tx *sql.Tx) error {
	var err error

	_, err = tx.Exec("ALTER TABLE `backends` ADD COLUMN `is_healthy` BOOL DEFAULT false;")
	if err != nil {
		return err
	}
	return err
}

func Down20211203205304(tx *sql.Tx) error {
	var err error

	_, err = tx.Exec("ALTER TABLE `backends` DROP COLUMN `is_healthy`;")
	if err != nil {
		return err
	}
	return err
}
