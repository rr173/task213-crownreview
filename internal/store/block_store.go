package store

import (
	"database/sql"
	"fmt"
	"time"

	"task213-crownreview/internal/model"
)

// SaveBlockPoints persists the raw points of a block (used for reprocessing after restart).
func (db *DB) SaveBlockPoints(blockID int64, points []model.Point3) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM block_points WHERE block_id = ?`, blockID); err != nil {
		return fmt.Errorf("clear points: %w", err)
	}
	stmt, err := tx.Prepare(
		`INSERT INTO block_points(block_id, seq, x, y, z, intensity) VALUES(?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare points: %w", err)
	}
	defer stmt.Close()
	for i, p := range points {
		if _, err := stmt.Exec(blockID, i, p.X, p.Y, p.Z, p.Intensity); err != nil {
			return fmt.Errorf("insert point: %w", err)
		}
	}
	return tx.Commit()
}

// GetBlockPoints loads all points of a block ordered by seq.
func (db *DB) GetBlockPoints(blockID int64) ([]model.Point3, error) {
	rows, err := db.conn.Query(
		`SELECT x, y, z, intensity FROM block_points WHERE block_id = ? ORDER BY seq`, blockID)
	if err != nil {
		return nil, fmt.Errorf("query points: %w", err)
	}
	defer rows.Close()
	out := []model.Point3{}
	for rows.Next() {
		var x, y, z, it float64
		if err := rows.Scan(&x, &y, &z, &it); err != nil {
			return nil, fmt.Errorf("scan point: %w", err)
		}
		out = append(out, model.Point3{X: x, Y: y, Z: z, Intensity: it})
	}
	return out, nil
}

