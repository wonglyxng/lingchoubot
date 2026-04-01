package model

import "testing"

func TestTaskStatusCanTransitionTo(t *testing.T) {
	tests := []struct {
		from    TaskStatus
		to      TaskStatus
		allowed bool
	}{
		// pending
		{TaskStatusPending, TaskStatusAssigned, true},
		{TaskStatusPending, TaskStatusCancelled, true},
		{TaskStatusPending, TaskStatusInProgress, false},
		{TaskStatusPending, TaskStatusCompleted, false},

		// assigned
		{TaskStatusAssigned, TaskStatusInProgress, true},
		{TaskStatusAssigned, TaskStatusCancelled, true},
		{TaskStatusAssigned, TaskStatusCompleted, false},
		{TaskStatusAssigned, TaskStatusPending, false},

		// in_progress
		{TaskStatusInProgress, TaskStatusInReview, true},
		{TaskStatusInProgress, TaskStatusCompleted, true},
		{TaskStatusInProgress, TaskStatusFailed, true},
		{TaskStatusInProgress, TaskStatusBlocked, true},
		{TaskStatusInProgress, TaskStatusCancelled, true},
		{TaskStatusInProgress, TaskStatusPending, false},
		{TaskStatusInProgress, TaskStatusAssigned, false},

		// in_review
		{TaskStatusInReview, TaskStatusPendingApproval, true},
		{TaskStatusInReview, TaskStatusRevisionRequired, true},
		{TaskStatusInReview, TaskStatusFailed, true},
		{TaskStatusInReview, TaskStatusCompleted, false},
		{TaskStatusInReview, TaskStatusInProgress, false},
		{TaskStatusInReview, TaskStatusCancelled, false},
		{TaskStatusInReview, TaskStatusPending, false},

		// pending_approval
		{TaskStatusPendingApproval, TaskStatusCompleted, true},
		{TaskStatusPendingApproval, TaskStatusRevisionRequired, true},
		{TaskStatusPendingApproval, TaskStatusFailed, true},
		{TaskStatusPendingApproval, TaskStatusInProgress, false},
		{TaskStatusPendingApproval, TaskStatusPending, false},

		// revision_required
		{TaskStatusRevisionRequired, TaskStatusInProgress, true},
		{TaskStatusRevisionRequired, TaskStatusCancelled, true},
		{TaskStatusRevisionRequired, TaskStatusCompleted, false},

		// blocked
		{TaskStatusBlocked, TaskStatusInProgress, true},
		{TaskStatusBlocked, TaskStatusCancelled, true},
		{TaskStatusBlocked, TaskStatusCompleted, false},

		// failed
		{TaskStatusFailed, TaskStatusPending, true},
		{TaskStatusFailed, TaskStatusCancelled, true},
		{TaskStatusFailed, TaskStatusCompleted, false},

		// completed (terminal)
		{TaskStatusCompleted, TaskStatusPending, false},
		{TaskStatusCompleted, TaskStatusInProgress, false},

		// cancelled (terminal)
		{TaskStatusCancelled, TaskStatusPending, false},
		{TaskStatusCancelled, TaskStatusInProgress, false},
	}

	for _, tt := range tests {
		name := string(tt.from) + " -> " + string(tt.to)
		t.Run(name, func(t *testing.T) {
			got := tt.from.CanTransitionTo(tt.to)
			if got != tt.allowed {
				t.Errorf("CanTransitionTo(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.allowed)
			}
		})
	}
}

func TestTaskStatusConstants(t *testing.T) {
	statuses := []TaskStatus{
		TaskStatusPending,
		TaskStatusAssigned,
		TaskStatusInProgress,
		TaskStatusInReview,
		TaskStatusPendingApproval,
		TaskStatusRevisionRequired,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusCancelled,
		TaskStatusBlocked,
	}
	seen := make(map[TaskStatus]bool)
	for _, s := range statuses {
		if s == "" {
			t.Error("empty task status constant")
		}
		if seen[s] {
			t.Errorf("duplicate task status: %s", s)
		}
		seen[s] = true
	}
}
