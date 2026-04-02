import type {
  Project, Phase, Agent, Task, Artifact, ArtifactVersion,
  ApprovalRequest, AuditLog, WorkflowRun, ListResponse,
  ReviewReport, TaskContract, TaskAssignment, HandoffSnapshot, ToolCall,
  LLMProvider, LLMModel,
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
  if (body.data == null) {
    return {} as T;
  }
  // 后端空列表可能返回 items:null，在此统一修正为空数组
  const data = body.data as Record<string, unknown>;
  if (data && "items" in data && data.items == null) {
    data.items = [];
  }
  return body.data;
}

async function getList<T>(path: string): Promise<T[]> {
  const data = await get<ListResponse<T>>(path);
  return Array.isArray(data.items) ? data.items : [];
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

async function patch<T>(path: string, data: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  const body: APIResponse<T> = await res.json();
  if (!body.success) {
    throw new Error(body.error?.message || "API error");
  }
  return body.data;
}

async function del<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`, { method: "DELETE" });
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
    delete: (id: string) => del<null>(`/api/v1/projects/${id}`),
  },

  phases: {
    listByProject: (projectId: string) =>
      getList<Phase>(`/api/v1/projects/${projectId}/phases`),
    get: (id: string) => get<Phase>(`/api/v1/phases/${id}`),
    create: (data: Partial<Phase>) => post<Phase>("/api/v1/phases", data),
    update: (id: string, data: Partial<Phase>) => put<Phase>(`/api/v1/phases/${id}`, data),
    delete: (id: string) => del<null>(`/api/v1/phases/${id}`),
  },

  agents: {
    list: (limit = 100, offset = 0) =>
      get<ListResponse<Agent>>(`/api/v1/agents?limit=${limit}&offset=${offset}`),
    get: (id: string) => get<Agent>(`/api/v1/agents/${id}`),
    orgTree: (rootId?: string) =>
      getList<Agent>(`/api/v1/agents/org-tree${rootId ? `?root_id=${rootId}` : ""}`),
    create: (data: Partial<Agent>) => post<Agent>("/api/v1/agents", data),
    update: (id: string, data: Partial<Agent>) => put<Agent>(`/api/v1/agents/${id}`, data),
    delete: (id: string) => del<null>(`/api/v1/agents/${id}`),
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
    get: (id: string) => get<Task>(`/api/v1/tasks/${id}`),
    create: (data: Partial<Task>) => post<Task>("/api/v1/tasks", data),
    update: (id: string, data: Partial<Task>) => put<Task>(`/api/v1/tasks/${id}`, data),
    delete: (id: string) => del<null>(`/api/v1/tasks/${id}`),
    transition: (id: string, status: string) => patch<Task>(`/api/v1/tasks/${id}/status`, { status }),
  },

  taskContracts: {
    list: (taskId: string) =>
      getList<TaskContract>(`/api/v1/tasks/${taskId}/contracts`),
    latest: (taskId: string) =>
      get<TaskContract>(`/api/v1/tasks/${taskId}/contracts/latest`),
    get: (id: string) => get<TaskContract>(`/api/v1/task-contracts/${id}`),
    create: (data: Partial<TaskContract>) => post<TaskContract>("/api/v1/task-contracts", data),
    update: (id: string, data: Partial<TaskContract>) => put<TaskContract>(`/api/v1/task-contracts/${id}`, data),
  },

  taskAssignments: {
    list: (params?: { task_id?: string; agent_id?: string; limit?: number; offset?: number }) => {
      const q = new URLSearchParams();
      if (params?.task_id) q.set("task_id", params.task_id);
      if (params?.agent_id) q.set("agent_id", params.agent_id);
      q.set("limit", String(params?.limit || 100));
      q.set("offset", String(params?.offset || 0));
      return get<ListResponse<TaskAssignment>>(`/api/v1/task-assignments?${q.toString()}`);
    },
    get: (id: string) => get<TaskAssignment>(`/api/v1/task-assignments/${id}`),
    create: (data: Partial<TaskAssignment>) => post<TaskAssignment>("/api/v1/task-assignments", data),
    updateStatus: (id: string, status: string) =>
      patch<TaskAssignment>(`/api/v1/task-assignments/${id}/status`, { status }),
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
    get: (id: string) => get<Artifact>(`/api/v1/artifacts/${id}`),
    create: (data: Partial<Artifact>) => post<Artifact>("/api/v1/artifacts", data),
    versions: (artifactId: string) =>
      getList<ArtifactVersion>(`/api/v1/artifacts/${artifactId}/versions`),
    addVersion: (artifactId: string, data: Partial<ArtifactVersion>) =>
      post<ArtifactVersion>(`/api/v1/artifacts/${artifactId}/versions`, data),
  },

  reviews: {
    list: (params?: { run_id?: string; task_id?: string; reviewer_id?: string; limit?: number; offset?: number }) => {
      const q = new URLSearchParams();
      if (params?.run_id) q.set("run_id", params.run_id);
      if (params?.task_id) q.set("task_id", params.task_id);
      if (params?.reviewer_id) q.set("reviewer_id", params.reviewer_id);
      q.set("limit", String(params?.limit || 50));
      q.set("offset", String(params?.offset || 0));
      return get<ListResponse<ReviewReport>>(`/api/v1/reviews?${q.toString()}`);
    },
    get: (id: string) => get<ReviewReport>(`/api/v1/reviews/${id}`),
    create: (data: Partial<ReviewReport>) => post<ReviewReport>("/api/v1/reviews", data),
  },

  handoffs: {
    list: (params?: { task_id?: string; limit?: number; offset?: number }) => {
      const q = new URLSearchParams();
      if (params?.task_id) q.set("task_id", params.task_id);
      q.set("limit", String(params?.limit || 50));
      q.set("offset", String(params?.offset || 0));
      return get<ListResponse<HandoffSnapshot>>(`/api/v1/handoff-snapshots?${q.toString()}`);
    },
    get: (id: string) => get<HandoffSnapshot>(`/api/v1/handoff-snapshots/${id}`),
    create: (data: Partial<HandoffSnapshot>) => post<HandoffSnapshot>("/api/v1/handoff-snapshots", data),
    latestByTask: (taskId: string) =>
      get<HandoffSnapshot>(`/api/v1/tasks/${taskId}/handoff-snapshots/latest`),
  },

  toolCalls: {
    list: (params?: { limit?: number; offset?: number }) => {
      const q = new URLSearchParams();
      q.set("limit", String(params?.limit || 50));
      q.set("offset", String(params?.offset || 0));
      return get<ListResponse<ToolCall>>(`/api/v1/tool-calls?${q.toString()}`);
    },
    get: (id: string) => get<ToolCall>(`/api/v1/tool-calls/${id}`),
    tools: () => getList<unknown>("/api/v1/tools"),
  },

  approvals: {
    list: (params?: { status?: string; limit?: number; offset?: number }) => {
      const q = new URLSearchParams();
      if (params?.status) q.set("status", params.status);
      q.set("limit", String(params?.limit || 50));
      q.set("offset", String(params?.offset || 0));
      return get<ListResponse<ApprovalRequest>>(`/api/v1/approvals?${q.toString()}`);
    },
    get: (id: string) => get<ApprovalRequest>(`/api/v1/approvals/${id}`),
    create: (data: Partial<ApprovalRequest>) => post<ApprovalRequest>("/api/v1/approvals", data),
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
    start: (projectId: string) => post<WorkflowRun>("/api/v1/orchestrator/runs", { project_id: projectId }),
  },

  llmProviders: {
    list: (enabledOnly = false) =>
      get<{ items: LLMProvider[] }>(`/api/v1/llm-providers${enabledOnly ? "?enabled_only=true" : ""}`),
    get: (id: string) => get<LLMProvider>(`/api/v1/llm-providers/${id}`),
    create: (data: Partial<LLMProvider>) => post<LLMProvider>("/api/v1/llm-providers", data),
    update: (id: string, data: Partial<LLMProvider>) => put<LLMProvider>(`/api/v1/llm-providers/${id}`, data),
    delete: (id: string) => del<null>(`/api/v1/llm-providers/${id}`),
    createModel: (providerId: string, data: Partial<LLMModel>) =>
      post<LLMModel>(`/api/v1/llm-providers/${providerId}/models`, data),
    listModels: (providerId: string) =>
      get<{ items: LLMModel[] }>(`/api/v1/llm-providers/${providerId}/models`),
    updateModel: (modelId: string, data: Partial<LLMModel>) =>
      put<LLMModel>(`/api/v1/llm-models/${modelId}`, data),
    deleteModel: (modelId: string) => del<null>(`/api/v1/llm-models/${modelId}`),
  },
};
