package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"

	"task213-crownreview/internal/httpapi"
	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "", "SQLite database path (empty = in-memory)")
	smoke := flag.Bool("smoke-test", false, "run self-contained smoke test and exit 0 on success")
	flag.Parse()

	if *smoke {
		if err := runSmokeTest(); err != nil {
			fmt.Fprintln(os.Stderr, "SMOKE-TEST FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("SMOKE-TEST OK")
		os.Exit(0)
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	svc := service.New(db)
	srv := httpapi.NewServer(svc, db, *addr, *dbPath)
	fmt.Printf("task213-crownreview listening on %s\n", *addr)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// makeBranchPoints generates a vertical branch as a column of points with a gap,
// optionally leaving a break gap near the top.
func makeBranchPoints(baseX, baseY, baseZ, height float64, gap bool) []model.Point3 {
	pts := []model.Point3{}
	step := 0.04
	n := int(height / step)
	for i := 0; i < n; i++ {
		z := baseZ + float64(i)*step
		if gap && z > baseZ+height*0.6 && z < baseZ+height*0.6+0.3 {
			continue // simulated break: missing points in a ~0.3m gap
		}
		pts = append(pts, model.Point3{
			X:         baseX + 0.01*math.Sin(float64(i)),
			Y:         baseY + 0.01*math.Cos(float64(i)),
			Z:         z,
			Intensity: 0.8,
		})
	}
	return pts
}

// runSmokeTest exercises the full closed loop against a temp SQLite file,
// then closes and reopens to verify persistence and restart recovery.
func runSmokeTest() error {
	tmp, err := os.MkdirTemp("", "crownreview-smoke-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	dbFile := filepath.Join(tmp, "smoke.db")

	phase1 := func() (batchID, blkID, candReal, candFalse, verID int64, e error) {
		db, err := store.Open(dbFile)
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}
		defer db.Close()
		svc := service.New(db)

		b, err := svc.CreateBatch("smoke-batch", "self-check survey")
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}
		// A tree crown with a real break (gap) and a stub that reconnects above the gap.
		real := append(makeBranchPoints(0, 0, 0, 2.0, true),
			makeBranchPoints(0.05, 0, 2.0, 0.4, false)...)
		blk, err := svc.UploadBlock(b.ID, &pointcloud.BlockInput{
			TreeID: "T1", CoordSys: "local-tree", Points: real})
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}
		blk, err = svc.ParseBlock(blk.ID)
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}
		cands, err := svc.ListCandidates(blk.ID, "")
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}
		if len(cands) == 0 {
			return 0, 0, 0, 0, 0, fmt.Errorf("no candidates generated")
		}
		// Resolve every candidate: confirm the highest-confidence one (the real
		// break) and reject all others so the version can be frozen.
		var realCand *model.BreakCandidate
		for i := range cands {
			if realCand == nil || cands[i].Confidence > realCand.Confidence {
				realCand = cands[i]
			}
		}
		if realCand == nil {
			return 0, 0, 0, 0, 0, fmt.Errorf("real candidate missing")
		}
		for i := range cands {
			if cands[i].ID == realCand.ID {
				if _, err := svc.ConfirmCandidate(cands[i].ID); err != nil {
					return 0, 0, 0, 0, 0, err
				}
			} else {
				if _, err := svc.RejectCandidate(cands[i].ID); err != nil {
					return 0, 0, 0, 0, 0, err
				}
			}
		}
		v, err := svc.CreateVersion(b.ID)
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}
		v, err = svc.FreezeVersion(v.ID)
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}
		if v.Status != model.VerStatusFrozen {
			return 0, 0, 0, 0, 0, fmt.Errorf("version not frozen: %s", v.Status)
		}
		pb, err := svc.PublishBatch(b.ID)
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}
		if pb.Status != model.BatchStatusPublished {
			return 0, 0, 0, 0, 0, fmt.Errorf("batch not published: %s", pb.Status)
		}
		return b.ID, blk.ID, realCand.ID, 0, v.ID, nil
	}

	bID, blkID, candReal, candFalse, verID, err := phase1()
	if err != nil {
		return fmt.Errorf("phase1: %w", err)
	}

	// Second session: reopen DB, verify recovery and immutability.
	db2, err := store.Open(dbFile)
	if err != nil {
		return err
	}
	defer db2.Close()
	svc2 := service.New(db2)

	batch, err := svc2.GetBatch(bID)
	if err != nil {
		return fmt.Errorf("reopen batch: %w", err)
	}
	if batch.Status != model.BatchStatusPublished {
		return fmt.Errorf("restart recovery: batch status %s", batch.Status)
	}
	blk, err := svc2.GetBlock(blkID)
	if err != nil {
		return fmt.Errorf("reopen block: %w", err)
	}
	if blk.Status != model.BlockStatusLayered {
		return fmt.Errorf("restart recovery: block status %s", blk.Status)
	}
	rc, err := svc2.GetCandidate(candReal)
	if err != nil {
		return fmt.Errorf("reopen candidate: %w", err)
	}
	if rc.Status != model.CandStatusConfirmed {
		return fmt.Errorf("restart recovery: candidate status %s", rc.Status)
	}
	fv, err := svc2.FrozenVersionForBatch(bID)
	if err != nil {
		return fmt.Errorf("reopen frozen version: %w", err)
	}
	if fv == nil || fv.ID != verID {
		return fmt.Errorf("restart recovery: frozen version mismatch")
	}
	if fv.Status != model.VerStatusFrozen {
		return fmt.Errorf("restart recovery: version not frozen %s", fv.Status)
	}
	// idempotent re-upload of same block hash must be a duplicate, not an error
	if _, err := svc2.UploadBlock(bID, &pointcloud.BlockInput{
		TreeID:   "T1",
		CoordSys: "local-tree",
		Points:   append(makeBranchPoints(0, 0, 0, 2.0, true), makeBranchPoints(0.05, 0, 2.0, 0.4, false)...),
	}); err != nil {
		return fmt.Errorf("idempotent re-upload: %w", err)
	}
	_ = candFalse
	return nil
}
