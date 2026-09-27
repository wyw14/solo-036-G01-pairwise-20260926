package defect

import (
	"errors"
	"fmt"
	"time"

	"github.com/wyw14/solo-036/internal/model"
)

type RecordResult struct {
	Record   model.DefectRecord
	Merged   bool
	Reopened bool
}

// Record attaches a raw reading to the nearest defect of the same run and
// anomaly kind within the merge window, or opens a new defect. When new
// evidence lands on an already judged defect, a system assessment version
// returns it to pending review.
func (s *Store) Record(reading model.DefectReading) (RecordResult, error) {
	if reading.ID == "" || reading.RunID == "" {
		return RecordResult{}, errors.New("reading identity is incomplete")
	}
	if !model.ValidAnomalyKind(reading.Kind) || !model.ValidProbeKind(reading.Probe) {
		return RecordResult{}, errors.New("reading kind or probe is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, defect := range s.defects {
		for _, existing := range defect.Readings {
			if existing.ID == reading.ID {
				return RecordResult{}, fmt.Errorf("reading %s is already recorded", reading.ID)
			}
		}
	}
	target, found := s.nearestLocked(reading)
	now := time.Now().UTC()
	if !found {
		record := model.DefectRecord{
			ID:        model.NewRevision(),
			RunID:     reading.RunID,
			Kind:      reading.Kind,
			Kilometer: reading.Kilometer,
			CreatedAt: now,
			UpdatedAt: now,
		}
		reading.DefectID = record.ID
		record.Readings = []model.DefectReading{reading}
		record.Assessments = []model.DefectAssessment{{
			Version:  1,
			Severity: record.SuggestedSeverity(),
			Status:   model.ReviewPending,
			Reviewer: "system",
			Note:     "initial assessment from first reading",
			System:   true,
			At:       now,
		}}
		s.defects[record.ID] = record
		return RecordResult{Record: record}, nil
	}
	reading.DefectID = target.ID
	target.Readings = append(target.Readings, reading)
	target.Kilometer = meanKilometer(target.Readings)
	target.UpdatedAt = now
	result := RecordResult{Merged: true}
	if status := target.Status(); status == model.ReviewConfirmed || status == model.ReviewDismissed {
		target.Assessments = append(target.Assessments, model.DefectAssessment{
			Version:  int64(len(target.Assessments)) + 1,
			Severity: target.SuggestedSeverity(),
			Status:   model.ReviewPending,
			Reviewer: "system",
			Note:     fmt.Sprintf("reading %s merged into judged defect; review reopened", reading.ID),
			System:   true,
			At:       now,
		})
		result.Reopened = true
	}
	s.defects[target.ID] = target
	result.Record = target
	return result, nil
}

func (s *Store) nearestLocked(reading model.DefectReading) (model.DefectRecord, bool) {
	var best model.DefectRecord
	bestDistance := s.mergeWindowKm
	found := false
	for _, defect := range s.defects {
		if defect.RunID != reading.RunID || defect.Kind != reading.Kind {
			continue
		}
		distance := defect.Kilometer - reading.Kilometer
		if distance < 0 {
			distance = -distance
		}
		if distance <= bestDistance {
			best, bestDistance, found = defect, distance, true
		}
	}
	return best, found
}

func meanKilometer(readings []model.DefectReading) float64 {
	total := 0.0
	for _, reading := range readings {
		total += reading.Kilometer
	}
	return total / float64(len(readings))
}
