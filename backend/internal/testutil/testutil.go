// Package testutil provides in-memory fake repositories and a wired Engine builder
// for integration tests. All fakes are safe for concurrent single-goroutine test usage.
package testutil

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/orchestrator"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/runtime"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

// ---------- generic helpers ----------

func genID(prefix string, seq int) string {
	return fmt.Sprintf("%s-%04d", prefix, seq)
}

// ---------- FakeProjectRepo ----------

type FakeProjectRepo struct {
	mu       sync.Mutex
	projects map[string]*model.Project
	seq      int
}

func NewFakeProjectRepo() *FakeProjectRepo {
	return &FakeProjectRepo{projects: map[string]*model.Project{}}
}

func (r *FakeProjectRepo) Create(_ context.Context, p *model.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if p.ID == "" {
		p.ID = genID("proj", r.seq)
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	cp := *p
	r.projects[cp.ID] = &cp
	return nil
}
func (r *FakeProjectRepo) GetByID(_ context.Context, id string) (*model.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.projects[id]
	if p == nil {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}
func (r *FakeProjectRepo) List(_ context.Context, limit, offset int) ([]*model.Project, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]*model.Project, 0, len(r.projects))
	for _, p := range r.projects {
		cp := *p
		all = append(all, &cp)
	}
	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}
func (r *FakeProjectRepo) Update(_ context.Context, p *model.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	cp.UpdatedAt = time.Now()
	r.projects[cp.ID] = &cp
	return nil
}
func (r *FakeProjectRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.projects, id)
	return nil
}

// ---------- FakePhaseRepo ----------

type FakePhaseRepo struct {
	mu     sync.Mutex
	phases map[string]*model.Phase
	seq    int
}

func NewFakePhaseRepo() *FakePhaseRepo {
	return &FakePhaseRepo{phases: map[string]*model.Phase{}}
}

func (r *FakePhaseRepo) Create(_ context.Context, p *model.Phase) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if p.ID == "" {
		p.ID = genID("phase", r.seq)
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	cp := *p
	r.phases[cp.ID] = &cp
	return nil
}
func (r *FakePhaseRepo) GetByID(_ context.Context, id string) (*model.Phase, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.phases[id]
	if p == nil {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}
func (r *FakePhaseRepo) ListByProject(_ context.Context, projectID string) ([]*model.Phase, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.Phase
	for _, p := range r.phases {
		if p.ProjectID == projectID {
			cp := *p
			result = append(result, &cp)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SortOrder == result[j].SortOrder {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].SortOrder < result[j].SortOrder
	})
	return result, nil
}
func (r *FakePhaseRepo) Update(_ context.Context, p *model.Phase) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	cp.UpdatedAt = time.Now()
	r.phases[cp.ID] = &cp
	return nil
}
func (r *FakePhaseRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.phases, id)
	return nil
}

// ---------- FakeAgentRepo ----------

type FakeAgentRepo struct {
	mu     sync.Mutex
	agents map[string]*model.Agent
	seq    int
}

func NewFakeAgentRepo() *FakeAgentRepo {
	return &FakeAgentRepo{agents: map[string]*model.Agent{}}
}

func (r *FakeAgentRepo) Create(_ context.Context, a *model.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if a.ID == "" {
		a.ID = genID("agent", r.seq)
	}
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	cp := *a
	r.agents[cp.ID] = &cp
	return nil
}
func (r *FakeAgentRepo) GetByID(_ context.Context, id string) (*model.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a := r.agents[id]
	if a == nil {
		return nil, nil
	}
	cp := *a
	return &cp, nil
}
func (r *FakeAgentRepo) GetByRoleCode(_ context.Context, roleCode model.RoleCode) (*model.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.agents {
		if a.RoleCode == roleCode {
			cp := *a
			return &cp, nil
		}
	}
	return nil, nil
}
func (r *FakeAgentRepo) List(_ context.Context, limit, offset int) ([]*model.Agent, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]*model.Agent, 0, len(r.agents))
	for _, a := range r.agents {
		cp := *a
		all = append(all, &cp)
	}
	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}
