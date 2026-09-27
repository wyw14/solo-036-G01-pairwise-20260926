package defect

import "sort"

// Bucket aggregates merged defects over one interval of the line.
type Bucket struct {
	StartKm       float64 `json:"start_km"`
	EndKm         float64 `json:"end_km"`
	Defects       int     `json:"defects"`
	PendingReview int     `json:"pending_review"`
	Confirmed     int     `json:"confirmed"`
	Dismissed     int     `json:"dismissed"`
	WallLoss      int     `json:"wall_loss"`
	Geometry      int     `json:"geometry"`
}

// Distribution is the along-line defect profile of one run. Only buckets
// that contain at least one defect are returned.
type Distribution struct {
	RunID    string   `json:"run_id"`
	BucketKm float64  `json:"bucket_km"`
	Buckets  []Bucket `json:"buckets"`
}

// Distribute groups defects into intervals of bucketKm along the line.
func Distribute(runID string, defects []Defect, bucketKm float64) Distribution {
	if bucketKm <= 0 {
		bucketKm = DefaultBucketKm
	}
	byIndex := make(map[int]*Bucket)
	for _, defect := range defects {
		index := int(defect.Kilometer / bucketKm)
		bucket, ok := byIndex[index]
		if !ok {
			bucket = &Bucket{
				StartKm: float64(index) * bucketKm,
				EndKm:   float64(index+1) * bucketKm,
			}
			byIndex[index] = bucket
		}
		bucket.Defects++
		switch defect.Status {
		case StatusPendingReview:
			bucket.PendingReview++
		case StatusConfirmed:
			bucket.Confirmed++
		case StatusDismissed:
			bucket.Dismissed++
		}
		switch defect.Kind {
		case KindWallLoss:
			bucket.WallLoss++
		case KindGeometry:
			bucket.Geometry++
		}
	}
	buckets := make([]Bucket, 0, len(byIndex))
	for _, bucket := range byIndex {
		buckets = append(buckets, *bucket)
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].StartKm < buckets[j].StartKm })
	return Distribution{RunID: runID, BucketKm: bucketKm, Buckets: buckets}
}
