package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/solo-036/internal/model"
	"github.com/wyw14/solo-036/internal/service"
)

type defectReadingRequest struct {
	SourceID     string  `json:"source_id"`
	Probe        string  `json:"probe"`
	Kind         string  `json:"kind"`
	Kilometer    float64 `json:"kilometer"`
	DepthPercent float64 `json:"depth_percent"`
	LengthMM     float64 `json:"length_mm"`
	WidthMM      float64 `json:"width_mm"`
	RecordedAt   string  `json:"recorded_at"`
}

type defectAssessmentRequest struct {
	Severity string `json:"severity"`
	Status   string `json:"status"`
	Reviewer string `json:"reviewer"`
	Note     string `json:"note"`
}

func (h *Handler) RecordDefect(w http.ResponseWriter, r *http.Request) {
	var request defectReadingRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	recordedAt, err := parseObservedAt(request.RecordedAt)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	result, err := h.runtime.RecordDefect(chi.URLParam(r, "runID"), service.DefectReadingRequest{
		SourceID:     request.SourceID,
		Probe:        model.ProbeKind(request.Probe),
		Kind:         model.AnomalyKind(request.Kind),
		Kilometer:    request.Kilometer,
		DepthPercent: request.DepthPercent,
		LengthMM:     request.LengthMM,
		WidthMM:      request.WidthMM,
		RecordedAt:   recordedAt,
	})
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) RunDefects(w http.ResponseWriter, r *http.Request) {
	defects, err := h.runtime.RunDefects(chi.URLParam(r, "runID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, defects)
}

func (h *Handler) AssessDefect(w http.ResponseWriter, r *http.Request) {
	var request defectAssessmentRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(request.Reviewer) == "" {
		writeError(w, r, http.StatusBadRequest, errors.New("reviewer is required"))
		return
	}
	record, err := h.runtime.AssessDefect(chi.URLParam(r, "defectID"), service.DefectAssessmentRequest{
		Severity: model.Severity(request.Severity),
		Status:   model.ReviewStatus(request.Status),
		Reviewer: request.Reviewer,
		Note:     request.Note,
	})
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (h *Handler) DefectDistribution(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimSpace(r.URL.Query().Get("run_id"))
	writeJSON(w, http.StatusOK, h.runtime.DefectDistribution(runID))
}

func (h *Handler) PendingDefects(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.runtime.PendingDefects())
}
