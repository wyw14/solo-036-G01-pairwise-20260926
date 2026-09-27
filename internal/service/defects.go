package service

import (
	"fmt"
	"time"

	"github.com/wyw14/solo-036/internal/defect"
	"github.com/wyw14/solo-036/internal/model"
)

type DefectReadingRequest struct {
	Kind       defect.Kind   `json:"kind"`
	Source     defect.Source `json:"source"`
	Kilometer  float64       `json:"kilometer"`
	Value      float64       `json:"value"`
	Unit       string        `json:"unit"`
	ObservedAt time.Time     `json:"observed_at"`
}

type DefectAssessRequest struct {
	Status     defect.Status   `json:"status"`
	Severity   defect.Severity `json:"severity"`
	Note       string          `json:"note"`
	AssessedBy string          `json:"assessed_by"`
}

// RecordDefect stores a raw inspection reading and merges it into the
// defect cluster at the same location.
func (r *Runtime) RecordDefect(runID string, request DefectReadingRequest) (defect.Defect, error) {
	if _, ok := r.Runs.Get(runID); !ok {
		return defect.Defect{}, fmt.Errorf("run %s not found", runID)
	}
	if request.ObservedAt.IsZero() {
		request.ObservedAt = time.Now().UTC()
	}
	now := time.Now().UTC()
	merged, err := r.DefectLog.AddReading(defect.Reading{
		ID:         model.NewRevision(),
		RunID:      runID,
		Kind:       request.Kind,
		Source:     request.Source,
		Kilometer:  request.Kilometer,
		Value:      request.Value,
		Unit:       request.Unit,
		ObservedAt: request.ObservedAt,
	}, now)
	if err != nil {
		return defect.Defect{}, err
	}
	if err := r.appendEvent(model.Event{
		RunID:   runID,
		Kind:    "defect",
		Payload: fmt.Sprintf("%s/%s/%.4f/%d", merged.ID, merged.Kind, merged.Kilometer, len(merged.Readings)),
		At:      now,
	}); err != nil {
		return defect.Defect{}, err
	}
	return merged, nil
}

// AssessDefect appends a new judgment version to a defect.
func (r *Runtime) AssessDefect(runID, defectID string, request DefectAssessRequest) (defect.Defect, error) {
	if _, ok := r.Runs.Get(runID); !ok {
		return defect.Defect{}, fmt.Errorf("run %s not found", runID)
	}
	assessedBy := request.AssessedBy
	if assessedBy == "" {
		assessedBy = "operator"
	}
	now := time.Now().UTC()
	assessed, err := r.DefectLog.Assess(runID, defectID, request.Status, request.Severity, request.Note, assessedBy, now)
	if err != nil {
		return defect.Defect{}, err
	}
	latest, _ := assessed.LatestAssessment()
	if err := r.appendEvent(model.Event{
		RunID:   runID,
		Kind:    "defect_assessment",
		Payload: fmt.Sprintf("%s/v%d/%s/%s", defectID, latest.Version, latest.Status, latest.Severity),
		At:      now,
	}); err != nil {
		return defect.Defect{}, err
	}
	return assessed, nil
}

func (r *Runtime) Defects(runID string) ([]defect.Defect, error) {
	if _, ok := r.Runs.Get(runID); !ok {
		return nil, fmt.Errorf("run %s not found", runID)
	}
	return r.DefectLog.List(runID), nil
}

func (r *Runtime) DefectDistribution(runID string, bucketKm float64) (defect.Distribution, error) {
	if _, ok := r.Runs.Get(runID); !ok {
		return defect.Distribution{}, fmt.Errorf("run %s not found", runID)
	}
	return defect.Distribute(runID, r.DefectLog.List(runID), bucketKm), nil
}

func (r *Runtime) PendingDefects(runID string) ([]defect.Defect, error) {
	if _, ok := r.Runs.Get(runID); !ok {
		return nil, fmt.Errorf("run %s not found", runID)
	}
	return r.DefectLog.PendingReview(runID), nil
}
