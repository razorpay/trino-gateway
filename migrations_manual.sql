-- ============================================================================
-- Trino Gateway — manual DB migration script
-- Generated from internal/gatewayserver/database/migrations/*.go
-- Migrations applied in chronological order:
--   20210805195304_bootstrap
--   20211203205304_alter_backends
--   20220107205304_increase_query_text_size
--   20240524205304_add_auth_delegation
--   20240525205304_add_set_source
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 20210805195304_bootstrap (UP)
-- ----------------------------------------------------------------------------
CREATE TABLE backends (
    id VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    scheme ENUM('http', 'https') DEFAULT 'http',
    external_url VARCHAR(255) NOT NULL,
    is_enabled BOOL DEFAULT FALSE,
    uptime_schedule VARCHAR(255) DEFAULT '* * * * *',
    cluster_load INT DEFAULT 0,
    threshold_cluster_load INT DEFAULT 0,
    stats_updated_at INT(11) NULL,
    created_at INT(11) NOT NULL,
    updated_at INT(11) NOT NULL,
    PRIMARY KEY (id),
    KEY users_created_at_index (created_at),
    KEY users_updated_at_index (updated_at)
);

-- groups is a reserved keyword in MySQL, so the table is named groups_
CREATE TABLE `groups_` (
    id VARCHAR(255) NOT NULL,
    strategy ENUM('random', 'round_robin', 'least_load') DEFAULT 'random',
    is_enabled BOOL DEFAULT FALSE,
    last_routed_backend VARCHAR(255),
    created_at INT(11) NOT NULL,
    updated_at INT(11) NOT NULL,
    PRIMARY KEY (id),
    KEY users_created_at_index (created_at),
    KEY users_updated_at_index (updated_at)
);

CREATE TABLE group_backends_mappings (
    id INT AUTO_INCREMENT,
    group_id VARCHAR(255),
    backend_id VARCHAR(255),
    created_at INT(11),
    updated_at INT(11),
    PRIMARY KEY (id),
    UNIQUE KEY (group_id, backend_id),
    KEY users_created_at_index (created_at),
    KEY users_updated_at_index (updated_at)
);

CREATE TABLE policies (
    id VARCHAR(255),
    rule_type ENUM('header_client_tags', 'header_connection_properties', 'header_client_host', 'listening_port'),
    rule_value VARCHAR(255),
    group_id VARCHAR(255),
    fallback_group_id VARCHAR(255),
    is_enabled BOOL,
    created_at INT(11),
    updated_at INT(11),
    PRIMARY KEY (id),
    KEY users_created_at_index (created_at),
    KEY users_updated_at_index (updated_at)
);

CREATE TABLE queries (
    id VARCHAR(255),
    text VARCHAR(255),
    client_ip VARCHAR(255),
    group_id VARCHAR(255) NULL,
    backend_id VARCHAR(255) NULL,
    username VARCHAR(255),
    server_host VARCHAR(255),
    submitted_at INT(11),
    created_at INT(11),
    updated_at INT(11),
    PRIMARY KEY (id),
    KEY users_created_at_index (created_at),
    KEY users_updated_at_index (updated_at)
);

ALTER TABLE group_backends_mappings ADD FOREIGN KEY (group_id) REFERENCES groups_ (id);
ALTER TABLE group_backends_mappings ADD FOREIGN KEY (backend_id) REFERENCES backends (id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE policies ADD FOREIGN KEY (group_id) REFERENCES groups_ (id);
ALTER TABLE policies ADD FOREIGN KEY (fallback_group_id) REFERENCES groups_ (id);
ALTER TABLE queries ADD FOREIGN KEY (group_id) REFERENCES groups_ (id);
ALTER TABLE queries ADD FOREIGN KEY (backend_id) REFERENCES backends (id);

-- ----------------------------------------------------------------------------
-- 20211203205304_alter_backends (UP)
-- ----------------------------------------------------------------------------
ALTER TABLE `backends` ADD COLUMN `is_healthy` BOOL DEFAULT FALSE;

-- ----------------------------------------------------------------------------
-- 20220107205304_increase_query_text_size (UP)
-- ----------------------------------------------------------------------------
ALTER TABLE `queries` MODIFY COLUMN `text` VARCHAR(500);

-- ----------------------------------------------------------------------------
-- 20240524205304_add_auth_delegation (UP)
-- ----------------------------------------------------------------------------
ALTER TABLE `policies` ADD COLUMN `is_auth_delegated` BOOL DEFAULT FALSE;

-- ----------------------------------------------------------------------------
-- 20240525205304_add_set_source (UP)
-- ----------------------------------------------------------------------------
ALTER TABLE `policies` ADD COLUMN `set_request_source` VARCHAR(255) DEFAULT '';

-- ============================================================================
-- ROLLBACK (DOWN migrations) — run manually only if you need to undo everything
-- ============================================================================
-- ALTER TABLE `policies` DROP COLUMN `set_request_source`;
-- ALTER TABLE `policies` DROP COLUMN `is_auth_delegated`;
-- ALTER TABLE `queries` MODIFY COLUMN `text` VARCHAR(255);
-- ALTER TABLE `backends` DROP COLUMN `is_healthy`;
-- DROP TABLE IF EXISTS queries;
-- DROP TABLE IF EXISTS policies;
-- DROP TABLE IF EXISTS group_backends_mappings;
-- DROP TABLE IF EXISTS backends;
-- DROP TABLE IF EXISTS groups_;
