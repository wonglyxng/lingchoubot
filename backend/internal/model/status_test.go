package model

import (
	"testing"
)

func TestProjectStatusConstants(t *testing.T) {
	statuses := []ProjectStatus{
		ProjectStatusPlanning,
		ProjectStatusActive,
		ProjectStatusPaused,
		ProjectStatusCompleted,
		ProjectStatusCancelled,
	}
	seen := map[ProjectStatus]bool{}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty project status constant")
		}
		if seen[s] {
			t.Errorf("duplicate project status: %s", s)
		}
		seen[s] = true
	}
	if len(statuses) != 5 {
		t.Errorf("expected 5 project statuses, got %d", len(statuses))
	}
}

func TestPhaseStatusConstants(t *testing.T) {
	statuses := []PhaseStatus{
		PhaseStatusPending,
		PhaseStatusActive,
		PhaseStatusCompleted,
		PhaseStatusSkipped,
	}
	seen := map[PhaseStatus]bool{}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty phase status constant")
		}
		if seen[s] {
			t.Errorf("duplicate phase status: %s", s)
		}
		seen[s] = true
	}
	if len(statuses) != 4 {
		t.Errorf("expected 4 phase statuses, got %d", len(statuses))
	}
}

func TestTaskStatusTransitions_Comprehensive(t *testing.T) {
	// All valid transitions
	validCases := []struct {
		from, to TaskStatus
	}{
		{TaskStatusPending, TaskStatusAssigned},
		{TaskStatusPending, TaskStatusCancelled},
		{TaskStatusAssigned, TaskStatusInProgress},
		{TaskStatusAssigned, TaskStatusCancelled},
		{TaskStatusInProgress, TaskStatusInReview},
		{TaskStatusInProgress, TaskStatusCompleted},
		{TaskStatusInProgress, TaskStatusFailed},
		{TaskStatusInProgress, TaskStatusBlocked},
		{TaskStatusInProgress, TaskStatusCancelled},
		{TaskStatusInReview, TaskStatusPendingApproval},
		{TaskStatusInReview, TaskStatusRevisionRequired},
		{TaskStatusInReview, TaskStatusFailed},
		{TaskStatusPendingApproval, TaskStatusCompleted},
		{TaskStatusPendingApproval, TaskStatusRevisionRequired},
		{TaskStatusPendingApproval, TaskStatusFailed},
		{TaskStatusRevisionRequired, TaskStatusAssigned},
		{TaskStatusRevisionRequired, TaskStatusInProgress},
		{TaskStatusRevisionRequired, TaskStatusCancelled},
		{TaskStatusBlocked, TaskStatusInProgress},
		{TaskStatusBlocked, TaskStatusCancelled},
		{TaskStatusFailed, TaskStatusPending},
		{TaskStatusFailed, TaskStatusCancelled},
	}
	for _, c := range validCases {
		if !c.from.CanTransitionTo(c.to) {
			t.Errorf("expected valid transition %s -> %s", c.from, c.to)
		}
	}

	// Invalid transitions
	invalidCases := []struct {
		from, to TaskStatus
	}{
		{TaskStatusPending, TaskStatusCompleted},
		{TaskStatusPending, TaskStatusInProgress},
		{TaskStatusPending, TaskStatusInReview},
		{TaskStatusAssigned, TaskStatusCompleted},
		{TaskStatusAssigned, TaskStatusInReview},
		{TaskStatusCompleted, TaskStatusPending},
		{TaskStatusCompleted, TaskStatusInProgress},
		{TaskStatusCancelled, TaskStatusPending},
		{TaskStatusCancelled, TaskStatusInProgress},
		{TaskStatusInReview, TaskStatusPending},
		{TaskStatusInReview, TaskStatusAssigned},
		{TaskStatusInReview, TaskStatusCompleted},
		{TaskStatusBlocked, TaskStatusCompleted},
	}
	for _, c := range invalidCases {
		if c.from.CanTransitionTo(c.to) {
			t.Errorf("expected invalid transition %s -> %s", c.from, c.to)
		}
	}

	// Self-transitions should be invalid
	allStatuses := []TaskStatus{
		TaskStatusPending, TaskStatusAssigned, TaskStatusInProgress,
		TaskStatusInReview, TaskStatusPendingApproval, TaskStatusRevisionRequired, TaskStatusCompleted,
		TaskStatusFailed, TaskStatusCancelled, TaskStatusBlocked,
	}
	for _, s := range allStatuses {
		if s.CanTransitionTo(s) {
			t.Errorf("self-transition should be invalid: %s -> %s", s, s)
		}
	}

	// Unknown/terminal: completed and cancelled have no outgoing transitions
	if TaskStatusCompleted.CanTransitionTo(TaskStatusPending) {
		t.Error("completed should be terminal")
	}
	if TaskStatusCancelled.CanTransitionTo(TaskStatusPending) {
		t.Error("cancelled should be terminal")
	}

	// Unknown status should not transition
	unknown := TaskStatus("unknown_status")
	if unknown.CanTransitionTo(TaskStatusPending) {
		t.Error("unknown status should not allow transitions")
	}
}

func TestApprovalStatusCanDecide(t *testing.T) {
	// Only pending approvals can be decided
	if ApprovalStatusPending == "" {
		t.Error("pending status should not be empty")
	}
	if ApprovalStatusApproved == ApprovalStatusRejected {
		t.Error("approved and rejected should be different")
	}
	// Ensure all three values are distinct
	statuses := map[ApprovalStatus]bool{
		ApprovalStatusPending:  true,
		ApprovalStatusApproved: true,
		ApprovalStatusRejected: true,
	}
	if len(statuses) != 3 {
		t.Error("expected 3 distinct approval statuses")
	}
}
