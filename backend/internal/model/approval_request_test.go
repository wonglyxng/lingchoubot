package model

import "testing"

func TestApprovalStatusConstants(t *testing.T) {
	if ApprovalStatusPending != "pending" {
		t.Errorf("ApprovalStatusPending = %q", ApprovalStatusPending)
	}
	if ApprovalStatusApproved != "approved" {
		t.Errorf("ApprovalStatusApproved = %q", ApprovalStatusApproved)
	}
	if ApprovalStatusRejected != "rejected" {
		t.Errorf("ApprovalStatusRejected = %q", ApprovalStatusRejected)
	}
}

func TestApprovalStatusValues(t *testing.T) {
	all := []ApprovalStatus{
		ApprovalStatusPending,
		ApprovalStatusApproved,
		ApprovalStatusRejected,
	}
	seen := make(map[ApprovalStatus]bool)
	for _, s := range all {
		if s == "" {
			t.Error("empty approval status constant")
		}
		if seen[s] {
			t.Errorf("duplicate approval status: %s", s)
		}
		seen[s] = true
	}
}