func (r *FakeAgentRepo) Update(_ context.Context, a *model.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *a
	cp.UpdatedAt = time.Now()
	r.agents[cp.ID] = &cp
	return nil
}
func (r *FakeAgentRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, id)
	return nil
}
func (r *FakeAgentRepo) GetSubordinates(_ context.Context, _ string) ([]*model.Agent, error) {
	return nil, nil
}
func (r *FakeAgentRepo) GetOrgTree(_ context.Context, _ string) ([]*model.Agent, error) {
	return nil, nil
}
func (r *FakeAgentRepo) FindByRoleAndSpec(_ context.Context, role model.AgentRole, spec model.AgentSpecialization) (*model.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.agents {
		if a.Role == role && a.Specialization == spec && a.Status == model.AgentStatusActive {
			cp := *a
			return &cp, nil
		}
	}
	return nil, nil
}
func (r *FakeAgentRepo) FindByRoleCode(_ context.Context, roleCode model.RoleCode) (*model.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.agents {
		if a.RoleCode == roleCode && a.Status == model.AgentStatusActive {
			cp := *a
			return &cp, nil
		}
	}
	return nil, nil
}

// ---------- FakeTaskRepo ----------

type FakeTaskRepo struct {
	mu    sync.Mutex
	tasks map[string]*model.Task
	seq   int
}

func NewFakeTaskRepo() *FakeTaskRepo {
	return &FakeTaskRepo{tasks: map[string]*model.Task{}}
}

func (r *FakeTaskRepo) Create(_ context.Context, t *model.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if t.ID == "" {
		t.ID = genID("task", r.seq)
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	cp := *t
	r.tasks[cp.ID] = &cp
	return nil
}
func (r *FakeTaskRepo) GetByID(_ context.Context, id string) (*model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.tasks[id]
	if t == nil {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}
func (r *FakeTaskRepo) List(_ context.Context, p repository.TaskListParams) ([]*model.Task, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.Task
	for _, t := range r.tasks {
		if p.PhaseID != "" && (t.PhaseID == nil || *t.PhaseID != p.PhaseID) {
			continue
		}
		if p.ProjectID != "" && t.ProjectID != p.ProjectID {
			continue
		}
		cp := *t
		result = append(result, &cp)
	}
	total := len(result)
	offset := p.Offset
	limit := p.Limit
	if limit == 0 {
		limit = 100
	}
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return result[offset:end], total, nil
}
func (r *FakeTaskRepo) Update(_ context.Context, t *model.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *t
	cp.UpdatedAt = time.Now()
	r.tasks[cp.ID] = &cp
	return nil
}
func (r *FakeTaskRepo) UpdateStatus(_ context.Context, id string, status model.TaskStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t := r.tasks[id]; t != nil {
		t.Status = status
		t.UpdatedAt = time.Now()
	}
	return nil
}
func (r *FakeTaskRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tasks, id)
	return nil
}

// ---------- FakeTaskContractRepo ----------

type FakeTaskContractRepo struct {
	mu        sync.Mutex
	contracts map[string]*model.TaskContract
	seq       int
}

func NewFakeTaskContractRepo() *FakeTaskContractRepo {
	return &FakeTaskContractRepo{contracts: map[string]*model.TaskContract{}}
}

func (r *FakeTaskContractRepo) Create(_ context.Context, c *model.TaskContract) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if c.ID == "" {
		c.ID = genID("contract", r.seq)
	}
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	cp := *c
	r.contracts[cp.ID] = &cp
	return nil
}
func (r *FakeTaskContractRepo) GetByID(_ context.Context, id string) (*model.TaskContract, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.contracts[id]
	if c == nil {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}
func (r *FakeTaskContractRepo) GetLatestByTaskID(_ context.Context, taskID string) (*model.TaskContract, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var latest *model.TaskContract
	for _, c := range r.contracts {
		if c.TaskID == taskID {
			if latest == nil || c.Version > latest.Version {
				cp := *c
				latest = &cp
			}
		}
	}
	return latest, nil
}
func (r *FakeTaskContractRepo) ListByTaskID(_ context.Context, taskID string) ([]*model.TaskContract, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.TaskContract
	for _, c := range r.contracts {
		if c.TaskID == taskID {
			cp := *c
			result = append(result, &cp)
		}
	}
	return result, nil
}
func (r *FakeTaskContractRepo) NextVersion(_ context.Context, taskID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	maxVer := 0
	for _, c := range r.contracts {
		if c.TaskID == taskID && c.Version > maxVer {
			maxVer = c.Version
		}
	}
	return maxVer + 1, nil
}
func (r *FakeTaskContractRepo) Update(_ context.Context, c *model.TaskContract) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *c
	cp.UpdatedAt = time.Now()
	r.contracts[cp.ID] = &cp
	return nil
}

