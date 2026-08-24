package store

import (
	"fmt"
	"time"

	"task213-crownreview/internal/model"
)

// SaveOcclusion replaces occlusion zones for a block and returns them.
func (db *DB) SaveOcclusion(blockID int64, zones []model.OcclusionZone) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM occlusion_zones WHERE block_id = ?`, blockID); err != nil {
		return fmt.Errorf("clear zones: %w", err)
	}
	stmt, err := tx.Prepare(
		`INSERT INTO occlusion_zones(block_id, center_x, center_y, center_z, radius, score, created_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare zone: %w", err)
	}
	defer stmt.Close()
	now := NowUTC().Format(time.RFC3339)
	for _, z := range zones {
		if _, err := stmt.Exec(blockID, z.Center[0], z.Center[1], z.Center[2], z.Radius, z.Score, now); err != nil {
			return fmt.Errorf("insert zone: %w", err)
		}
	}
	return tx.Commit()
}

// GetOcclusion loads all occlusion zones of a block.
func (db *DB) GetOcclusion(blockID int64) ([]model.OcclusionZone, error) {
	rows, err := db.conn.Query(
		`SELECT id, center_x, center_y, center_z, radius, score, created_at
		 FROM occlusion_zones WHERE block_id = ? ORDER BY id`, blockID)
	if err != nil {
		return nil, fmt.Errorf("query zones: %w", err)
	}
	defer rows.Close()
	out := []model.OcclusionZone{}
	for rows.Next() {
		z := model.OcclusionZone{BlockID: blockID}
		var created string
		if err := rows.Scan(&z.ID, &z.Center[0], &z.Center[1], &z.Center[2], &z.Radius, &z.Score, &created); err != nil {
			return nil, fmt.Errorf("scan zone: %w", err)
		}
		z.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, z)
	}
	return out, nil
}
