package defect

import (
	"sort"

	"github.com/wyw14/solo-036/internal/model"
)

func (s *Store) Get(id string) (model.DefectRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.defects[id]
	return record, ok
}

func (s *Store) List(runID string) []model.DefectRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.DefectRecord, 0)
	for _, record := range s.defects {
		if runID == "" || record.RunID == runID {
			out = append(out, record)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kilometer == out[j].Kilometer {
			return out[i].ID < out[j].ID
		}
		return out[i].Kilometer < out[j].Kilometer
	})
	return out
}

// Pending returns defects whose latest judgment still needs review, ordered
// along the line.
func (s *Store) Pending() []model.DefectRecord {
	out := make([]model.DefectRecord, 0)
	for _, record := range s.List("") {
		if record.Status() == model.ReviewPending {
			out = append(out, record)
		}
	}
	return out
}

func (s *Store) Snapshot() []model.DefectRecord {
	return s.List("")
}

func Restore(store *Store, records []model.DefectRecord) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.defects = make(map[string]model.DefectRecord, len(records))
	for _, record := range records {
		store.defects[record.ID] = record
	}
}