// ---------- FakeTaskAssignmentRepo ----------

type FakeTaskAssignmentRepo struct {
	mu          sync.Mutex
	assignments map[string]*model.TaskAssignment
	seq         int
}

func NewFakeTaskAssignmentRepo() *FakeTaskAssignmentRepo {
	return &FakeTaskAssignmentRepo{assignments: map[string]*model.TaskAssignment{}}
}

func (r *FakeTaskAssignmentRepo) Create(_ context.Context, a *model.TaskAssignment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if a.ID == "" {
		a.ID = genID("assign", r.seq)
	}
	now := time.Now()
	a.CreatedAt = now
	cp := *a
	r.assignments[cp.ID] = &cp
	return nil
}
func (r *FakeTaskAssignmentRepo) GetByID(_ context.Context, id string) (*model.TaskAssignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a := r.assignments[id]
	if a == nil {
		return nil, nil
	}
	cp := *a
	return &cp, nil
}
func (r *FakeTaskAssignmentRepo) List(_ context.Context, p repository.AssignmentListParams) ([]*model.TaskAssignment, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.TaskAssignment
	for _, a := range r.assignments {
		if p.TaskID != "" && a.TaskID != p.TaskID {
			continue
		}
		cp := *a
		result = append(result, &cp)
	}
	return result, len(result), nil
}
func (r *FakeTaskAssignmentRepo) UpdateStatus(_ context.Context, id string, status model.AssignmentStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a := r.assignments[id]; a != nil {
		a.Status = status
	}
	return nil
}

// ---------- FakeArtifactRepo ----------

type FakeArtifactRepo struct {
	mu        sync.Mutex
	artifacts map[string]*model.Artifact
	seq       int
}

func NewFakeArtifactRepo() *FakeArtifactRepo {
	return &FakeArtifactRepo{artifacts: map[string]*model.Artifact{}}
}

func (r *FakeArtifactRepo) Create(_ context.Context, a *model.Artifact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if a.ID == "" {
		a.ID = genID("artifact", r.seq)
	}
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	cp := *a
	r.artifacts[cp.ID] = &cp
	return nil
}

func (r *FakeArtifactRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.artifacts, id)
	return nil
}

func (r *FakeArtifactRepo) GetByID(_ context.Context, id string) (*model.Artifact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a := r.artifacts[id]
	if a == nil {
		return nil, nil
	}
	cp := *a
	return &cp, nil
}
func (r *FakeArtifactRepo) List(_ context.Context, p repository.ArtifactListParams) ([]*model.Artifact, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.Artifact
	for _, a := range r.artifacts {
		if p.ProjectID != "" && a.ProjectID != p.ProjectID {
			continue
		}
		cp := *a
		result = append(result, &cp)
	}
	return result, len(result), nil
}

