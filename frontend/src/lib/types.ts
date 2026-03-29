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
  description: string;
  reports_to?: string;
  status: string;
  capabilities: unknown;
  metadata: Record<string, unknown>;
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

export interface ToolCall {
  id: string;
  task_id?: string;
  agent_id: string;
  tool_name: string;
  input: unknown;
  output: unknown;
  status: string;
  error_message: string;
  duration_ms: number;
  metadata: Record<string, unknown>;
  created_at: string;
  completed_at?: string;
}

export interface ListResponse<T> {
  items: T[];
  total: number;
}
