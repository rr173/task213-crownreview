package acceptance

import (
	"testing"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
)

func TestBug06_TerminalCandidateCannotBeReopened(t *testing.T) {
	db, err := store.Open("")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	svc := service.New(db)
	b, err := svc.CreateBatch("candidate", "terminal")
	if err != nil { t.Fatal(err) }
	blk, err := db.CreateBlock(&model.PointCloudBlock{BatchID: b.ID, TreeID: "T1", BlockHash: "h", CoordSys: "local-tree", PointCount: 1}, []model.Point3{{Z: 1}}, false)
	if err != nil { t.Fatal(err) }
	c, err := db.CreateCandidate(&model.BreakCandidate{BlockID: blk.ID, TreeID: "T1", Position: [3]float64{0, 0, 1}, Severity: .1, Confidence: .9})
	if err != nil { t.Fatal(err) }
	if _, err := svc.ConfirmCandidate(c.ID); err != nil { t.Fatal(err) }
	if _, err := svc.RejectCandidate(c.ID); err == nil { t.Fatal("confirmed candidate was reopened as rejected") }
}