// CountByProject returns the number of artifacts for a given project (test helper).
func (r *FakeArtifactRepo) CountByProject(projectID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, a := range r.artifacts {
		if a.ProjectID == projectID {
			n++
		}
	}
	return n
}

// ---------- FakeArtifactVersionRepo ----------

type FakeArtifactVersionRepo struct {
	mu       sync.Mutex
	versions map[string]*model.ArtifactVersion
	seq      int
}

func NewFakeArtifactVersionRepo() *FakeArtifactVersionRepo {
	return &FakeArtifactVersionRepo{versions: map[string]*model.ArtifactVersion{}}
}

func (r *FakeArtifactVersionRepo) Create(_ context.Context, v *model.ArtifactVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if v.ID == "" {
		v.ID = genID("artver", r.seq)
	}
	v.CreatedAt = time.Now()
	cp := *v
	r.versions[cp.ID] = &cp
	return nil
}
func (r *FakeArtifactVersionRepo) GetByID(_ context.Context, id string) (*model.ArtifactVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := r.versions[id]
	if v == nil {
		return nil, nil
	}
	cp := *v
	return &cp, nil
}
func (r *FakeArtifactVersionRepo) ListByArtifact(_ context.Context, artifactID string) ([]*model.ArtifactVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.ArtifactVersion
	for _, v := range r.versions {
		if v.ArtifactID == artifactID {
			cp := *v
			result = append(result, &cp)
		}
	}
	return result, nil
}
func (r *FakeArtifactVersionRepo) NextVersion(_ context.Context, artifactID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	maxVer := 0
	for _, v := range r.versions {
		if v.ArtifactID == artifactID && v.Version > maxVer {
			maxVer = v.Version
		}
	}
	return maxVer + 1, nil
}

// TotalCount returns total number of versions (test helper).
func (r *FakeArtifactVersionRepo) TotalCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.versions)
}

// ---------- FakeHandoffSnapshotRepo ----------

type FakeHandoffSnapshotRepo struct {
	mu      sync.Mutex
	entries map[string]*model.HandoffSnapshot
	seq     int
}

func NewFakeHandoffSnapshotRepo() *FakeHandoffSnapshotRepo {
	return &FakeHandoffSnapshotRepo{entries: map[string]*model.HandoffSnapshot{}}
}

func (r *FakeHandoffSnapshotRepo) Create(_ context.Context, s *model.HandoffSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if s.ID == "" {
		s.ID = genID("handoff", r.seq)
	}
	s.CreatedAt = time.Now()
	cp := *s
	r.entries[cp.ID] = &cp
	return nil
}
func (r *FakeHandoffSnapshotRepo) GetByID(_ context.Context, id string) (*model.HandoffSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.entries[id]
	if s == nil {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}
func (r *FakeHandoffSnapshotRepo) List(_ context.Context, p repository.HandoffListParams) ([]*model.HandoffSnapshot, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.HandoffSnapshot
	for _, s := range r.entries {
		if p.TaskID != "" && s.TaskID != p.TaskID {
			continue
		}
		cp := *s
		result = append(result, &cp)
	}
	return result, len(result), nil
}
func (r *FakeHandoffSnapshotRepo) GetLatestByTaskID(_ context.Context, taskID string) (*model.HandoffSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var latest *model.HandoffSnapshot
	for _, s := range r.entries {
		if s.TaskID == taskID {
			if latest == nil || s.CreatedAt.After(latest.CreatedAt) {
				cp := *s
				latest = &cp
			}
		}
	}
	return latest, nil
}

// ---------- FakeReviewReportRepo ----------

type FakeReviewReportRepo struct {
	mu      sync.Mutex
	reports map[string]*model.ReviewReport
	seq     int
}

func NewFakeReviewReportRepo() *FakeReviewReportRepo {
	return &FakeReviewReportRepo{reports: map[string]*model.ReviewReport{}}
}

func (r *FakeReviewReportRepo) Create(_ context.Context, rr *model.ReviewReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if rr.ID == "" {
		rr.ID = genID("review", r.seq)
	}
	rr.CreatedAt = time.Now()
	cp := *rr
	r.reports[cp.ID] = &cp
	return nil
}
func (r *FakeReviewReportRepo) GetByID(_ context.Context, id string) (*model.ReviewReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rr := r.reports[id]
	if rr == nil {
		return nil, nil
	}
	cp := *rr
	return &cp, nil
}
func (r *FakeReviewReportRepo) List(_ context.Context, p repository.ReviewListParams) ([]*model.ReviewReport, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.ReviewReport
	for _, rr := range r.reports {
		if p.RunID != "" {
			if rr.RunID == nil || *rr.RunID != p.RunID {
				continue
			}
		}
		if p.TaskID != "" && rr.TaskID != p.TaskID {
			continue
		}
		if p.ReviewerID != "" && rr.ReviewerID != p.ReviewerID {
			continue
		}
		if p.Verdict != "" && string(rr.Verdict) != p.Verdict {
			continue
		}
		cp := *rr
		result = append(result, &cp)
	}
	return result, len(result), nil
}

// TotalCount returns total number of reviews (test helper).
func (r *FakeReviewReportRepo) TotalCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.reports)
}

