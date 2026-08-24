package store

import (
	"fmt"
	"time"

	"task213-crownreview/internal/model"
)

// SaveSkeleton replaces all skeleton edges for a block and returns them.
func (db *DB) SaveSkeleton(blockID int64, edges []model.SkeletonEdge) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(
		`INSERT INTO skeleton_edges
		 (block_id, from_node, to_node, from_x, from_y, from_z, to_x, to_y, to_z, radius, created_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare edge: %w", err)
	}
	defer stmt.Close()
	now := NowUTC().Format(time.RFC3339)
	for _, e := range edges {
		if _, err := stmt.Exec(blockID, e.FromNode, e.ToNode,
			e.FromPoint[0], e.FromPoint[1], e.FromPoint[2],
			e.ToPoint[0], e.ToPoint[1], e.ToPoint[2], e.Radius, now); err != nil {
			return fmt.Errorf("insert edge: %w", err)
		}
	}
	return tx.Commit()
}

// GetSkeleton loads all skeleton edges of a block.
func (db *DB) GetSkeleton(blockID int64) ([]model.SkeletonEdge, error) {
	rows, err := db.conn.Query(
		`SELECT id, from_node, to_node, from_x, from_y, from_z, to_x, to_y, to_z, radius, created_at
		 FROM skeleton_edges WHERE block_id = ? ORDER BY id`, blockID)
	if err != nil {
		return nil, fmt.Errorf("query edges: %w", err)
	}
	defer rows.Close()
	out := []model.SkeletonEdge{}
	for rows.Next() {
		e := model.SkeletonEdge{BlockID: blockID}
		var created string
		if err := rows.Scan(&e.ID, &e.FromNode, &e.ToNode,
			&e.FromPoint[0], &e.FromPoint[1], &e.FromPoint[2],
			&e.ToPoint[0], &e.ToPoint[1], &e.ToPoint[2], &e.Radius, &created); err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, e)
	}
	return out, nil
}

// SkeletonCount returns the number of skeleton edges for a block.
func (db *DB) SkeletonCount(blockID int64) (int, error) {
	var n int
	if err := db.conn.QueryRow(
		`SELECT COUNT(*) FROM skeleton_edges WHERE block_id = ?`, blockID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count edges: %w", err)
	}
	return n, nil
}
