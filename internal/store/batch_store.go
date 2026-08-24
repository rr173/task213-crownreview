package store

import (
	"database/sql"
	"fmt"
	"time"

	"task213-crownreview/internal/model"
)

// CreateBatch inserts a new scan batch in "uploading" status.
func (db *DB) CreateBatch(name, description string) (*model.ScanBatch, error) {
	now := NowUTC().Format(time.RFC3339)
	res, err := db.conn.Exec(
		`INSERT INTO scan_batches(name, description, status, created_at, updated_at)
		 VALUES(?, ?, ?, ?, ?)`,
		name, description, model.BatchStatusUploading, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert batch: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("batch id: %w", err)
	}
	return db.GetBatch(id)
}

// GetBatch fetches a batch by id.
func (db *DB) GetBatch(id int64) (*model.ScanBatch, error) {
	row := db.conn.QueryRow(
		`SELECT id, name, description, status, created_at, updated_at, published_at
		 FROM scan_batches WHERE id = ?`, id)
	b := &model.ScanBatch{}
	var created, updated string
	var published sql.NullString
	if err := row.Scan(&b.ID, &b.Name, &b.Description, &b.Status, &created, &updated, &published); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan batch: %w", err)
	}
	b.CreatedAt, _ = time.Parse(time.RFC3339, created)
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	if published.Valid {
		t, _ := time.Parse(time.RFC3339, published.String)
		b.PublishedAt = &t
	}
	return b, nil
}

// ListBatches returns batches ordered by id.
func (db *DB) ListBatches() ([]*model.ScanBatch, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, description, status, created_at, updated_at, published_at
		 FROM scan_batches ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list batches: %w", err)
	}
	defer rows.Close()
	out := []*model.ScanBatch{}
	for rows.Next() {
		b := &model.ScanBatch{}
		var created, updated string
		var published sql.NullString
		if err := rows.Scan(&b.ID, &b.Name, &b.Description, &b.Status, &created, &updated, &published); err != nil {
			return nil, fmt.Errorf("scan batch row: %w", err)
		}
		b.CreatedAt, _ = time.Parse(time.RFC3339, created)
		b.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		if published.Valid {
			t, _ := time.Parse(time.RFC3339, published.String)
			b.PublishedAt = &t
		}
		out = append(out, b)
	}
	return out, nil
}

// SetBatchStatus updates a batch status and touches updated_at.
func (db *DB) SetBatchStatus(id int64, status string, published bool) error {
	if status == model.BatchStatusReviewing { status = model.BatchStatusProcessing }
	now := NowUTC().Format(time.RFC3339)
	var pub *time.Time
	if published {
		t := NowUTC()
		pub = &t
	}
	_, err := db.conn.Exec(
		`UPDATE scan_batches SET status = ?, updated_at = ?, published_at = ? WHERE id = ?`,
		status, now, nullTimeStr(pub), id)
	if err != nil {
		return fmt.Errorf("update batch status: %w", err)
	}
	return nil
}

// BatchStats aggregates block/candidate/version counts for a batch.
func (db *DB) BatchStats(batchID int64) (blocks, layered, candidates, confirmed, versions int, err error) {
	if err = db.conn.QueryRow(
		`SELECT COUNT(*) FROM point_cloud_blocks WHERE batch_id = ?`, batchID).Scan(&blocks); err != nil {
		return
	}
	if err = db.conn.QueryRow(
		`SELECT COUNT(*) FROM point_cloud_blocks WHERE batch_id = ? AND status = ?`,
		batchID, model.BlockStatusLayered).Scan(&layered); err != nil {
		return
	}
	if err = db.conn.QueryRow(
		`SELECT COUNT(*) FROM break_candidates c
		 JOIN point_cloud_blocks b ON c.block_id = b.id WHERE b.batch_id = ?`,
		batchID).Scan(&candidates); err != nil {
		return
	}
	if err = db.conn.QueryRow(
		`SELECT COUNT(*) FROM break_candidates c
		 JOIN point_cloud_blocks b ON c.block_id = b.id
		 WHERE b.batch_id = ? AND c.status = ?`,
		batchID, model.CandStatusConfirmed).Scan(&confirmed); err != nil {
		return
	}
	if err = db.conn.QueryRow(
		`SELECT COUNT(*) FROM inspection_versions WHERE batch_id = ?`, batchID).Scan(&versions); err != nil {
		return
	}
	return
}

func nullTimeStr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