// ---------- FakeApprovalRepo ----------

type FakeApprovalRepo struct {
	mu        sync.Mutex
	approvals map[string]*model.ApprovalRequest
	seq       int
}

func NewFakeApprovalRepo() *FakeApprovalRepo {
	return &FakeApprovalRepo{approvals: map[string]*model.ApprovalRequest{}}
}

func (r *FakeApprovalRepo) Create(_ context.Context, a *model.ApprovalRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if a.ID == "" {
		a.ID = genID("approval", r.seq)
	}
	now := time.Now()
	a.CreatedAt = now
	cp := *a
	r.approvals[cp.ID] = &cp
	return nil
}
func (r *FakeApprovalRepo) GetByID(_ context.Context, id string) (*model.ApprovalRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a := r.approvals[id]
	if a == nil {
		return nil, nil
	}
	cp := *a
	return &cp, nil
}
func (r *FakeApprovalRepo) List(_ context.Context, p repository.ApprovalListParams) ([]*model.ApprovalRequest, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.ApprovalRequest
	for _, a := range r.approvals {
		cp := *a
		result = append(result, &cp)
	}
	return result, len(result), nil
}
func (r *FakeApprovalRepo) Decide(_ context.Context, id string, status model.ApprovalStatus, note string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a := r.approvals[id]; a != nil {
		a.Status = status
		a.DecisionNote = note
		now := time.Now()
		a.DecidedAt = &now
	}
	return nil
}

// ---------- FakeAuditRepo ----------

type FakeAuditRepo struct {
	mu      sync.Mutex
	entries []*model.AuditLog
	seq     int
}

func NewFakeAuditRepo() *FakeAuditRepo {
	return &FakeAuditRepo{}
}

func (r *FakeAuditRepo) Create(_ context.Context, a *model.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if a.ID == "" {
		a.ID = genID("audit", r.seq)
	}
	a.CreatedAt = time.Now()
	cp := *a
	r.entries = append(r.entries, &cp)
	return nil
}
func (r *FakeAuditRepo) List(_ context.Context, _ repository.AuditListParams) ([]*model.AuditLog, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]*model.AuditLog, len(r.entries))
	for i, e := range r.entries {
		c := *e
		cp[i] = &c
	}
	return cp, len(cp), nil
}
func (r *FakeAuditRepo) ProjectTimeline(_ context.Context, _ string, _, _ int) ([]*model.AuditLog, int, error) {
	return nil, 0, nil
}
func (r *FakeAuditRepo) TaskTimeline(_ context.Context, _ string, _, _ int) ([]*model.AuditLog, int, error) {
	return nil, 0, nil
}

