package defect

import (
	"sync"

	"github.com/wyw14/solo-036/internal/model"
)

// DefaultMergeWindowKm is the ±1 m location tolerance for merging duplicate
// findings of one run into a single defect.
const DefaultMergeWindowKm = 0.001

type Store struct {
	mu            sync.RWMutex
	defects       map[string]model.DefectRecord
	mergeWindowKm float64
}

func NewStore(mergeWindowKm float64) *Store {
	if mergeWindowKm <= 0 {
		mergeWindowKm = DefaultMergeWindowKm
	}
	return &Store{defects: make(map[string]model.DefectRecord), mergeWindowKm: mergeWindowKm}
}

func (s *Store) SetMergeWindowMeters(meters float64) {
	if meters <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mergeWindowKm = meters / 1000
}

func (s *Store) MergeWindowKm() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mergeWindowKm
}
