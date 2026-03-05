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

	// v2: face recognition tables
	`CREATE TABLE IF NOT EXISTS face_subjects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		notes TEXT,
		alert_enabled INTEGER DEFAULT 1,
		created_at TEXT DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS face_embeddings (
		id TEXT PRIMARY KEY,
		subject_id TEXT REFERENCES face_subjects(id) ON DELETE CASCADE,
		embedding BLOB NOT NULL,
		crop_path TEXT,
		created_at TEXT DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS face_sightings (
		id TEXT PRIMARY KEY,
		subject_id TEXT REFERENCES face_subjects(id) ON DELETE CASCADE,
		camera_id TEXT REFERENCES cameras(id),
		confidence REAL NOT NULL,
		crop_path TEXT,
		seen_at TEXT DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS face_visitors (
		id TEXT PRIMARY KEY,
		cluster_id TEXT,
		camera_id TEXT REFERENCES cameras(id),
		embedding BLOB NOT NULL,
		crop_path TEXT,
		seen_at TEXT DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS face_clusters (
		id TEXT PRIMARY KEY,
		label TEXT,
		first_seen TEXT,
		last_seen TEXT,
		visit_count INTEGER DEFAULT 1,
		representative_crop TEXT
	);

	CREATE TABLE IF NOT EXISTS face_monitor_config (
		camera_id TEXT PRIMARY KEY REFERENCES cameras(id) ON DELETE CASCADE,
		monitor_type TEXT NOT NULL CHECK(monitor_type IN ('realtime','batch')),
		interval_seconds INTEGER DEFAULT 2
	);

	CREATE TABLE IF NOT EXISTS face_alerts (
		id TEXT PRIMARY KEY,
		sighting_id TEXT REFERENCES face_sightings(id),
		sent_at TEXT DEFAULT (datetime('now')),
		success INTEGER DEFAULT 1,
		error_message TEXT
	);`,

	// v3: add 'both' monitor type
	`CREATE TABLE face_monitor_config_new (
		camera_id TEXT PRIMARY KEY REFERENCES cameras(id) ON DELETE CASCADE,
		monitor_type TEXT NOT NULL CHECK(monitor_type IN ('realtime','batch','both')),
		interval_seconds INTEGER DEFAULT 2
	);
	INSERT INTO face_monitor_config_new SELECT * FROM face_monitor_config;
	DROP TABLE face_monitor_config;
	ALTER TABLE face_monitor_config_new RENAME TO face_monitor_config;`,
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
