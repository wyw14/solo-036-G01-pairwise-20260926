package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ProbeKind string

const (
	ProbeMFL     ProbeKind = "mfl"
	ProbeCaliper ProbeKind = "caliper"
	ProbeUT      ProbeKind = "ut"
	ProbeOther   ProbeKind = "other"
)

type AnomalyKind string

const (
	AnomalyMetalLoss AnomalyKind = "metal_loss"
	AnomalyDent      AnomalyKind = "dent"
	AnomalyOvality   AnomalyKind = "ovality"
	AnomalyOther     AnomalyKind = "other"
)

type ReviewStatus string

const (
	ReviewPending   ReviewStatus = "pending_review"
	ReviewConfirmed ReviewStatus = "confirmed"
	ReviewDismissed ReviewStatus = "dismissed"
)

type Severity string

const (
	SeverityMinor    Severity = "minor"
	SeverityModerate Severity = "moderate"
	SeveritySevere   Severity = "severe"
	SeverityCritical Severity = "critical"
)

func ValidProbeKind(kind ProbeKind) bool {
	switch kind {
	case ProbeMFL, ProbeCaliper, ProbeUT, ProbeOther:
		return true
	}
	return false
}

func ValidAnomalyKind(kind AnomalyKind) bool {
	switch kind {
	case AnomalyMetalLoss, AnomalyDent, AnomalyOvality, AnomalyOther:
		return true
	}
	return false
}

func ValidReviewStatus(status ReviewStatus) bool {
	switch status {
	case ReviewPending, ReviewConfirmed, ReviewDismissed:
		return true
	}
	return false
}

func ValidSeverity(severity Severity) bool {
	switch severity {
	case SeverityMinor, SeverityModerate, SeveritySevere, SeverityCritical:
		return true
	}
	return false
}

// DefectReading is one raw inline-inspection measurement. Readings are
// immutable: merging never rewrites them, it only attaches them to a defect.
type DefectReading struct {
	ID           string      `json:"id"`
	RunID        string      `json:"run_id"`
	DefectID     string      `json:"defect_id"`
	SourceID     string      `json:"source_id"`
	Probe        ProbeKind   `json:"probe"`
	Kind         AnomalyKind `json:"kind"`
	Kilometer    float64     `json:"kilometer"`
	DepthPercent float64     `json:"depth_percent"`
	LengthMM     float64     `json:"length_mm"`
	WidthMM      float64     `json:"width_mm"`
	RecordedAt   time.Time   `json:"recorded_at"`
}

// DefectAssessment is one versioned judgment over a defect. Versions are
// append-only; the latest version is the current judgment.
type DefectAssessment struct {
	Version  int64        `json:"version"`
	Severity Severity     `json:"severity"`
	Status   ReviewStatus `json:"status"`
	Reviewer string       `json:"reviewer"`
	Note     string       `json:"note"`
	System   bool         `json:"system"`
	At       time.Time    `json:"at"`
}

// DefectRecord is a merged anomaly at one location of one run. It keeps every
// raw reading and every assessment version.
type DefectRecord struct {
	ID          string             `json:"id"`
	RunID       string             `json:"run_id"`
	Kind        AnomalyKind        `json:"kind"`
	Kilometer   float64            `json:"kilometer"`
	Readings    []DefectReading    `json:"readings"`
	Assessments []DefectAssessment `json:"assessments"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

func (d DefectRecord) LatestAssessment() DefectAssessment {
	if len(d.Assessments) == 0 {
		return DefectAssessment{Status: ReviewPending}
	}
	return d.Assessments[len(d.Assessments)-1]
}

func (d DefectRecord) Status() ReviewStatus { return d.LatestAssessment().Status }

func (d DefectRecord) MaxDepthPercent() float64 {
	depth := 0.0
	for _, reading := range d.Readings {
		if reading.DepthPercent > depth {
			depth = reading.DepthPercent
		}
	}
	return depth
}

func (d DefectRecord) SuggestedSeverity() Severity {
	switch depth := d.MaxDepthPercent(); {
	case depth >= 50:
		return SeverityCritical
	case depth >= 30:
		return SeveritySevere
	case depth >= 15:
		return SeverityModerate
	default:
		return SeverityMinor
	}
}

func (r DefectReading) Validate(maxKilometer float64) error {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.RunID) == "" || strings.TrimSpace(r.DefectID) == "" {
		return errors.New("reading identity is incomplete")
	}
	if strings.TrimSpace(r.SourceID) == "" {
		return errors.New("reading source is required")
	}
	if !ValidProbeKind(r.Probe) {
		return fmt.Errorf("invalid probe kind %q", r.Probe)
	}
	if !ValidAnomalyKind(r.Kind) {
		return fmt.Errorf("invalid anomaly kind %q", r.Kind)
	}
	if r.Kilometer < 0 || (maxKilometer > 0 && r.Kilometer > maxKilometer) {
		return fmt.Errorf("reading kilometer %.4f is outside the line", r.Kilometer)
	}
	if r.DepthPercent < 0 || r.DepthPercent > 100 {
		return errors.New("reading depth must be within 0 and 100 percent")
	}
	if r.LengthMM < 0 || r.WidthMM < 0 {
		return errors.New("reading dimensions must be non-negative")
	}
	if r.RecordedAt.IsZero() {
		return errors.New("reading time is required")
	}
	return nil
}

func (a DefectAssessment) Validate() error {
	if a.Version <= 0 {
		return errors.New("assessment version must be positive")
	}
	if !ValidSeverity(a.Severity) {
		return fmt.Errorf("invalid severity %q", a.Severity)
	}
	if !ValidReviewStatus(a.Status) {
		return fmt.Errorf("invalid review status %q", a.Status)
	}
	if a.At.IsZero() {
		return errors.New("assessment time is required")
	}
	return nil
}

func (d DefectRecord) Validate(maxKilometer float64) error {
	if strings.TrimSpace(d.ID) == "" || strings.TrimSpace(d.RunID) == "" {
		return errors.New("defect identity is incomplete")
	}
	if !ValidAnomalyKind(d.Kind) {
		return fmt.Errorf("invalid anomaly kind %q", d.Kind)
	}
	if d.Kilometer < 0 || (maxKilometer > 0 && d.Kilometer > maxKilometer) {
		return fmt.Errorf("defect kilometer %.4f is outside the line", d.Kilometer)
	}
	if len(d.Readings) == 0 {
		return errors.New("defect has no readings")
	}
	for _, reading := range d.Readings {
		if reading.DefectID != d.ID || reading.RunID != d.RunID || reading.Kind != d.Kind {
			return fmt.Errorf("reading %s does not belong to defect %s", reading.ID, d.ID)
		}
		if err := reading.Validate(maxKilometer); err != nil {
			return fmt.Errorf("reading %s: %w", reading.ID, err)
		}
	}
	if len(d.Assessments) == 0 {
		return errors.New("defect has no assessment")
	}
	for index, assessment := range d.Assessments {
		if err := assessment.Validate(); err != nil {
			return fmt.Errorf("assessment %d: %w", index+1, err)
		}
		if assessment.Version != int64(index+1) {
			return fmt.Errorf("assessment versions are not sequential at %d", index+1)
		}
	}
	return nil
}
