"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  Activity,
  ArrowRight,
  Database,
  GitBranch,
} from "lucide-react";

interface HealthStatus {
  api: "ok" | "error" | "loading";
  db: "ready" | "error" | "loading";
}

export default function Home() {
  const [health, setHealth] = useState<HealthStatus>({
    api: "loading",
    db: "loading",
  });

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
  }, []);

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">系统概览</h1>
        <p className="mt-1 text-sm text-gray-500">
          运行状态与核心链路说明
        </p>
      </div>

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
