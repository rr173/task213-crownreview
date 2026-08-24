package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// SchemaVersion is the current migration version.
const SchemaVersion = 1

// Open opens (or creates) the SQLite database at path and runs migrations.
// When path is empty an in-memory database is used.
func Open(path string) (*DB, error) {
	dsn := path
	if dsn == "" {
		dsn = ":memory:"
	} else {
		dsn = dsn + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(DELETE)&_pragma=foreign_keys(1)"
	}
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1) // SQLite single-writer; serialize to keep WAL stable
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return db, nil
}

// DB wraps the SQLite connection.
type DB struct {
	conn *sql.DB
}

// Conn exposes the underlying connection for transactions.
func (db *DB) Conn() *sql.DB { return db.conn }

// Close closes the database.
func (db *DB) Close() error { return db.conn.Close() }

func (db *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS scan_batches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			published_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS point_cloud_blocks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES scan_batches(id),
			tree_id TEXT NOT NULL,
			block_hash TEXT NOT NULL,
			coord_sys TEXT NOT NULL,
			status TEXT NOT NULL,
			point_count INTEGER NOT NULL DEFAULT 0,
			bbox_min_x REAL NOT NULL DEFAULT 0,
			bbox_min_y REAL NOT NULL DEFAULT 0,
			bbox_min_z REAL NOT NULL DEFAULT 0,
			bbox_max_x REAL NOT NULL DEFAULT 0,
			bbox_max_y REAL NOT NULL DEFAULT 0,
			bbox_max_z REAL NOT NULL DEFAULT 0,
			summary TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			parsed_at TEXT
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_block_batch_tree ON point_cloud_blocks(batch_id, tree_id)`,
		`CREATE INDEX IF NOT EXISTS idx_block_hash ON point_cloud_blocks(block_hash)`,
		`CREATE TABLE IF NOT EXISTS block_points (
			block_id INTEGER NOT NULL REFERENCES point_cloud_blocks(id),
			seq INTEGER NOT NULL,
			x REAL NOT NULL, y REAL NOT NULL, z REAL NOT NULL,
			intensity REAL NOT NULL DEFAULT 0,
			PRIMARY KEY (block_id, seq)
		)`,
		`CREATE TABLE IF NOT EXISTS skeleton_edges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			block_id INTEGER NOT NULL REFERENCES point_cloud_blocks(id),
			from_node TEXT NOT NULL,
			to_node TEXT NOT NULL,
			from_x REAL NOT NULL, from_y REAL NOT NULL, from_z REAL NOT NULL,
			to_x REAL NOT NULL, to_y REAL NOT NULL, to_z REAL NOT NULL,
			radius REAL NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_edge_block ON skeleton_edges(block_id)`,
		`CREATE TABLE IF NOT EXISTS break_candidates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			block_id INTEGER NOT NULL REFERENCES point_cloud_blocks(id),
			tree_id TEXT NOT NULL,
			status TEXT NOT NULL,
			pos_x REAL NOT NULL, pos_y REAL NOT NULL, pos_z REAL NOT NULL,
			severity REAL NOT NULL DEFAULT 0,
			confidence REAL NOT NULL DEFAULT 0,
			edge_a TEXT NOT NULL DEFAULT '',
			edge_b TEXT NOT NULL DEFAULT '',
			reason TEXT NOT NULL DEFAULT '',
			merged_into INTEGER,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cand_block ON break_candidates(block_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cand_status ON break_candidates(status)`,
		`CREATE TABLE IF NOT EXISTS occlusion_zones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			block_id INTEGER NOT NULL REFERENCES point_cloud_blocks(id),
			center_x REAL NOT NULL, center_y REAL NOT NULL, center_z REAL NOT NULL,
			radius REAL NOT NULL DEFAULT 0,
			score REAL NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_occ_block ON occlusion_zones(block_id)`,
		`CREATE TABLE IF NOT EXISTS review_opinions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			candidate_id INTEGER NOT NULL REFERENCES break_candidates(id),
			version_id INTEGER NOT NULL REFERENCES inspection_versions(id),
			reviewer TEXT NOT NULL,
			opinion TEXT NOT NULL,
			detail TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_opinion_cand ON review_opinions(candidate_id)`,
		`CREATE INDEX IF NOT EXISTS idx_opinion_ver ON review_opinions(version_id)`,
		`CREATE TABLE IF NOT EXISTS inspection_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES scan_batches(id),
			status TEXT NOT NULL,
			label TEXT NOT NULL DEFAULT '',
			snapshot TEXT NOT NULL DEFAULT '',
			canonical_hash TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			frozen_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ver_batch ON inspection_versions(batch_id)`,
	}
	for _, s := range stmts {
		if _, err := db.conn.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	if _, err := db.conn.Exec(
		`INSERT INTO schema_meta(key, value) VALUES('version', ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		fmt.Sprintf("%d", SchemaVersion),
	); err != nil {
		return fmt.Errorf("schema meta: %w", err)
	}
	return nil
}

// NowUTC returns current time in UTC.
func NowUTC() time.Time { return time.Now().UTC() }
