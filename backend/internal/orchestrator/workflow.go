// Package orchestrator implements the workflow engine.
//
// Workflow run state is persisted via model.WorkflowRun / model.WorkflowStep
// and managed through service.WorkflowService (see WP2-01).
//
// The legacy in-memory RunStore was removed after DB persistence was confirmed stable.
package orchestrator
