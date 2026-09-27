package defect

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Kind classifies what the inspection tool detected.
type Kind string

const (
	KindWallLoss Kind = "wall_loss"
	KindGeometry Kind = "geometry"
)

// Source identifies the probe technology that produced a reading.
type Source string

const (
	SourceMFL        Source = "mfl"
	SourceUltrasonic Source = "ultrasonic"
	SourceCaliper    Source = "caliper"
)

// Status is the review state of a merged defect.
type Status string

const (
	StatusPendingReview Status = "pending_review"
	StatusConfirmed     Status = "confirmed"
	StatusDismissed     Status = "dismissed"
)

// Severity is assigned by an assessor, not by the tool.
type Severity string

const (
	SeverityMinor    Severity = "minor"
	SeverityModerate Severity = "moderate"
	SeveritySevere   Severity = "severe"
)

// MergeToleranceKm is the maximum distance between two readings of the
// same kind for them to be treated as the same physical defect (0.5 m).
const MergeToleranceKm = 0.0005

// DefaultBucketKm is the default interval for the along-line distribution.
const DefaultBucketKm = 10.0

// Reading is one raw measurement reported by an inspection probe. Readings
// are immutable once recorded and are never rewritten by assessment.
type Reading struct {
	ID         string    `json:"id"`
	RunID      string    `json:"run_id"`
	DefectID   string    `json:"defect_id"`
	Kind       Kind      `json:"kind"`
	Source     Source    `json:"source"`
	Kilometer  float64   `json:"kilometer"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	ObservedAt time.Time `json:"observed_at"`
	RecordedAt time.Time `json:"recorded_at"`
}

// Assessment is one versioned judgment of a defect. Every reassessment
// appends a new version; older versions are retained.
type Assessment struct {
	Version    int       `json:"version"`
	Status     Status    `json:"status"`
	Severity   Severity  `json:"severity"`
	Note       string    `json:"note,omitempty"`
	AssessedBy string    `json:"assessed_by"`
	At         time.Time `json:"at"`
}

// Defect is a merged cluster of readings at one location along the line.
type Defect struct {
	ID          string       `json:"id"`
	RunID       string       `json:"run_id"`
	Kind        Kind         `json:"kind"`
	Kilometer   float64      `json:"kilometer"`
	Status      Status       `json:"status"`
	Readings    []Reading    `json:"readings"`
	Assessments []Assessment `json:"assessments"`
	FirstAt     time.Time    `json:"first_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func ValidKind(kind Kind) bool {
	return kind == KindWallLoss || kind == KindGeometry
}

func ValidSource(source Source) bool {
	switch source {
	case SourceMFL, SourceUltrasonic, SourceCaliper:
		return true
	}
	return false
}

func ValidSeverity(severity Severity) bool {
	switch severity {
	case SeverityMinor, SeverityModerate, SeveritySevere:
		return true
	}
	return false
}

func (r Reading) Validate() error {
	if strings.TrimSpace(r.RunID) == "" {
		return errors.New("reading run is required")
	}
	if !ValidKind(r.Kind) {
		return fmt.Errorf("invalid defect kind %q", r.Kind)
	}
	if !ValidSource(r.Source) {
		return fmt.Errorf("invalid probe source %q", r.Source)
	}
	if r.Kilometer < 0 {
		return errors.New("reading kilometer must be non-negative")
	}
	if r.Value < 0 {
		return errors.New("reading value must be non-negative")
	}
	if strings.TrimSpace(r.Unit) == "" {
		return errors.New("reading unit is required")
	}
	return nil
}

// LatestAssessment returns the most recent judgment, if any.
func (d Defect) LatestAssessment() (Assessment, bool) {
	if len(d.Assessments) == 0 {
		return Assessment{}, false
	}
	return d.Assessments[len(d.Assessments)-1], true
}