// Entries returns a copy of the audit log entries (test helper).
func (r *FakeAuditRepo) Entries() []*model.AuditLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]*model.AuditLog, len(r.entries))
	for i, e := range r.entries {
		c := *e
		cp[i] = &c
	}
	return cp
}

// ---------- FakeWorkflowRunRepo ----------

type FakeWorkflowRunRepo struct {
	mu   sync.Mutex
	runs map[string]*model.WorkflowRun
	seq  int
}

func NewFakeWorkflowRunRepo() *FakeWorkflowRunRepo {
	return &FakeWorkflowRunRepo{runs: map[string]*model.WorkflowRun{}}
}

func (r *FakeWorkflowRunRepo) Create(_ context.Context, run *model.WorkflowRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if run.ID == "" {
		run.ID = genID("run", r.seq)
	}
	cp := *run
	r.runs[cp.ID] = &cp
	return nil
}
func (r *FakeWorkflowRunRepo) GetByID(_ context.Context, id string) (*model.WorkflowRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run := r.runs[id]
	if run == nil {
		return nil, nil
	}
	cp := *run
	return &cp, nil
}
func (r *FakeWorkflowRunRepo) UpdateStatus(_ context.Context, run *model.WorkflowRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *run
	r.runs[cp.ID] = &cp
	return nil
}
func (r *FakeWorkflowRunRepo) List(_ context.Context, _ repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]*model.WorkflowRun, 0, len(r.runs))
	for _, run := range r.runs {
		cp := *run
		all = append(all, &cp)
	}
	return all, len(all), nil
}

// ---------- FakeWorkflowStepRepo ----------

type FakeWorkflowStepRepo struct {
	mu    sync.Mutex
	steps map[string]*model.WorkflowStep
	seq   int
}

func NewFakeWorkflowStepRepo() *FakeWorkflowStepRepo {
	return &FakeWorkflowStepRepo{steps: map[string]*model.WorkflowStep{}}
}

func (r *FakeWorkflowStepRepo) Create(_ context.Context, step *model.WorkflowStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if step.ID == "" {
		step.ID = genID("step", r.seq)
	}
	step.CreatedAt = time.Now()
	cp := *step
	r.steps[cp.ID] = &cp
	return nil
}
func (r *FakeWorkflowStepRepo) UpdateStatus(_ context.Context, step *model.WorkflowStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *step
	r.steps[cp.ID] = &cp
	return nil
}
func (r *FakeWorkflowStepRepo) ListByRunID(_ context.Context, runID string) ([]*model.WorkflowStep, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.WorkflowStep
	for _, s := range r.steps {
		if s.RunID == runID {
			cp := *s
			result = append(result, &cp)
		}
	}
	return result, nil
}

// StepsForRun returns steps for a run (test helper).
func (r *FakeWorkflowStepRepo) StepsForRun(runID string) []*model.WorkflowStep {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.WorkflowStep
	for _, s := range r.steps {
		if s.RunID == runID {
			cp := *s
			result = append(result, &cp)
		}
	}
	return result
}

// ---------- FakeToolCallRepo ----------

type FakeToolCallRepo struct {
	mu    sync.Mutex
	calls map[string]*model.ToolCall
	seq   int
}

func NewFakeToolCallRepo() *FakeToolCallRepo {
	return &FakeToolCallRepo{calls: map[string]*model.ToolCall{}}
}

