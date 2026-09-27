package defect

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Store struct {
	mu      sync.RWMutex
	defects map[string]map[string]Defect // runID -> defectID -> defect
}

func NewStore() *Store {
	return &Store{defects: make(map[string]map[string]Defect)}
}

// AddReading records a raw reading and merges it into the defect cluster at
// the same location (same run, same kind, within MergeToleranceKm), creating
// a new cluster when none matches. New evidence on an already-assessed
// defect returns it to pending_review so it gets re-examined.
func (s *Store) AddReading(reading Reading, now time.Time) (Defect, error) {
	if err := reading.Validate(); err != nil {
		return Defect{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	runDefects, ok := s.defects[reading.RunID]
	if !ok {
		runDefects = make(map[string]Defect)
		s.defects[reading.RunID] = runDefects
	}
	for _, existing := range runDefects {
		if existing.Kind != reading.Kind {
			continue
		}
		if delta := existing.Kilometer - reading.Kilometer; delta > MergeToleranceKm || delta < -MergeToleranceKm {
			continue
		}
		reading.DefectID = existing.ID
		reading.RecordedAt = now
		existing.Readings = append(existing.Readings, reading)
		existing.Kilometer = clusterCenter(existing.Readings)
		existing.Status = StatusPendingReview
		existing.UpdatedAt = now
		runDefects[existing.ID] = existing
		return copyDefect(existing), nil
	}
	defect := Defect{
		ID:          reading.ID,
		RunID:       reading.RunID,
		Kind:        reading.Kind,
		Status:      StatusPendingReview,
		Assessments: make([]Assessment, 0),
		FirstAt:     now,
		UpdatedAt:   now,
	}
	reading.DefectID = defect.ID
	reading.RecordedAt = now
	defect.Readings = []Reading{reading}
	defect.Kilometer = reading.Kilometer
	runDefects[defect.ID] = defect
	return copyDefect(defect), nil
}

// Assess appends a new judgment version to a defect and sets its status.
func (s *Store) Assess(runID, defectID string, status Status, severity Severity, note, assessedBy string, now time.Time) (Defect, error) {
	if status != StatusConfirmed && status != StatusDismissed {
		return Defect{}, fmt.Errorf("assessment status must be %s or %s", StatusConfirmed, StatusDismissed)
	}
	if !ValidSeverity(severity) {
		return Defect{}, fmt.Errorf("invalid severity %q", severity)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	runDefects, ok := s.defects[runID]
	if !ok {
		return Defect{}, fmt.Errorf("run %s has no defects", runID)
	}
	defect, ok := runDefects[defectID]
	if !ok {
		return Defect{}, fmt.Errorf("defect %s not found in run %s", defectID, runID)
	}
	defect.Assessments = append(defect.Assessments, Assessment{
		Version:    len(defect.Assessments) + 1,
		Status:     status,
		Severity:   severity,
		Note:       note,
		AssessedBy: assessedBy,
		At:         now,
	})
	defect.Status = status
	defect.UpdatedAt = now
	runDefects[defectID] = defect
	return copyDefect(defect), nil
}

func (s *Store) Get(runID, defectID string) (Defect, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	runDefects, ok := s.defects[runID]
	if !ok {
		return Defect{}, false
	}
	defect, ok := runDefects[defectID]
	return copyDefect(defect), ok
}

// List returns all merged defects of a run ordered by kilometer.
func (s *Store) List(runID string) []Defect {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Defect, 0, len(s.defects[runID]))
	for _, defect := range s.defects[runID] {
		out = append(out, copyDefect(defect))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kilometer == out[j].Kilometer {
			return out[i].ID < out[j].ID
		}
		return out[i].Kilometer < out[j].Kilometer
	})
	return out
}

// PendingReview returns defects still awaiting judgment, ordered by kilometer.
func (s *Store) PendingReview(runID string) []Defect {
	out := make([]Defect, 0)
	for _, defect := range s.List(runID) {
		if defect.Status == StatusPendingReview {
			out = append(out, defect)
		}
	}
	return out
}

func clusterCenter(readings []Reading) float64 {
	total := 0.0
	for _, reading := range readings {
		total += reading.Kilometer
	}
	return total / float64(len(readings))
}

func copyDefect(defect Defect) Defect {
	defect.Readings = append([]Reading(nil), defect.Readings...)
	defect.Assessments = append([]Assessment(nil), defect.Assessments...)
	return defect
}
