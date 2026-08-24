package store

import (
	"fmt"
	"time"

	"task213-crownreview/internal/model"
)

// AddOpinion records an expert opinion tied to a candidate and a version.
func (db *DB) AddOpinion(o *model.ReviewOpinion) (*model.ReviewOpinion, error) {
	now := NowUTC().Format(time.RFC3339)
	res, err := db.conn.Exec(
		`INSERT INTO review_opinions(candidate_id, version_id, reviewer, opinion, detail, created_at)
		 VALUES(?, ?, ?, ?, ?, ?)`,
		o.CandidateID, o.VersionID, o.Reviewer, o.Opinion, o.Detail, now)
	if err != nil {
		return nil, fmt.Errorf("insert opinion: %w", err)
	}
	id, _ := res.LastInsertId()
	row := db.conn.QueryRow(
		`SELECT id, candidate_id, version_id, reviewer, opinion, detail, created_at
		 FROM review_opinions WHERE id = ?`, id)
	out := &model.ReviewOpinion{}
	var created string
	if err := row.Scan(&out.ID, &out.CandidateID, &out.VersionID, &out.Reviewer, &out.Opinion, &out.Detail, &created); err != nil {
		return nil, fmt.Errorf("scan opinion: %w", err)
	}
	out.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return out, nil
}

// ListOpinions returns opinions for a candidate ordered by id.
func (db *DB) ListOpinions(candidateID int64) ([]*model.ReviewOpinion, error) {
	rows, err := db.conn.Query(
		`SELECT id, candidate_id, version_id, reviewer, opinion, detail, created_at
		 FROM review_opinions WHERE candidate_id = ? ORDER BY id`, candidateID)
	if err != nil {
		return nil, fmt.Errorf("query opinions: %w", err)
	}
	defer rows.Close()
	out := []*model.ReviewOpinion{}
	for rows.Next() {
		o := &model.ReviewOpinion{}
		var created string
		if err := rows.Scan(&o.ID, &o.CandidateID, &o.VersionID, &o.Reviewer, &o.Opinion, &o.Detail, &created); err != nil {
			return nil, fmt.Errorf("scan opinion: %w", err)
		}
		o.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, o)
	}
	return out, nil
}
