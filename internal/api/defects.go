package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/solo-036/internal/defect"
	"github.com/wyw14/solo-036/internal/service"
)

type defectReadingRequest struct {
	Kind       defect.Kind   `json:"kind"`
	Source     defect.Source `json:"source"`
	Kilometer  float64       `json:"kilometer"`
	Value      float64       `json:"value"`
	Unit       string        `json:"unit"`
	ObservedAt string        `json:"observed_at"`
}

type defectAssessRequest struct {
	Status     defect.Status   `json:"status"`
	Severity   defect.Severity `json:"severity"`
	Note       string          `json:"note"`
	AssessedBy string          `json:"assessed_by"`
}

func (h *Handler) RecordDefect(w http.ResponseWriter, r *http.Request) {
	var request defectReadingRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	observedAt, err := parseObservedAt(request.ObservedAt)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	merged, err := h.runtime.RecordDefect(chi.URLParam(r, "runID"), service.DefectReadingRequest{
		Kind:       request.Kind,
		Source:     request.Source,
		Kilometer:  request.Kilometer,
		Value:      request.Value,
		Unit:       request.Unit,
		ObservedAt: observedAt,
	})
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, merged)
}

func (h *Handler) Defects(w http.ResponseWriter, r *http.Request) {
	defects, err := h.runtime.Defects(chi.URLParam(r, "runID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, defects)
}

func (h *Handler) DefectDistribution(w http.ResponseWriter, r *http.Request) {
	bucketKm := defect.DefaultBucketKm
	if raw := r.URL.Query().Get("bucket_km"); raw != "" {
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil || parsed <= 0 {
			writeError(w, r, http.StatusBadRequest, errors.New("bucket_km must be a positive number"))
			return
		}
		bucketKm = parsed
	}
	distribution, err := h.runtime.DefectDistribution(chi.URLParam(r, "runID"), bucketKm)
	if err != nil {
		writeError(w, r, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, distribution)
}

func (h *Handler) PendingDefects(w http.ResponseWriter, r *http.Request) {
	defects, err := h.runtime.PendingDefects(chi.URLParam(r, "runID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, defects)
}

func (h *Handler) AssessDefect(w http.ResponseWriter, r *http.Request) {
	var request defectAssessRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	assessed, err := h.runtime.AssessDefect(chi.URLParam(r, "runID"), chi.URLParam(r, "defectID"), service.DefectAssessRequest{
		Status:     request.Status,
		Severity:   request.Severity,
		Note:       request.Note,
		AssessedBy: request.AssessedBy,
	})
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, assessed)
}
