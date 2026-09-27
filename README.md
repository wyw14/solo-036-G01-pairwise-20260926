PigFlow is a local HTTP control service for long-distance pipeline pigging operations.

## Inline inspection defects

Findings from inspection runs (wall-thickness and geometry anomalies) are recorded per run, mileage and probe source. Duplicate findings at the same location merge into one defect; raw readings and every judgment version are kept.

- `POST /api/operations/{runID}/defects` — record a raw reading (`source_id`, `probe`: mfl/caliper/ut/other, `kind`: metal_loss/dent/ovality/other, `kilometer`, `depth_percent`, `length_mm`, `width_mm`, optional `recorded_at`). Findings within ±1 m (configurable via `PIGFLOW_DEFECT_MERGE_WINDOW_M`) of an existing defect of the same run and kind merge into it. New evidence on a judged defect reopens it for review.
- `GET /api/operations/{runID}/defects` — list defects of a run, ordered by mileage.
- `POST /api/defects/{defectID}/assessments` — append a judgment version (`severity`: minor/moderate/severe/critical, `status`: pending_review/confirmed/dismissed, `reviewer`, `note`).
- `GET /api/defects/distribution?run_id=` — defect distribution along the line: per-segment buckets, mileage-ordered detail, and severity/kind summary.
- `GET /api/defects/pending` — defects awaiting review.