func (r *FakeToolCallRepo) Create(_ context.Context, tc *model.ToolCall) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if tc.ID == "" {
		tc.ID = genID("toolcall", r.seq)
	}
	tc.CreatedAt = time.Now()
	cp := *tc
	r.calls[cp.ID] = &cp
	return nil
}
func (r *FakeToolCallRepo) GetByID(_ context.Context, id string) (*model.ToolCall, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tc := r.calls[id]
	if tc == nil {
		return nil, nil
	}
	cp := *tc
	return &cp, nil
}
func (r *FakeToolCallRepo) List(_ context.Context, p repository.ToolCallListParams) ([]*model.ToolCall, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.ToolCall
	for _, tc := range r.calls {
		cp := *tc
		result = append(result, &cp)
	}
	return result, len(result), nil
}
func (r *FakeToolCallRepo) Complete(_ context.Context, id string, status model.ToolCallStatus, output model.JSON, errMsg string, durationMs int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tc := r.calls[id]; tc != nil {
		tc.Status = status
		tc.Output = output
		tc.ErrorMessage = errMsg
		tc.DurationMs = durationMs
	}
	return nil
}
func (r *FakeToolCallRepo) UpdateDenied(_ context.Context, id string, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tc := r.calls[id]; tc != nil {
		tc.Status = model.ToolCallStatusDenied
		tc.DeniedReason = reason
	}
	return nil
}

// ---------- Fixture: fully wired Engine ----------

// Fixture holds all fake repos and wired services for integration tests.
type Fixture struct {
	// Repos
	ProjectRepo         *FakeProjectRepo
	PhaseRepo           *FakePhaseRepo
	AgentRepo           *FakeAgentRepo
	TaskRepo            *FakeTaskRepo
	ContractRepo        *FakeTaskContractRepo
	AssignmentRepo      *FakeTaskAssignmentRepo
	ArtifactRepo        *FakeArtifactRepo
	ArtifactVersionRepo *FakeArtifactVersionRepo
	HandoffRepo         *FakeHandoffSnapshotRepo
	ReviewRepo          *FakeReviewReportRepo
	ApprovalRepo        *FakeApprovalRepo
	AuditRepo           *FakeAuditRepo
	WorkflowRunRepo     *FakeWorkflowRunRepo
	WorkflowStepRepo    *FakeWorkflowStepRepo
	ToolCallRepo        *FakeToolCallRepo

	// Services
	AuditSvc      *service.AuditService
	ProjectSvc    *service.ProjectService
	PhaseSvc      *service.PhaseService
	AgentSvc      *service.AgentService
	TaskSvc       *service.TaskService
	ContractSvc   *service.TaskContractService
	AssignmentSvc *service.TaskAssignmentService
	ArtifactSvc   *service.ArtifactService
	HandoffSvc    *service.HandoffSnapshotService
	ReviewSvc     *service.ReviewReportService
	ApprovalSvc   *service.ApprovalRequestService
	ToolCallSvc   *service.ToolCallService
	WorkflowSvc   *service.WorkflowService

	// Engine
	Engine   *orchestrator.Engine
	Registry *runtime.Registry
	Logger   *slog.Logger
}

