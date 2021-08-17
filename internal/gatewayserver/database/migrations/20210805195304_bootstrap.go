package migration

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20210805195304, Down20210805195304)
}

func Up20210805195304(tx *sql.Tx) error {
	var err error
	_, err = tx.Exec(`
        CREATE TABLE backends (
            id VARCHAR(255) NOT NULL,
            hostname VARCHAR(255) NOT NULL,
            scheme ENUM('http', 'https') DEFAULT 'http',
            external_url VARCHAR(255) NOT NULL,
            is_enabled bool DEFAULT FALSE,
            uptime_schedule VARCHAR(255) DEFAULT '',
            created_at INT(11) NOT NULL,
            updated_at INT(11) NOT NULL,
            PRIMARY KEY (id),
            KEY users_created_at_index (created_at),
            KEY users_updated_at_index (updated_at)
        );`)
	if err != nil {
		return err
	}

	// groups is a keyword in mysql, so we use groups_
	_, err = tx.Exec("CREATE TABLE `groups_` (" +
		`id VARCHAR(255) NOT NULL,
            strategy ENUM('random', 'round_robin') DEFAULT 'random',
            is_enabled bool DEFAULT FALSE,
            created_at INT(11) NOT NULL,
            updated_at INT(11) NOT NULL,
            PRIMARY KEY (id),
            KEY users_created_at_index (created_at),
            KEY users_updated_at_index (updated_at)
        );`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`CREATE TABLE group_backends_mappings (
            id int AUTO_INCREMENT,
            group_id varchar(255),
            backend_id varchar(255),
            created_at int(11),
            updated_at int(11),
            PRIMARY KEY (id),
            KEY users_created_at_index (created_at),
            KEY users_updated_at_index (updated_at)
        );`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`CREATE TABLE policies (
            id varchar(255),
            rule_type ENUM ('header_client_tags', 'header_connection_properties', 'header_client_host', 'listening_port'),
            rule_value varchar(255),
            group_id varchar(255),
            fallback_group_id varchar(255),
            is_enabled bool,
            created_at int(11),
            updated_at int(11),
            PRIMARY KEY (id),
            KEY users_created_at_index (created_at),
            KEY users_updated_at_index (updated_at)
        );`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`CREATE TABLE queries (
            id varchar(255),
            text varchar(255),
            client_ip varchar(255),
            group_id varchar(255),
            backend_id varchar(255),
            username varchar(255),
            received_at int(11),
            submitted_at int(11),
            created_at int(11),
            updated_at int(11),
            PRIMARY KEY (id),
            KEY users_created_at_index (created_at),
            KEY users_updated_at_index (updated_at)
        );`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`ALTER TABLE group_backends_mappings ADD FOREIGN KEY (group_id) REFERENCES groups_ (id);`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`ALTER TABLE group_backends_mappings ADD FOREIGN KEY (backend_id) REFERENCES backends (id);`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`ALTER TABLE policies ADD FOREIGN KEY (group_id) REFERENCES groups_ (id);`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`ALTER TABLE policies ADD FOREIGN KEY (fallback_group_id) REFERENCES groups_ (id);`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`ALTER TABLE queries ADD FOREIGN KEY (group_id) REFERENCES groups_ (id);`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`ALTER TABLE queries ADD FOREIGN KEY (backend_id) REFERENCES backends (id);`)
	if err != nil {
		return err
	}
	return err
}

func Down20210805195304(tx *sql.Tx) error {
	var err error
	_, err = tx.Exec(`DROP TABLE IF EXISTS queries;`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DROP TABLE IF EXISTS policies;`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DROP TABLE IF EXISTS group_backends_mappings;`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DROP TABLE IF EXISTS backends;`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DROP TABLE IF EXISTS groups_;`)
	if err != nil {
		return err
	}
	return err
}
