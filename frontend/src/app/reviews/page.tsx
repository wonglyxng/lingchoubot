"use client";

import { useEffect, useState, useCallback } from "react";
import { FileSearch } from "lucide-react";
import { api } from "@/lib/api";
import type { ReviewReport } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { formatTime } from "@/lib/utils";

function verdictBadge(verdict: string) {
  switch (verdict) {
    case "approved":
      return { label: "通过", variant: "success" as const };
    case "rejected":
      return { label: "拒绝", variant: "error" as const };
    case "needs_revision":
      return { label: "需修订", variant: "warning" as const };
    default:
      return { label: verdict, variant: "default" as const };
  }
}

export default function ReviewsPage() {
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<ReviewReport[]>([]);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api.reviews
      .list()
      .then((res) => setItems(res.items ?? []))
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-orange-50 text-orange-600">
          <FileSearch className="h-5 w-5" />
        </div>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">评审报告</h1>
          <p className="mt-1 text-sm text-gray-500">独立评审记录</p>
        </div>
      </div>

      {loading && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">加载中…</div>
      )}
      {!loading && error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">{error}</div>
      )}
      {!loading && !error && items.length === 0 && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">暂无评审报告</div>
      )}
      {!loading && !error && items.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                <th className="px-4 py-3">摘要</th>
                <th className="px-4 py-3">任务 ID</th>
                <th className="px-4 py-3">评审者</th>
                <th className="px-4 py-3">结论</th>
                <th className="px-4 py-3">时间</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {items.map((r) => {
                const vb = verdictBadge(r.verdict);
                return (
                  <tr key={r.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 font-medium text-gray-900">{r.summary || "—"}</td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-600">{r.task_id.slice(0, 8)}</td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-600">{r.reviewer_id.slice(0, 8)}</td>
                    <td className="px-4 py-3"><StatusBadge label={vb.label} variant={vb.variant} /></td>
                    <td className="px-4 py-3 text-xs text-gray-400">{formatTime(r.created_at)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
