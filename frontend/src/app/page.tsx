"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import Link from "next/link";
import {
  Activity,
  ArrowRight,
  Database,
  GitBranch,
  ShieldCheck,
  AlertTriangle,
  CheckCircle,
  Clock,
  Wifi,
  WifiOff,
  ScrollText,
} from "lucide-react";
import { api } from "@/lib/api";
import type { WorkflowRun, ApprovalRequest, AuditLog } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { getWorkflowStatus, getApprovalStatus, relativeTime } from "@/lib/utils";
import { useEventStream, type SSEEvent } from "@/lib/useEventStream";

interface HealthStatus {
  api: "ok" | "error" | "loading";
  db: "ready" | "error" | "loading";
}

export default function Home() {
  const [health, setHealth] = useState<HealthStatus>({
    api: "loading",
    db: "loading",
  });
  const [activeRuns, setActiveRuns] = useState<WorkflowRun[]>([]);
  const [pendingApprovals, setPendingApprovals] = useState<ApprovalRequest[]>([]);
  const [recentAudit, setRecentAudit] = useState<AuditLog[]>([]);
  const [failedRuns, setFailedRuns] = useState<WorkflowRun[]>([]);

  const fetchDashboard = useCallback(() => {
    api.workflows.list({ status: "running", limit: 10 })
      .then((res) => setActiveRuns(res.items ?? []))
      .catch(() => setActiveRuns([]));
    api.workflows.list({ status: "failed", limit: 5 })
      .then((res) => setFailedRuns(res.items ?? []))
      .catch(() => setFailedRuns([]));
    api.approvals.list({ status: "pending" })
      .then((res) => setPendingApprovals(res.items ?? []))
      .catch(() => setPendingApprovals([]));
    api.audit.list(8, 0)
      .then((res) => setRecentAudit(res.items ?? []))
      .catch(() => setRecentAudit([]));
  }, []);

  useEffect(() => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

    fetch(`${apiUrl}/healthz`)
      .then((res) => res.json())
      .then(() => setHealth((h) => ({ ...h, api: "ok" })))
      .catch(() => setHealth((h) => ({ ...h, api: "error" })));

    fetch(`${apiUrl}/readyz`)
      .then((res) => res.json())
      .then((data) =>
        setHealth((h) => ({
          ...h,
          db: data.data?.status === "ready" ? "ready" : "error",
        }))
      )
      .catch(() => setHealth((h) => ({ ...h, db: "error" })));

    fetchDashboard();
  }, [fetchDashboard]);

  // SSE real-time updates for dashboard
  const topics = useMemo(() => ["workflow", "approval", "audit", "tool_call"], []);
  const onEvent = useCallback((_evt: SSEEvent) => {
    fetchDashboard();
  }, [fetchDashboard]);

  const { connected, mode } = useEventStream({
    topics,
    onEvent,
    onPoll: fetchDashboard,
    pollInterval: 10000,
  });

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">系统概览</h1>
        <p className="mt-1 text-sm text-gray-500">
          运行状态与核心链路说明
          {connected ? (
            <span className="ml-2 inline-flex items-center gap-1 text-green-600">
              <Wifi className="h-3 w-3" /> 实时
            </span>
          ) : mode === "poll" ? (
            <span className="ml-2 inline-flex items-center gap-1 text-amber-600">
              <WifiOff className="h-3 w-3" /> 轮询
            </span>
          ) : null}
        </p>
      </div>

      {/* Health Cards */}
      <div className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2">
        <HealthCard
          title="API 服务"
          subtitle="/healthz"
          status={health.api}
          okLabel="正常"
          Icon={Activity}
        />
        <HealthCard
          title="数据库"
          subtitle="/readyz"
          status={health.db}
          okLabel="就绪"
          Icon={Database}
        />
      </div>

      {/* Runtime Status Panels */}
      <div className="mb-6 grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Active Workflows */}
        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <div className="mb-3 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Activity className="h-4 w-4 text-blue-600" />
              <span className="text-sm font-medium text-gray-900">运行中工作流</span>
            </div>
            <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700">
              {activeRuns.length}
            </span>
          </div>
          {activeRuns.length === 0 ? (
            <p className="text-xs text-gray-400">无活跃运行</p>
          ) : (
            <ul className="space-y-2">
              {activeRuns.slice(0, 5).map((r) => {
                const st = getWorkflowStatus(r.status);
                return (
                  <li key={r.id} className="flex items-center justify-between text-xs">
                    <Link href="/workflows" className="font-mono text-blue-600 hover:underline">
                      #{r.id.slice(0, 8)}
                    </Link>
                    <StatusBadge label={st.label} variant={st.variant} />
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        {/* Pending Approvals */}
        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <div className="mb-3 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <ShieldCheck className="h-4 w-4 text-amber-600" />
              <span className="text-sm font-medium text-gray-900">待审批</span>
            </div>
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${
              pendingApprovals.length > 0
                ? "bg-amber-100 text-amber-700"
                : "bg-gray-100 text-gray-500"
            }`}>
              {pendingApprovals.length}
            </span>
          </div>
          {pendingApprovals.length === 0 ? (
            <p className="text-xs text-gray-400">无待审批项</p>
          ) : (
            <ul className="space-y-2">
              {pendingApprovals.slice(0, 5).map((a) => {
                const st = getApprovalStatus(a.status);
                return (
                  <li key={a.id} className="flex items-center justify-between text-xs">
                    <Link href="/approvals" className="max-w-[160px] truncate text-gray-700 hover:underline">
                      {a.title}
                    </Link>
                    <StatusBadge label={st.label} variant={st.variant} />
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        {/* Recent Failures */}
        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <div className="mb-3 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-red-600" />
              <span className="text-sm font-medium text-gray-900">最近失败</span>
            </div>
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${
              failedRuns.length > 0
                ? "bg-red-100 text-red-700"
                : "bg-gray-100 text-gray-500"
            }`}>
              {failedRuns.length}
            </span>
          </div>
          {failedRuns.length === 0 ? (
            <div className="flex items-center gap-1 text-xs text-gray-400">
              <CheckCircle className="h-3 w-3 text-green-500" />
              无失败运行
            </div>
          ) : (
            <ul className="space-y-2">
              {failedRuns.slice(0, 5).map((r) => (
                <li key={r.id} className="flex items-center justify-between text-xs">
                  <Link href="/workflows" className="font-mono text-red-600 hover:underline">
                    #{r.id.slice(0, 8)}
                  </Link>
                  <span className="max-w-[120px] truncate text-red-500" title={r.error || ""}>
                    {r.error || "失败"}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {/* Recent Audit Events */}
      {recentAudit.length > 0 && (
        <div className="mb-6 rounded-lg border border-gray-200 bg-white">
          <div className="flex items-center justify-between border-b border-gray-100 px-4 py-3">
            <div className="flex items-center gap-2">
              <ScrollText className="h-4 w-4 text-gray-500" />
              <span className="text-sm font-medium text-gray-900">最近事件</span>
            </div>
            <Link href="/audit" className="text-xs text-blue-600 hover:underline">
              查看全部 <ArrowRight className="ml-0.5 inline h-3 w-3" />
            </Link>
          </div>
          <ul className="divide-y divide-gray-50">
            {recentAudit.map((a) => (
              <li key={a.id} className="flex items-center gap-3 px-4 py-2.5 text-xs">
                <span className="inline-flex shrink-0 rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] uppercase text-gray-600">
                  {a.event_type}
                </span>
                <span className="min-w-0 flex-1 truncate text-gray-700">
                  {a.event_summary}
                </span>
                <span className="shrink-0 text-gray-400">
                  <Clock className="mr-0.5 inline h-3 w-3" />
                  {relativeTime(a.created_at)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Quick Links */}
      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <div className="border-b border-gray-100 bg-gray-50 px-5 py-4">
          <div className="flex items-center gap-2 text-sm font-medium text-gray-900">
            <GitBranch className="h-4 w-4 text-blue-600" />
            灵筹 MVP 控制台
          </div>
          <p className="mt-1 text-sm text-gray-500">
            面向复杂项目交付的多智能体组织操作系统
          </p>
        </div>
        <div className="px-5 py-4 text-sm leading-relaxed text-gray-600">
          通过左侧导航可访问项目、任务看板、Agent、工件、审批与审计。数据接入 API
          后将在此展示实时列表与状态。
        </div>
        <div className="flex flex-wrap gap-2 border-t border-gray-100 bg-gray-50/80 px-5 py-4">
          {[
            { href: "/projects", label: "项目" },
            { href: "/tasks", label: "任务" },
            { href: "/workflows", label: "工作流" },
            { href: "/approvals", label: "审批" },
          ].map((l) => (
            <Link
              key={l.href}
              href={l.href}
              className="inline-flex items-center gap-1 rounded-md border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 transition-colors hover:border-blue-200 hover:bg-blue-50 hover:text-blue-700"
            >
              {l.label}
              <ArrowRight className="h-3 w-3" />
            </Link>
          ))}
        </div>
        <div className="border-t border-gray-100 px-5 py-3 text-xs text-gray-400">
          核心链路：项目 → 阶段 → 任务 → 工件 → 审批 → 审计
        </div>
      </div>
    </div>
  );
}

function HealthCard({
  title,
  subtitle,
  status,
  okLabel,
  Icon,
}: {
  title: string;
  subtitle: string;
  status: string;
  okLabel: string;
  Icon: typeof Activity;
}) {
  const ok = status === "ok" || status === "ready";
  const loading = status === "loading";

  return (
    <div className="flex items-start gap-4 rounded-lg border border-gray-200 bg-white p-4">
      <div
        className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${
          loading
            ? "bg-amber-50 text-amber-600"
            : ok
              ? "bg-green-50 text-green-600"
              : "bg-red-50 text-red-600"
        }`}
      >
        <Icon className={`h-5 w-5 ${loading ? "animate-pulse" : ""}`} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-sm font-medium text-gray-900">{title}</div>
        <div className="mt-0.5 font-mono text-xs text-gray-400">{subtitle}</div>
        <div className="mt-2 text-xs text-gray-500">
          {loading ? "检测中…" : ok ? okLabel : "异常"}
        </div>
      </div>
    </div>
  );
}
