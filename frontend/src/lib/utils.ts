export type StatusVariant = "default" | "info" | "success" | "warning" | "error" | "muted";

interface StatusDef {
  label: string;
  variant: StatusVariant;
}

const taskStatusMap: Record<string, StatusDef> = {
  pending:            { label: "待处理", variant: "default" },
  assigned:           { label: "已分派", variant: "info" },
  in_progress:        { label: "进行中", variant: "info" },
  in_review:          { label: "评审中", variant: "warning" },
  pending_approval:    { label: "待审批", variant: "warning" },
  revision_required:  { label: "需修订", variant: "error" },
  completed:          { label: "已完成", variant: "success" },
  failed:             { label: "失败",   variant: "error" },
  cancelled:          { label: "已取消", variant: "muted" },
  blocked:            { label: "阻塞",   variant: "warning" },
};

const approvalStatusMap: Record<string, StatusDef> = {
  pending:  { label: "待审批", variant: "warning" },
  approved: { label: "已批准", variant: "success" },
  rejected: { label: "已拒绝", variant: "error" },
};

const projectStatusMap: Record<string, StatusDef> = {
  planning:  { label: "规划中", variant: "default" },
  active:    { label: "进行中", variant: "info" },
  paused:    { label: "暂停",   variant: "warning" },
  completed: { label: "已完成", variant: "success" },
  cancelled: { label: "已取消", variant: "muted" },
};

const phaseStatusMap: Record<string, StatusDef> = {
  pending:   { label: "待开始", variant: "default" },
  active:    { label: "进行中", variant: "info" },
  completed: { label: "已完成", variant: "success" },
  skipped:   { label: "已跳过", variant: "muted" },
};

const agentRoleMap: Record<string, string> = {
  pm:         "项目经理",
  supervisor: "主管",
  worker:     "执行者",
  reviewer:   "评审者",
};

const agentTypeMap: Record<string, string> = {
  mock:  "模拟",
  llm:   "LLM",
  human: "人工",
};

const agentSpecMap: Record<string, string> = {
  general:  "通用",
  backend:  "后端",
  frontend: "前端",
  qa:       "测试",
  release:  "发布",
  devops:   "运维",
  design:   "设计",
};

const workflowStatusMap: Record<string, StatusDef> = {
  pending:   { label: "待运行", variant: "default" },
  running:   { label: "运行中", variant: "info" },
  waiting_approval: { label: "等待审批", variant: "warning" },
  waiting_manual_intervention: { label: "待人工介入", variant: "error" },
  completed: { label: "已完成", variant: "success" },
  failed:    { label: "失败",   variant: "error" },
  cancelled: { label: "已取消", variant: "muted" },
};

export function getTaskStatus(status: string): StatusDef {
  return taskStatusMap[status] || { label: status, variant: "default" };
}

export function getApprovalStatus(status: string): StatusDef {
  return approvalStatusMap[status] || { label: status, variant: "default" };
}

export function getProjectStatus(status: string): StatusDef {
  return projectStatusMap[status] || { label: status, variant: "default" };
}

export function getPhaseStatus(status: string): StatusDef {
  return phaseStatusMap[status] || { label: status, variant: "default" };
}

export function getAgentRole(role: string): string {
  return agentRoleMap[role] || role;
}

export function getAgentType(t: string): string {
  return agentTypeMap[t] || t;
}

export function getAgentSpec(s: string): string {
  return agentSpecMap[s] || s;
}

export function getWorkflowStatus(status: string): StatusDef {
  return workflowStatusMap[status] || { label: status, variant: "default" };
}

const variantClasses: Record<StatusVariant, string> = {
  default: "bg-gray-100 text-gray-700",
  info:    "bg-blue-100 text-blue-700",
  success: "bg-green-100 text-green-700",
  warning: "bg-yellow-100 text-yellow-800",
  error:   "bg-red-100 text-red-700",
  muted:   "bg-gray-100 text-gray-400",
};

export function getVariantClass(variant: StatusVariant): string {
  return variantClasses[variant];
}

export function formatTime(dateStr?: string): string {
  if (!dateStr) return "-";
  const d = new Date(dateStr);
  return d.toLocaleString("zh-CN", {
    year: "numeric", month: "2-digit", day: "2-digit",
    hour: "2-digit", minute: "2-digit", second: "2-digit",
  });
}

export function relativeTime(dateStr?: string): string {
  if (!dateStr) return "-";
  const d = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - d.getTime();
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return `${seconds}秒前`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}小时前`;
  const days = Math.floor(hours / 24);
  return `${days}天前`;
}

export function asRecord(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }
  return value as Record<string, unknown>;
}

export function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is string => typeof item === "string" && item.trim().length > 0);
}

export function metadataString(metadata: Record<string, unknown>, key: string): string {
  const value = metadata[key];
  return typeof value === "string" ? value : "";
}

export function metadataNumber(metadata: Record<string, unknown>, key: string): number | null {
  const value = metadata[key];
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  return null;
}

export function truncateText(value: string, max = 240): string {
  if (value.length <= max) {
    return value;
  }
  return `${value.slice(0, max)}...`;
}
