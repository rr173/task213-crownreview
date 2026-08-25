package store

import (
	"database/sql"
	"fmt"
	"time"

	"task213-crownreview/internal/model"
)

// CreateVersion inserts a draft inspection version for a batch.
func (db *DB) CreateVersion(batchID int64, label, snapshot, canonicalHash string) (*model.InspectionVersion, error) {
	now := NowUTC().Format(time.RFC3339)
	res, err := db.conn.Exec(
		`INSERT INTO inspection_versions(batch_id, status, label, snapshot, canonical_hash, created_at, frozen_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?)`,
		batchID, model.VerStatusDraft, label, snapshot, canonicalHash, now, nil)
	if err != nil {
		return nil, fmt.Errorf("insert version: %w", err)
	}
	id, _ := res.LastInsertId()
	return db.GetVersion(id)
}

// GetVersion fetches a version by id.
func (db *DB) GetVersion(id int64) (*model.InspectionVersion, error) {
	row := db.conn.QueryRow(
		`SELECT id, batch_id, status, label, snapshot, canonical_hash, created_at, frozen_at
		 FROM inspection_versions WHERE id = ?`, id)
	v := &model.InspectionVersion{}
	var created string
	var frozen sql.NullString
	if err := row.Scan(&v.ID, &v.BatchID, &v.Status, &v.Label, &v.Snapshot, &v.CanonicalHash, &created, &frozen); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan version: %w", err)
	}
	v.CreatedAt, _ = time.Parse(time.RFC3339, created)
	if frozen.Valid {
		t, _ := time.Parse(time.RFC3339, frozen.String)
		v.FrozenAt = &t
	}
	return v, nil
}

// SetVersionStatus transitions a version; when frozen, records frozen_at.
func (db *DB) SetVersionStatus(id int64, status string) error {
	now := NowUTC().Format(time.RFC3339)
	var frozen interface{}
	if status == model.VerStatusFrozen {
		frozen = now
	}
	_, err := db.conn.Exec(
		`UPDATE inspection_versions SET status = ?, frozen_at = COALESCE(frozen_at, ?) WHERE id = ?`,
		status, frozen, id)
	if err != nil {
		return fmt.Errorf("update version status: %w", err)
	}
	return nil
}

// SetVersionCanonical overwrites the canonical hash (called at freeze time).
func (db *DB) SetVersionCanonical(id int64, hash string) error {
	_, err := db.conn.Exec(
		`UPDATE inspection_versions SET canonical_hash = ? WHERE id = ?`, hash, id)
	if err != nil {
		return fmt.Errorf("update version canonical: %w", err)
	}
	return nil
}

// ListVersions returns versions for a batch ordered by id.
func (db *DB) ListVersions(batchID int64) ([]*model.InspectionVersion, error) {
	rows, err := db.conn.Query(
		`SELECT id, batch_id, status, label, snapshot, canonical_hash, created_at, frozen_at
		 FROM inspection_versions WHERE batch_id = ? ORDER BY id`, batchID)
	if err != nil {
		return nil, fmt.Errorf("query versions: %w", err)
	}
	defer rows.Close()
	out := []*model.InspectionVersion{}
	for rows.Next() {
		v := &model.InspectionVersion{}
		var created string
		var frozen sql.NullString
		if err := rows.Scan(&v.ID, &v.BatchID, &v.Status, &v.Label, &v.Snapshot, &v.CanonicalHash, &created, &frozen); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339, created)
		if frozen.Valid {
			t, _ := time.Parse(time.RFC3339, frozen.String)
			v.FrozenAt = &t
		}
		out = append(out, v)
	}
	return out, nil
}

// FrozenVersionForBatch returns the latest frozen version for a batch (or nil).
// The query is scoped to batchID so that one batch can never borrow a frozen
// version belonging to a different batch.
func (db *DB) FrozenVersionForBatch(batchID int64) (*model.InspectionVersion, error) {
	row := db.conn.QueryRow(
		`SELECT id, batch_id, status, label, snapshot, canonical_hash, created_at, frozen_at
		 FROM inspection_versions WHERE batch_id = ? AND status = ?
		 ORDER BY id DESC LIMIT 1`,
		batchID, model.VerStatusFrozen)
	v := &model.InspectionVersion{}
	var created string
	var frozen sql.NullString
	if err := row.Scan(&v.ID, &v.BatchID, &v.Status, &v.Label, &v.Snapshot, &v.CanonicalHash, &created, &frozen); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan frozen version: %w", err)
	}
	v.CreatedAt, _ = time.Parse(time.RFC3339, created)
	if frozen.Valid {
		t, _ := time.Parse(time.RFC3339, frozen.String)
		v.FrozenAt = &t
	}
	return v, nil
}
