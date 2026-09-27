PigFlow is a local HTTP control service for long-distance pipeline pigging operations.

## Inline inspection defects

Defects found by inspection tools (wall thickness loss, geometry anomalies)
are recorded per run, per kilometer, per probe source:

- `POST /api/operations/{runID}/defects` — record a raw reading
  (`kind`: `wall_loss`/`geometry`, `source`: `mfl`/`ultrasonic`/`caliper`,
  `kilometer`, `value`, `unit`). Readings of the same kind within 0.5 m are
  merged into one defect; raw readings are always retained.
- `POST /api/operations/{runID}/defects/{defectID}/assess` — append a
  versioned judgment (`status`: `confirmed`/`dismissed`, `severity`:
  `minor`/`moderate`/`severe`). New readings on an assessed defect return it
  to `pending_review`.
- `GET /api/operations/{runID}/defects` — merged defects ordered by kilometer.
- `GET /api/operations/{runID}/defects/distribution?bucket_km=10` — along-line
  distribution aggregated into buckets.
- `GET /api/operations/{runID}/defects/review` — defects awaiting review.
