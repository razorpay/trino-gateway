package migration

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20220107205304, Down20220107205304)
}

func Up20220107205304(tx *sql.Tx) error {
	var err error

	_, err = tx.Exec("ALTER TABLE `queries` MODIFY COLUMN `text` VARCHAR(500);")
	if err != nil {
		return err
	}
	return err
}

func Down20220107205304(tx *sql.Tx) error {
	var err error

	_, err = tx.Exec("ALTER TABLE `queries` MODIFY COLUMN `text` VARCHAR(255);")
	if err != nil {
		return err
	}
	return err
}
