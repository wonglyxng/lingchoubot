package model

import "time"

type ReviewVerdict string

const (
	ReviewVerdictApproved      ReviewVerdict = "approved"
	ReviewVerdictRejected      ReviewVerdict = "rejected"
	ReviewVerdictNeedsRevision ReviewVerdict = "needs_revision"
)

type ReviewReport struct {
	ID                string        `json:"id"`
	RunID             *string       `json:"run_id,omitempty"`
	TaskID            string        `json:"task_id"`
	ReviewerID        string        `json:"reviewer_id"`
	ArtifactVersionID *string       `json:"artifact_version_id,omitempty"`
	Verdict           ReviewVerdict `json:"verdict"`
	Summary           string        `json:"summary"`
	Findings          JSON          `json:"findings"`
	Recommendations   JSON          `json:"recommendations"`
	Metadata          JSON          `json:"metadata"`
	CreatedAt         time.Time     `json:"created_at"`
}