// NewFixture creates a fully wired Fixture with all fake repos, services, and Engine.
// The registry includes mock agents for all standard roles.
func NewFixture() *Fixture {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// repos
	projectRepo := NewFakeProjectRepo()
	phaseRepo := NewFakePhaseRepo()
	agentRepo := NewFakeAgentRepo()
	taskRepo := NewFakeTaskRepo()
	contractRepo := NewFakeTaskContractRepo()
	assignmentRepo := NewFakeTaskAssignmentRepo()
	artifactRepo := NewFakeArtifactRepo()
	artifactVerRepo := NewFakeArtifactVersionRepo()
	handoffRepo := NewFakeHandoffSnapshotRepo()
	reviewRepo := NewFakeReviewReportRepo()
	approvalRepo := NewFakeApprovalRepo()
	auditRepo := NewFakeAuditRepo()
	runRepo := NewFakeWorkflowRunRepo()
	stepRepo := NewFakeWorkflowStepRepo()
	toolCallRepo := NewFakeToolCallRepo()

	// services
	auditSvc := service.NewAuditService(auditRepo, logger)
	projectSvc := service.NewProjectService(projectRepo, auditSvc)
	phaseSvc := service.NewPhaseService(phaseRepo, projectSvc, auditSvc)
	agentSvc := service.NewAgentService(agentRepo, auditSvc)
	taskSvc := service.NewTaskService(taskRepo, auditSvc)
	contractSvc := service.NewTaskContractService(contractRepo, auditSvc)
	assignmentSvc := service.NewTaskAssignmentService(assignmentRepo, auditSvc)
	artifactSvc := service.NewArtifactService(artifactRepo, artifactVerRepo, auditSvc)
	handoffSvc := service.NewHandoffSnapshotService(handoffRepo, auditSvc)
	reviewSvc := service.NewReviewReportService(reviewRepo, taskSvc, auditSvc)
	approvalSvc := service.NewApprovalRequestService(approvalRepo, taskSvc, auditSvc)
	reviewSvc.SetApprovalService(approvalSvc)
	toolCallSvc := service.NewToolCallService(toolCallRepo, auditSvc)
	workflowSvc := service.NewWorkflowService(runRepo, stepRepo, auditSvc)

	// registry with deterministic test runners
	reg := runtime.NewRegistry()
	registerDeterministicTestRunners(reg)

	// orchestrator engine
	svc := &orchestrator.Services{
		Project:    projectSvc,
		Phase:      phaseSvc,
		Agent:      agentSvc,
		Task:       taskSvc,
		Contract:   contractSvc,
		Assignment: assignmentSvc,
		Artifact:   artifactSvc,
		Handoff:    handoffSvc,
		Review:     reviewSvc,
		Approval:   approvalSvc,
		Audit:      auditSvc,
	}
	engine := orchestrator.NewEngine(reg, svc, workflowSvc, logger)
	approvalSvc.SetWorkflowResumer(engine)

	return &Fixture{
		ProjectRepo:         projectRepo,
		PhaseRepo:           phaseRepo,
		AgentRepo:           agentRepo,
		TaskRepo:            taskRepo,
		ContractRepo:        contractRepo,
		AssignmentRepo:      assignmentRepo,
		ArtifactRepo:        artifactRepo,
		ArtifactVersionRepo: artifactVerRepo,
		HandoffRepo:         handoffRepo,
		ReviewRepo:          reviewRepo,
		ApprovalRepo:        approvalRepo,
		AuditRepo:           auditRepo,
		WorkflowRunRepo:     runRepo,
		WorkflowStepRepo:    stepRepo,
		ToolCallRepo:        toolCallRepo,

		AuditSvc:      auditSvc,
		ProjectSvc:    projectSvc,
		PhaseSvc:      phaseSvc,
		AgentSvc:      agentSvc,
		TaskSvc:       taskSvc,
		ContractSvc:   contractSvc,
		AssignmentSvc: assignmentSvc,
		ArtifactSvc:   artifactSvc,
		HandoffSvc:    handoffSvc,
		ReviewSvc:     reviewSvc,
		ApprovalSvc:   approvalSvc,
		ToolCallSvc:   toolCallSvc,
		WorkflowSvc:   workflowSvc,

		Engine:   engine,
		Registry: reg,
		Logger:   logger,
	}
}

// SeedStandardAgents creates the standard set of agents for a complete workflow.
func (f *Fixture) SeedStandardAgents(ctx context.Context) error {
	byRoleCode := make(map[model.RoleCode]string)
	for _, spec := range service.BaselineAgentSpecs() {
		agent := spec.Agent
		if spec.ReportsToRoleCode != "" {
			if parentID, ok := byRoleCode[spec.ReportsToRoleCode]; ok {
				agent.ReportsTo = &parentID
			}
		}
		if err := f.AgentSvc.Create(ctx, &agent); err != nil {
			return fmt.Errorf("seed agent %s: %w", agent.Name, err)
		}
		byRoleCode[agent.RoleCode] = agent.ID
	}
	return nil
}
