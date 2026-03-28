// 状态标签映射 — 与控制台展示一致

export type StatusVariant =
  | "default"
  | "info"
  | "success"
  | "warning"
  | "error"
  | "muted";

interface StatusDef {
  label: string;
  variant: StatusVariant;
}

const taskStatusMap: Record<string, StatusDef> = {
  PENDING: { label: "待处理", variant: "default" },
  READY: { label: "就绪", variant: "info" },
  RUNNING: { label: "运行中", variant: "info" },
  WAITING_APPROVAL: { label: "待审批", variant: "warning" },
  SUCCESS: { label: "已完成", variant: "success" },
  FAILED: { label: "失败", variant: "error" },
  CANCELLED: { label: "已取消", variant: "muted" },
};

const approvalStatusMap: Record<string, StatusDef> = {
  PENDING: { label: "待处理", variant: "warning" },
  APPROVED: { label: "已批准", variant: "success" },
  REJECTED: { label: "已驳回", variant: "error" },
};

export function getTaskStatus(status: string): StatusDef {
  return taskStatusMap[status] || { label: status, variant: "default" };
}

export function getApprovalStatus(status: string): StatusDef {
  return approvalStatusMap[status] || { label: status, variant: "default" };
}

const variantClasses: Record<StatusVariant, string> = {
  default: "bg-gray-100 text-gray-700",
  info: "bg-blue-100 text-blue-700",
  success: "bg-green-100 text-green-700",
  warning: "bg-yellow-100 text-yellow-800",
  error: "bg-red-100 text-red-700",
  muted: "bg-gray-100 text-gray-400",
};

export function getVariantClass(variant: StatusVariant): string {
  return variantClasses[variant];
}

export function formatTime(dateStr?: string): string {
  if (!dateStr) return "-";
  const d = new Date(dateStr);
  return d.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
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
