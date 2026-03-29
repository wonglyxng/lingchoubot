import type {
  Project, Phase, Agent, Task, Artifact, ArtifactVersion,
  ApprovalRequest, AuditLog, ListResponse,
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

export const api = {
  projects: {
    list: (limit = 50, offset = 0) =>
      get<ListResponse<Project>>(`/api/v1/projects?limit=${limit}&offset=${offset}`),
    get: (id: string) => get<Project>(`/api/v1/projects/${id}`),
  },

  phases: {
    listByProject: (projectId: string) =>
      get<Phase[]>(`/api/v1/projects/${projectId}/phases`),
  },

  agents: {
    list: (limit = 100, offset = 0) =>
      get<ListResponse<Agent>>(`/api/v1/agents?limit=${limit}&offset=${offset}`),
    orgTree: (rootId?: string) =>
      get<Agent[]>(`/api/v1/agents/org-tree${rootId ? `?root_id=${rootId}` : ""}`),
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
};
