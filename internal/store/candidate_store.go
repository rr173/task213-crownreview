package store

import (
	"database/sql"
	"fmt"
	"time"

	"task213-crownreview/internal/model"
)

// CreateCandidate inserts a new break candidate in "open" status.
func (db *DB) CreateCandidate(c *model.BreakCandidate) (*model.BreakCandidate, error) {
	now := NowUTC().Format(time.RFC3339)
	res, err := db.conn.Exec(
		`INSERT INTO break_candidates
		 (block_id, tree_id, status, pos_x, pos_y, pos_z, severity, confidence, edge_a, edge_b, reason, merged_into, created_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.BlockID, c.TreeID, model.CandStatusOpen,
		c.Position[0], c.Position[1], c.Position[2], c.Severity, c.Confidence,
		c.EdgeA, c.EdgeB, c.Reason, nil, now)
	if err != nil {
		return nil, fmt.Errorf("insert candidate: %w", err)
	}
	id, _ := res.LastInsertId()
	return db.GetCandidate(id)
}

// ReplaceCandidates atomically replaces all break candidates (and any review
// opinions tied to them) for a block, then inserts the given candidates in
// "open" status. This restores parse idempotency: re-parsing a block no longer
// accumulates duplicate candidate rows. foreign_keys is ON, so opinions that
// reference the doomed candidates are cleared first to avoid FK violations.
func (db *DB) ReplaceCandidates(blockID int64, cands []model.BreakCandidate) ([]*model.BreakCandidate, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`DELETE FROM review_opinions
		 WHERE candidate_id IN (SELECT id FROM break_candidates WHERE block_id = ?)`,
		blockID); err != nil {
		return nil, fmt.Errorf("clear opinions: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM break_candidates WHERE block_id = ?`, blockID); err != nil {
		return nil, fmt.Errorf("clear candidates: %w", err)
	}
	stmt, err := tx.Prepare(
		`INSERT INTO break_candidates
		 (block_id, tree_id, status, pos_x, pos_y, pos_z, severity, confidence, edge_a, edge_b, reason, merged_into, created_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, fmt.Errorf("prepare candidate: %w", err)
	}
	defer stmt.Close()
	now := NowUTC().Format(time.RFC3339)
	for _, c := range cands {
		if _, err := stmt.Exec(blockID, c.TreeID, model.CandStatusOpen,
			c.Position[0], c.Position[1], c.Position[2], c.Severity, c.Confidence,
			c.EdgeA, c.EdgeB, c.Reason, nil, now); err != nil {
			return nil, fmt.Errorf("insert candidate: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return db.ListCandidates(blockID, "")
}

// GetCandidate fetches a candidate by id.
func (db *DB) GetCandidate(id int64) (*model.BreakCandidate, error) {
	row := db.conn.QueryRow(
		`SELECT id, block_id, tree_id, status, pos_x, pos_y, pos_z, severity, confidence,
		 edge_a, edge_b, reason, merged_into, created_at
		 FROM break_candidates WHERE id = ?`, id)
	c := &model.BreakCandidate{}
	var merged sql.NullInt64
	var created string
	if err := row.Scan(&c.ID, &c.BlockID, &c.TreeID, &c.Status,
		&c.Position[0], &c.Position[1], &c.Position[2], &c.Severity, &c.Confidence,
		&c.EdgeA, &c.EdgeB, &c.Reason, &merged, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan candidate: %w", err)
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, created)
	if merged.Valid {
		v := merged.Int64
		c.MergedInto = &v
	}
	return c, nil
}

// SetCandidateStatus transitions a candidate to a new status.
func (db *DB) SetCandidateStatus(id int64, status string, mergedInto *int64) error {
	_, err := db.conn.Exec(
		`UPDATE break_candidates SET status = ?, merged_into = ? WHERE id = ?`,
		status, mergedInto, id)
	if err != nil {
		return fmt.Errorf("update candidate status: %w", err)
	}
	return nil
}

// ListCandidates returns candidates filtered by block and/or status (0/none = all).
func (db *DB) ListCandidates(blockID int64, status string) ([]*model.BreakCandidate, error) {
	q := `SELECT id, block_id, tree_id, status, pos_x, pos_y, pos_z, severity, confidence,
		 edge_a, edge_b, reason, merged_into, created_at FROM break_candidates`
	args := []interface{}{}
	switch {
	case blockID > 0 && status != "":
		q += ` WHERE block_id = ? AND status = ? ORDER BY id`
		args = append(args, blockID, status)
	case blockID > 0:
		q += ` WHERE block_id = ? ORDER BY id`
		args = append(args, blockID)
	case status != "":
		q += ` WHERE status = ? ORDER BY id`
		args = append(args, status)
	default:
		q += ` ORDER BY id`
	}
	rows, err := db.conn.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list candidates: %w", err)
	}
	defer rows.Close()
	out := []*model.BreakCandidate{}
	for rows.Next() {
		c := &model.BreakCandidate{}
		var merged sql.NullInt64
		var created string
		if err := rows.Scan(&c.ID, &c.BlockID, &c.TreeID, &c.Status,
			&c.Position[0], &c.Position[1], &c.Position[2], &c.Severity, &c.Confidence,
			&c.EdgeA, &c.EdgeB, &c.Reason, &merged, &created); err != nil {
			return nil, fmt.Errorf("scan candidate row: %w", err)
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, created)
		if merged.Valid {
			v := merged.Int64
			c.MergedInto = &v
		}
		out = append(out, c)
	}
	return out, nil
}

// CandidatesForBatch returns all candidates belonging to blocks of a batch.
func (db *DB) CandidatesForBatch(batchID int64) ([]*model.BreakCandidate, error) {
	rows, err := db.conn.Query(
		`SELECT c.id, c.block_id, c.tree_id, c.status, c.pos_x, c.pos_y, c.pos_z,
		 c.severity, c.confidence, c.edge_a, c.edge_b, c.reason, c.merged_into, c.created_at
		 FROM break_candidates c JOIN point_cloud_blocks b ON c.block_id = b.id
		 WHERE b.batch_id = ? ORDER BY c.id`, batchID)
	if err != nil {
		return nil, fmt.Errorf("candidates for batch: %w", err)
	}
	defer rows.Close()
	out := []*model.BreakCandidate{}
	for rows.Next() {
		c := &model.BreakCandidate{}
		var merged sql.NullInt64
		var created string
		if err := rows.Scan(&c.ID, &c.BlockID, &c.TreeID, &c.Status,
			&c.Position[0], &c.Position[1], &c.Position[2], &c.Severity, &c.Confidence,
			&c.EdgeA, &c.EdgeB, &c.Reason, &merged, &created); err != nil {
			return nil, fmt.Errorf("scan cand: %w", err)
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, created)
		if merged.Valid {
			v := merged.Int64
			c.MergedInto = &v
		}
		out = append(out, c)
	}
	return out, nil
}