// CreateBlock inserts a pending block. treeID must be unique within a batch among
// non-duplicate blocks; caller passes isDuplicate when hash already seen.
func (db *DB) CreateBlock(b *model.PointCloudBlock, points []model.Point3, isDuplicate bool) (*model.PointCloudBlock, error) {
	now := NowUTC().Format(time.RFC3339)
	status := model.BlockStatusPending
	if isDuplicate {
		status = model.BlockStatusDuplicate
	}
	res, err := db.conn.Exec(
		`INSERT INTO point_cloud_blocks
		 (batch_id, tree_id, block_hash, coord_sys, status, point_count,
		  bbox_min_x, bbox_min_y, bbox_min_z, bbox_max_x, bbox_max_y, bbox_max_z,
		  summary, created_at, parsed_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.BatchID, b.TreeID, b.BlockHash, b.CoordSys, status, b.PointCount,
		b.BBoxMin[0], b.BBoxMin[1], b.BBoxMin[2], b.BBoxMax[0], b.BBoxMax[1], b.BBoxMax[2],
		b.Summary, now, nil)
	if err != nil {
		return nil, fmt.Errorf("insert block: %w", err)
	}
	id, _ := res.LastInsertId()
	if err := db.SaveBlockPoints(id, points); err != nil {
		return nil, err
	}
	return db.GetBlock(id)
}

// GetBlock fetches a block by id.
func (db *DB) GetBlock(id int64) (*model.PointCloudBlock, error) {
	row := db.conn.QueryRow(
		`SELECT id, batch_id, tree_id, block_hash, coord_sys, status, point_count,
		 bbox_min_x, bbox_min_y, bbox_min_z, bbox_max_x, bbox_max_y, bbox_max_z,
		 summary, created_at, parsed_at FROM point_cloud_blocks WHERE id = ?`, id)
	b := &model.PointCloudBlock{}
	var cminx, cminy, cminz, cmaxx, cmaxy, cmaxz float64
	var created string
	var parsed sql.NullString
	if err := row.Scan(&b.ID, &b.BatchID, &b.TreeID, &b.BlockHash, &b.CoordSys, &b.Status,
		&b.PointCount, &cminx, &cminy, &cminz, &cmaxx, &cmaxy, &cmaxz,
		&b.Summary, &created, &parsed); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan block: %w", err)
	}
	b.BBoxMin = [3]float64{cminx, cminy, cminz}
	b.BBoxMax = [3]float64{cmaxx, cmaxy, cmaxz}
	b.CreatedAt, _ = time.Parse(time.RFC3339, created)
	if parsed.Valid {
		t, _ := time.Parse(time.RFC3339, parsed.String)
		b.ParsedAt = &t
	}
	return b, nil
}

// BlockExistsByTree reports whether a non-duplicate block already owns the tree id in the batch.
func (db *DB) BlockExistsByTree(batchID int64, treeID string) (*model.PointCloudBlock, bool, error) {
	row := db.conn.QueryRow(
		`SELECT id FROM point_cloud_blocks WHERE batch_id = ? AND tree_id = ? AND status != ?
		 ORDER BY id LIMIT 1`, batchID, treeID, model.BlockStatusDuplicate)
	var id int64
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("scan tree block: %w", err)
	}
	b, err := db.GetBlock(id)
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// BlockExistsByHash reports whether a block with the given hash exists.
func (db *DB) BlockExistsByHash(hash string) (bool, error) {
	var n int
	if err := db.conn.QueryRow(
		`SELECT COUNT(*) FROM point_cloud_blocks WHERE block_hash = ?`, hash).Scan(&n); err != nil {
		return false, fmt.Errorf("count hash: %w", err)
	}
	return n > 0, nil
}

// BlockByHash returns the first block with the given hash (idempotent lookup).
func (db *DB) BlockByHash(hash string) (*model.PointCloudBlock, error) {
	row := db.conn.QueryRow(
		`SELECT id, batch_id, tree_id, block_hash, coord_sys, status, point_count,
		 bbox_min_x, bbox_min_y, bbox_min_z, bbox_max_x, bbox_max_y, bbox_max_z,
		 summary, created_at, parsed_at FROM point_cloud_blocks WHERE block_hash = ? LIMIT 1`, hash)
	b := &model.PointCloudBlock{}
	var cminx, cminy, cminz, cmaxx, cmaxy, cmaxz float64
	var created string
	var parsed sql.NullString
	if err := row.Scan(&b.ID, &b.BatchID, &b.TreeID, &b.BlockHash, &b.CoordSys, &b.Status,
		&b.PointCount, &cminx, &cminy, &cminz, &cmaxx, &cmaxy, &cmaxz,
		&b.Summary, &created, &parsed); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan block by hash: %w", err)
	}
	b.BBoxMin = [3]float64{cminx, cminy, cminz}
	b.BBoxMax = [3]float64{cmaxx, cmaxy, cmaxz}
	b.CreatedAt, _ = time.Parse(time.RFC3339, created)
	if parsed.Valid {
		t, _ := time.Parse(time.RFC3339, parsed.String)
		b.ParsedAt = &t
	}
	return b, nil
}

// ListBlocks returns blocks for a batch (or all when batchID<=0).
func (db *DB) ListBlocks(batchID int64) ([]*model.PointCloudBlock, error) {
	q := `SELECT id, batch_id, tree_id, block_hash, coord_sys, status, point_count,
		 bbox_min_x, bbox_min_y, bbox_min_z, bbox_max_x, bbox_max_y, bbox_max_z,
		 summary, created_at, parsed_at FROM point_cloud_blocks`
	var rows *sql.Rows
	var err error
	if batchID > 0 {
		rows, err = db.conn.Query(q+` WHERE batch_id = ? ORDER BY id`, batchID)
	} else {
		rows, err = db.conn.Query(q + ` ORDER BY id`)
	}
	if err != nil {
		return nil, fmt.Errorf("list blocks: %w", err)
	}
	defer rows.Close()
	out := []*model.PointCloudBlock{}
	for rows.Next() {
		b := &model.PointCloudBlock{}
		var cminx, cminy, cminz, cmaxx, cmaxy, cmaxz float64
		var created string
		var parsed sql.NullString
		if err := rows.Scan(&b.ID, &b.BatchID, &b.TreeID, &b.BlockHash, &b.CoordSys, &b.Status,
			&b.PointCount, &cminx, &cminy, &cminz, &cmaxx, &cmaxy, &cmaxz,
			&b.Summary, &created, &parsed); err != nil {
			return nil, fmt.Errorf("scan block row: %w", err)
		}
		b.BBoxMin = [3]float64{cminx, cminy, cminz}
		b.BBoxMax = [3]float64{cmaxx, cmaxy, cmaxz}
		b.CreatedAt, _ = time.Parse(time.RFC3339, created)
		if parsed.Valid {
			t, _ := time.Parse(time.RFC3339, parsed.String)
			b.ParsedAt = &t
		}
		out = append(out, b)
	}
	return out, nil
}

// SetBlockStatus marks a block as layered/missing and records parsed_at.
func (db *DB) SetBlockStatus(id int64, status string) error {
	now := NowUTC().Format(time.RFC3339)
	var parsed interface{}
	if status == model.BlockStatusLayered {
		parsed = now
	}
	_, err := db.conn.Exec(
		`UPDATE point_cloud_blocks SET status = ?, parsed_at = ? WHERE id = ?`,
		status, parsed, id)
	if err != nil {
		return fmt.Errorf("update block status: %w", err)
	}
	return nil
}
