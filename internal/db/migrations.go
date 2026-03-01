package db

import (
	"database/sql"
	"fmt"
)

var migrations = []string{
	// v1: initial schema
	`CREATE TABLE IF NOT EXISTS cameras (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		onvif_address TEXT,
		rtsp_main TEXT NOT NULL,
		rtsp_sub TEXT,
		username TEXT,
		password TEXT,
		enabled INTEGER DEFAULT 1,
		sort_order INTEGER DEFAULT 0,
		created_at TEXT DEFAULT (datetime('now')),
		updated_at TEXT DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS groups (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		sort_order INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS camera_groups (
		camera_id TEXT REFERENCES cameras(id) ON DELETE CASCADE,
		group_id TEXT REFERENCES groups(id) ON DELETE CASCADE,
		PRIMARY KEY (camera_id, group_id)
	);

	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY
	);
	INSERT OR IGNORE INTO schema_version (version) VALUES (0);`,
}

func Migrate(db *sql.DB) error {
	// Ensure schema_version table exists
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY)`)
	if err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	var current int
	row := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version")
	if err := row.Scan(&current); err != nil {
		return fmt.Errorf("get schema version: %w", err)
	}

	for i := current; i < len(migrations); i++ {
		if _, err := db.Exec(migrations[i]); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := db.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", i+1); err != nil {
			return fmt.Errorf("update schema version: %w", err)
		}
	}

	return nil
}
