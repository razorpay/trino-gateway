package migration

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20260622120000, Down20260622120000)
}

func Up20260622120000(tx *sql.Tx) error {
	var err error

	_, err = tx.Exec("ALTER TABLE `policies` ADD COLUMN `is_email_access_restricted` BOOL DEFAULT false;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE `policies` ADD COLUMN `allowed_user_emails` TEXT;")
	if err != nil {
		return err
	}
	return err
}

func Down20260622120000(tx *sql.Tx) error {
	var err error

	_, err = tx.Exec("ALTER TABLE `policies` DROP COLUMN `allowed_user_emails`;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE `policies` DROP COLUMN `is_email_access_restricted`;")
	if err != nil {
		return err
	}
	return err
}
