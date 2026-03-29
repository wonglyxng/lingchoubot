package model

import "time"

type ArtifactType string

const (
	ArtifactTypePRD            ArtifactType = "prd"
	ArtifactTypeDesign         ArtifactType = "design"
	ArtifactTypeAPISpec        ArtifactType = "api_spec"
	ArtifactTypeSchemaSQL      ArtifactType = "schema_sql"
	ArtifactTypeSourceCode     ArtifactType = "source_code"
	ArtifactTypeTestReport     ArtifactType = "test_report"
	ArtifactTypeDeploymentPlan ArtifactType = "deployment_plan"
	ArtifactTypeReleaseNote    ArtifactType = "release_note"
	ArtifactTypeOther          ArtifactType = "other"
)

type Artifact struct {
	ID           string       `json:"id"`
	ProjectID    string       `json:"project_id"`
	TaskID       *string      `json:"task_id,omitempty"`
	Name         string       `json:"name"`
	ArtifactType ArtifactType `json:"artifact_type"`
	Description  string       `json:"description"`
	CreatedBy    *string      `json:"created_by,omitempty"`
	Metadata     JSON         `json:"metadata"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type ArtifactVersion struct {
	ID            string    `json:"id"`
	ArtifactID    string    `json:"artifact_id"`
	Version       int       `json:"version"`
	URI           string    `json:"uri"`
	ContentType   string    `json:"content_type"`
	SizeBytes     int64     `json:"size_bytes"`
	Checksum      string    `json:"checksum"`
	ChangeSummary string    `json:"change_summary"`
	CreatedBy     *string   `json:"created_by,omitempty"`
	Metadata      JSON      `json:"metadata"`
	CreatedAt     time.Time `json:"created_at"`
}
