package defect

import (
	"fmt"
	"time"

	"github.com/wyw14/solo-036/internal/model"
)

// Assess appends a new judgment version to a defect. History is never
// rewritten; the appended version becomes the current judgment.
func (s *Store) Assess(defectID string, severity model.Severity, status model.ReviewStatus, reviewer, note string) (model.DefectRecord, error) {
	if !model.ValidSeverity(severity) {
		return model.DefectRecord{}, fmt.Errorf("invalid severity %q", severity)
	}
	if !model.ValidReviewStatus(status) {
		return model.DefectRecord{}, fmt.Errorf("invalid review status %q", status)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.defects[defectID]
	if !ok {
		return model.DefectRecord{}, fmt.Errorf("defect %s not found", defectID)
	}
	record.Assessments = append(record.Assessments, model.DefectAssessment{
		Version:  int64(len(record.Assessments)) + 1,
		Severity: severity,
		Status:   status,
		Reviewer: reviewer,
		Note:     note,
		System:   false,
		At:       time.Now().UTC(),
	})
	record.UpdatedAt = time.Now().UTC()
	s.defects[defectID] = record
	return record, nil
}
