export interface Project {
  id: string;
  name: string;
  description: string;
  status: string;
  owner_agent_id?: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Phase {
  id: string;
  project_id: string;
  name: string;
  description: string;
  status: string;
  sort_order: number;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Agent {
  id: string;
  name: string;
  role: string;
  role_code?: string;
  agent_type: string;
  specialization: string;
  description: string;
  reports_to?: string;
  status: string;
  capabilities: unknown;
  metadata: Record<string, unknown>;
  managed_roles?: unknown;
  allowed_tools?: unknown;
  risk_level?: string;
  created_at: string;
  updated_at: string;
}

export interface Task {
  id: string;
  project_id: string;
  phase_id?: string;
  parent_task_id?: string;
  title: string;
  description: string;
  status: string;
  priority: number;
  assignee_id?: string;
  input_context: unknown;
  output_summary: unknown;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Artifact {
  id: string;
  project_id: string;
  task_id?: string;
  name: string;
  artifact_type: string;
  description: string;
  created_by?: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ArtifactVersion {
  id: string;
  artifact_id: string;
  version: number;
  uri: string;
  content_type: string;
  size_bytes: number;
  checksum: string;
  change_summary: string;
  created_by?: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface ApprovalRequest {
  id: string;
  project_id: string;
  task_id?: string;
  artifact_id?: string;
  requested_by: string;
  approver_type: string;
  approver_id: string;
  title: string;
  description: string;
  status: string;
  decision_note: string;
  decided_at?: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ApprovalDecisionResult {
  id: string;
  status: string;
  task_status?: string;
  workflow_run_id?: string;
  workflow_resume_status: string;
  workflow_resume_message?: string;
  warnings?: string[];
}

export interface AuditLog {
  id: string;
  actor_type: string;
  actor_id: string;
  event_type: string;
  event_summary: string;
  target_type: string;
  target_id: string;
  before_state?: unknown;
  after_state?: unknown;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface ReviewReport {
  id: string;
  run_id?: string;
  task_id: string;
  reviewer_id: string;
  artifact_version_id?: string;
  verdict: string;
  summary: string;
  findings: unknown;
  recommendations: unknown;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface ReviewHardGateResult {
  key: string;
  passed: boolean;
  reason: string;
}

export interface ReviewScoreItemResult {
  key: string;
  name: string;
  weight: number;
  score: number;
  max_score: number;
  reason?: string;
}

export interface ReviewReworkBrief {
  attempt: number;
  failed_hard_gate_keys: string[];
  low_score_item_keys: string[];
  must_fix_items: string[];
  suggestions: string[];
  requires_clarification: boolean;
}

export interface ReviewScorecardMetadata {
  template_key?: string;
  task_category?: string;
  pass_threshold?: number;
  total_score?: number;
  hard_gate_results?: ReviewHardGateResult[];
  score_items?: ReviewScoreItemResult[];
  must_fix_items?: string[];
  suggestions?: string[];
  rework_brief?: ReviewReworkBrief;
}

export interface ApprovalScoreBreakdownItem {
  key: string;
  name: string;
  weight: number;
  score: number;
  max_score: number;
}

export interface ApprovalScoreSummary {
  template_key?: string;
  pass_threshold?: number;
  total_score?: number;
  hard_gate_passed_count?: number;
  hard_gate_total_count?: number;
  score_breakdown_summary?: ApprovalScoreBreakdownItem[];
  must_fix_items?: string[];
}

export interface TaskContract {
  id: string;
  task_id: string;
  version: number;
  scope: string;
  non_goals: string;
  completion_definition: string;
  acceptance_criteria: unknown;
  constraints: unknown;
  metadata: Record<string, unknown>;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface TaskAssignment {
  id: string;
  task_id: string;
  agent_id: string;
  role: string;
  status: string;
  assigned_by?: string;
  started_at?: string;
  completed_at?: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface HandoffSnapshot {
  id: string;
  task_id: string;
  agent_id: string;
  summary: string;
  completed_items: unknown;
  pending_items: unknown;
  risks: unknown;
  next_steps: unknown;
  artifact_refs: unknown;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface ToolCall {
  id: string;
  task_id?: string;
  agent_id: string;
  tool_name: string;
  action: string;
  input: unknown;
  output: unknown;
  status: string;
  error_message?: string;
  denied_reason?: string;
  duration_ms: number;
  metadata: Record<string, unknown>;
  created_at: string;
  completed_at?: string;
}

export interface WorkflowRun {
  id: string;
  project_id: string;
  status: string;
  summary: string;
  error: string;
  metadata: Record<string, unknown>;
  steps?: WorkflowStep[];
  started_at?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface WorkflowStep {
  id: string;
  run_id: string;
  name: string;
  agent_role: string;
  agent_id?: string;
  task_id?: string;
  phase_id?: string;
  status: string;
  summary: string;
  error: string;
  sort_order: number;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

export interface LLMProvider {
  id: string;
  key: string;
  name: string;
  base_url: string;
  api_key?: string;
  is_builtin: boolean;
  is_enabled: boolean;
  sort_order: number;
  metadata: Record<string, unknown>;
  models?: LLMModel[];
  created_at: string;
  updated_at: string;
}

export interface LLMModel {
  id: string;
  provider_id: string;
  model_id: string;
  name: string;
  is_default: boolean;
  sort_order: number;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ListResponse<T> {
  items: T[];
  total: number;
}
