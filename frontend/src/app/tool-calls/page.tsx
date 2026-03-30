"use client";

import { useEffect, useState, useCallback } from "react";
import { Wrench } from "lucide-react";
import { api } from "@/lib/api";
import type { ToolCall } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { formatTime } from "@/lib/utils";

function toolCallStatus(status: string) {
  switch (status) {
    case "success":
      return { label: "成功", variant: "success" as const };
    case "failed":
      return { label: "失败", variant: "error" as const };
    case "denied":
      return { label: "拒绝", variant: "warning" as const };
    case "running":
      return { label: "执行中", variant: "info" as const };
    default:
      return { label: status, variant: "default" as const };
  }
}

export default function ToolCallsPage() {
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<ToolCall[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api.toolCalls
      .list()
      .then((res) => setItems(res.items ?? []))
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-violet-50 text-violet-600">
          <Wrench className="h-5 w-5" />
        </div>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">工具调用</h1>
          <p className="mt-1 text-sm text-gray-500">Tool Gateway 调用历史</p>
        </div>
      </div>

      {loading && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">加载中…</div>
      )}
      {!loading && error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">{error}</div>
      )}
      {!loading && !error && items.length === 0 && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">暂无工具调用记录</div>
      )}
      {!loading && !error && items.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                <th className="px-4 py-3">工具</th>
                <th className="px-4 py-3">操作</th>
                <th className="px-4 py-3">Agent</th>
                <th className="px-4 py-3">状态</th>
                <th className="px-4 py-3">耗时</th>
                <th className="px-4 py-3">时间</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {items.map((tc) => {
                const st = toolCallStatus(tc.status);
                return (
                  <tr
                    key={tc.id}
                    className="cursor-pointer hover:bg-gray-50"
                    onClick={() => setExpanded(expanded === tc.id ? null : tc.id)}
                  >
                    <td className="px-4 py-3 font-medium text-gray-900">{tc.tool_name}</td>
                    <td className="px-4 py-3 text-gray-600">{tc.action}</td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-600">{tc.agent_id.slice(0, 8)}</td>
                    <td className="px-4 py-3"><StatusBadge label={st.label} variant={st.variant} /></td>
                    <td className="px-4 py-3 text-xs text-gray-500">{tc.duration_ms}ms</td>
                    <td className="px-4 py-3 text-xs text-gray-400">{formatTime(tc.created_at)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {expanded && (() => {
            const tc = items.find((t) => t.id === expanded);
            if (!tc) return null;
            return (
              <div className="border-t border-gray-200 px-5 py-4 text-sm">
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <div>
                    <span className="text-xs font-medium text-gray-500">输入</span>
                    <pre className="mt-1 max-h-40 overflow-auto rounded bg-gray-50 p-2 text-xs text-gray-700">
                      {JSON.stringify(tc.input, null, 2)}
                    </pre>
                  </div>
                  <div>
                    <span className="text-xs font-medium text-gray-500">输出</span>
                    <pre className="mt-1 max-h-40 overflow-auto rounded bg-gray-50 p-2 text-xs text-gray-700">
                      {JSON.stringify(tc.output, null, 2)}
                    </pre>
                  </div>
                </div>
                {tc.error_message && (
                  <div className="mt-3">
                    <span className="text-xs font-medium text-red-500">错误</span>
                    <p className="mt-1 text-sm text-red-700">{tc.error_message}</p>
                  </div>
                )}
                {tc.denied_reason && (
                  <div className="mt-3">
                    <span className="text-xs font-medium text-orange-500">拒绝原因</span>
                    <p className="mt-1 text-sm text-orange-700">{tc.denied_reason}</p>
                  </div>
                )}
              </div>
            );
          })()}
        </div>
      )}
    </div>
  );
}
