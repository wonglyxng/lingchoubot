import type {
  Project, Phase, Agent, Task, Artifact, ArtifactVersion,
  ApprovalRequest, AuditLog, WorkflowRun, WorkflowStep, ListResponse,
} from "./types";

const BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface APIResponse<T> {
  success: boolean;
  data: T;
  error: { code: string; message: string } | null;
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`, { cache: "no-store" });
  const body: APIResponse<T> = await res.json();
  if (!body.success) {
    throw new Error(body.error?.message || "API error");
  }
  // 后端空列表可能返回 items:null，在此统一修正为空数组
  const data = body.data as Record<string, unknown>;
  if (data && "items" in data && data.items == null) {
    data.items = [];
  }
  return body.data;
}

async function post<T>(path: string, data: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  const body: APIResponse<T> = await res.json();
  if (!body.success) {
    throw new Error(body.error?.message || "API error");
  }
  return body.data;
}

async function put<T>(path: string, data: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  const body: APIResponse<T> = await res.json();
  if (!body.success) {
    throw new Error(body.error?.message || "API error");
  }
  return body.data;
}

export const api = {
  projects: {
    list: (limit = 50, offset = 0) =>
      get<ListResponse<Project>>(`/api/v1/projects?limit=${limit}&offset=${offset}`),
    get: (id: string) => get<Project>(`/api/v1/projects/${id}`),
    create: (data: Partial<Project>) => post<Project>("/api/v1/projects", data),
    update: (id: string, data: Partial<Project>) => put<Project>(`/api/v1/projects/${id}`, data),
  },

  phases: {
    listByProject: (projectId: string) =>
      get<Phase[]>(`/api/v1/projects/${projectId}/phases`),
    create: (data: Partial<Phase>) => post<Phase>("/api/v1/phases", data),
    update: (id: string, data: Partial<Phase>) => put<Phase>(`/api/v1/phases/${id}`, data),
  },

  agents: {
    list: (limit = 100, offset = 0) =>
      get<ListResponse<Agent>>(`/api/v1/agents?limit=${limit}&offset=${offset}`),
    orgTree: (rootId?: string) =>
      get<Agent[]>(`/api/v1/agents/org-tree${rootId ? `?root_id=${rootId}` : ""}`),
    create: (data: Partial<Agent>) => post<Agent>("/api/v1/agents", data),
    update: (id: string, data: Partial<Agent>) => put<Agent>(`/api/v1/agents/${id}`, data),
  },

  tasks: {
    list: (params?: { project_id?: string; phase_id?: string; status?: string; limit?: number; offset?: number }) => {
      const q = new URLSearchParams();
      if (params?.project_id) q.set("project_id", params.project_id);
      if (params?.phase_id) q.set("phase_id", params.phase_id);
      if (params?.status) q.set("status", params.status);
      q.set("limit", String(params?.limit || 100));
      q.set("offset", String(params?.offset || 0));
      return get<ListResponse<Task>>(`/api/v1/tasks?${q.toString()}`);
    },
    create: (data: Partial<Task>) => post<Task>("/api/v1/tasks", data),
    update: (id: string, data: Partial<Task>) => put<Task>(`/api/v1/tasks/${id}`, data),
  },

  artifacts: {
    list: (params?: { project_id?: string; task_id?: string; limit?: number; offset?: number }) => {
      const q = new URLSearchParams();
      if (params?.project_id) q.set("project_id", params.project_id);
      if (params?.task_id) q.set("task_id", params.task_id);
      q.set("limit", String(params?.limit || 50));
      q.set("offset", String(params?.offset || 0));
      return get<ListResponse<Artifact>>(`/api/v1/artifacts?${q.toString()}`);
    },
    versions: (artifactId: string) =>
      get<ArtifactVersion[]>(`/api/v1/artifacts/${artifactId}/versions`),
  },

  approvals: {
    list: (params?: { status?: string; limit?: number; offset?: number }) => {
      const q = new URLSearchParams();
      if (params?.status) q.set("status", params.status);
      q.set("limit", String(params?.limit || 50));
      q.set("offset", String(params?.offset || 0));
      return get<ListResponse<ApprovalRequest>>(`/api/v1/approvals?${q.toString()}`);
    },
    decide: (id: string, status: "approved" | "rejected", note: string) =>
      post<ApprovalRequest>(`/api/v1/approvals/${id}/decide`, { status, decision_note: note }),
  },

  audit: {
    list: (limit = 50, offset = 0) =>
      get<ListResponse<AuditLog>>(`/api/v1/audit-logs?limit=${limit}&offset=${offset}`),
    projectTimeline: (projectId: string, limit = 50, offset = 0) =>
      get<ListResponse<AuditLog>>(`/api/v1/projects/${projectId}/timeline?limit=${limit}&offset=${offset}`),
  },

  workflows: {
    list: (params?: { project_id?: string; status?: string; limit?: number; offset?: number }) => {
      const q = new URLSearchParams();
      if (params?.project_id) q.set("project_id", params.project_id);
      if (params?.status) q.set("status", params.status);
      q.set("limit", String(params?.limit || 50));
      q.set("offset", String(params?.offset || 0));
      return get<ListResponse<WorkflowRun>>(`/api/v1/orchestrator/runs?${q.toString()}`);
    },
    get: (id: string) => get<WorkflowRun>(`/api/v1/orchestrator/runs/${id}`),
    steps: (runId: string) => get<WorkflowStep[]>(`/api/v1/orchestrator/runs/${runId}/steps`),
    start: (projectId: string) => post<WorkflowRun>("/api/v1/orchestrator/runs", { project_id: projectId }),
  },
};
