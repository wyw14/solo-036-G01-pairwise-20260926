package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/wyw14/solo-036/internal/model"
)

type DefectReadingRequest struct {
	SourceID     string            `json:"source_id"`
	Probe        model.ProbeKind   `json:"probe"`
	Kind         model.AnomalyKind `json:"kind"`
	Kilometer    float64           `json:"kilometer"`
	DepthPercent float64           `json:"depth_percent"`
	LengthMM     float64           `json:"length_mm"`
	WidthMM      float64           `json:"width_mm"`
	RecordedAt   time.Time         `json:"recorded_at"`
}

type DefectAssessmentRequest struct {
	Severity model.Severity     `json:"severity"`
	Status   model.ReviewStatus `json:"status"`
	Reviewer string             `json:"reviewer"`
	Note     string             `json:"note"`
}

type DefectRecordResult struct {
	Merged   bool               `json:"merged"`
	Reopened bool               `json:"reopened"`
	Defect   model.DefectRecord `json:"defect"`
}

type DefectBucket struct {
	SegmentID string         `json:"segment_id"`
	FromKm    float64        `json:"from_km"`
	ToKm      float64        `json:"to_km"`
	Count     int            `json:"count"`
	Pending   int            `json:"pending"`
	ByKind    map[string]int `json:"by_kind"`
}

type DefectSummary struct {
	Total         int            `json:"total"`
	PendingReview int            `json:"pending_review"`
	ByKind        map[string]int `json:"by_kind"`
	BySeverity    map[string]int `json:"by_severity"`
}

type DefectDistribution struct {
	Defects []model.DefectRecord `json:"defects"`
	Buckets []DefectBucket       `json:"buckets"`
	Summary DefectSummary        `json:"summary"`
}

func (r *Runtime) maxLineKilometer() float64 {
	maximum := 0.0
	for _, station := range r.ensureControl().Network.Stations() {
		if station.Kilometer > maximum {
			maximum = station.Kilometer
		}
	}
	return maximum + 1
}

func (r *Runtime) RecordDefect(runID string, request DefectReadingRequest) (DefectRecordResult, error) {
	if _, ok := r.Runs.Get(runID); !ok {
		return DefectRecordResult{}, fmt.Errorf("run %s not found", runID)
	}
	if strings.TrimSpace(request.SourceID) == "" {
		return DefectRecordResult{}, errors.New("defect source is required")
	}
	if !model.ValidProbeKind(request.Probe) {
		return DefectRecordResult{}, fmt.Errorf("invalid probe kind %q", request.Probe)
	}
	if !model.ValidAnomalyKind(request.Kind) {
		return DefectRecordResult{}, fmt.Errorf("invalid anomaly kind %q", request.Kind)
	}
	if request.Kilometer < 0 || request.Kilometer > r.maxLineKilometer() {
		return DefectRecordResult{}, fmt.Errorf("kilometer %.4f is outside the line", request.Kilometer)
	}
	if request.DepthPercent < 0 || request.DepthPercent > 100 {
		return DefectRecordResult{}, errors.New("depth must be within 0 and 100 percent")
	}
	if request.LengthMM < 0 || request.WidthMM < 0 {
		return DefectRecordResult{}, errors.New("dimensions must be non-negative")
	}
	if request.RecordedAt.IsZero() {
		request.RecordedAt = time.Now().UTC()
	}
	reading := model.DefectReading{
		ID:           model.NewRevision(),
		RunID:        runID,
		SourceID:     strings.TrimSpace(request.SourceID),
		Probe:        request.Probe,
		Kind:         request.Kind,
		Kilometer:    request.Kilometer,
		DepthPercent: request.DepthPercent,
		LengthMM:     request.LengthMM,
		WidthMM:      request.WidthMM,
		RecordedAt:   request.RecordedAt,
	}
	recorded, err := r.Defects.Record(reading)
	if err != nil {
		return DefectRecordResult{}, err
	}
	if err := r.appendEvent(model.Event{
		RunID:   runID,
		Kind:    "defect_reading",
		Payload: fmt.Sprintf("%s/%s/%.4f/%s", recorded.Record.ID, reading.ID, reading.Kilometer, reading.SourceID),
	}); err != nil {
		return DefectRecordResult{}, err
	}
	if recorded.Reopened {
		if err := r.appendEvent(model.Event{
			RunID:   runID,
			Kind:    "defect_reopened",
			Payload: recorded.Record.ID,
		}); err != nil {
			return DefectRecordResult{}, err
		}
	}
	return DefectRecordResult{Merged: recorded.Merged, Reopened: recorded.Reopened, Defect: recorded.Record}, nil
}

func (r *Runtime) AssessDefect(defectID string, request DefectAssessmentRequest) (model.DefectRecord, error) {
	record, err := r.Defects.Assess(defectID, request.Severity, request.Status, strings.TrimSpace(request.Reviewer), request.Note)
	if err != nil {
		return model.DefectRecord{}, err
	}
	if err := r.appendEvent(model.Event{
		RunID:   record.RunID,
		Kind:    "defect_assessment",
		Epoch:   record.LatestAssessment().Version,
		Payload: fmt.Sprintf("%s/%s/%s", record.ID, request.Status, request.Severity),
	}); err != nil {
		return model.DefectRecord{}, err
	}
	return record, nil
}

func (r *Runtime) RunDefects(runID string) ([]model.DefectRecord, error) {
	if _, ok := r.Runs.Get(runID); !ok {
		return nil, fmt.Errorf("run %s not found", runID)
	}
	return r.Defects.List(runID), nil
}

func (r *Runtime) PendingDefects() []model.DefectRecord { return r.Defects.Pending() }

func (r *Runtime) DefectDistribution(runID string) DefectDistribution {
	defects := r.Defects.List(runID)
	control := r.ensureControl()
	stations := make(map[string]float64)
	for _, station := range control.Network.Stations() {
		stations[station.ID] = station.Kilometer
	}
	segments := control.Network.Segments()
	sort.Slice(segments, func(i, j int) bool {
		return stations[segments[i].From] < stations[segments[j].From]
	})
	buckets := make([]DefectBucket, 0, len(segments))
	for _, segment := range segments {
		buckets = append(buckets, DefectBucket{
			SegmentID: segment.ID,
			FromKm:    stations[segment.From],
			ToKm:      stations[segment.To],
			ByKind:    make(map[string]int),
		})
	}
	summary := DefectSummary{ByKind: make(map[string]int), BySeverity: make(map[string]int)}
	for _, record := range defects {
		summary.Total++
		summary.ByKind[string(record.Kind)]++
		summary.BySeverity[string(record.LatestAssessment().Severity)]++
		if record.Status() == model.ReviewPending {
			summary.PendingReview++
		}
		if len(buckets) == 0 {
			continue
		}
		target := len(buckets) - 1
		for index := range buckets {
			if record.Kilometer <= buckets[index].ToKm {
				target = index
				break
			}
		}
		buckets[target].Count++
		buckets[target].ByKind[string(record.Kind)]++
		if record.Status() == model.ReviewPending {
			buckets[target].Pending++
		}
	}
	return DefectDistribution{Defects: defects, Buckets: buckets, Summary: summary}
}
