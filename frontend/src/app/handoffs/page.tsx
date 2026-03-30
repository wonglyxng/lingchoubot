"use client";

import { useEffect, useState, useCallback } from "react";
import { ArrowRightLeft } from "lucide-react";
import { api } from "@/lib/api";
import type { HandoffSnapshot } from "@/lib/types";
import { formatTime } from "@/lib/utils";

function renderList(data: unknown): string {
  if (!data) return "—";
  if (Array.isArray(data)) return data.join("、") || "—";
  return JSON.stringify(data);
}

export default function HandoffsPage() {
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<HandoffSnapshot[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api.handoffs
      .list()
      .then((res) => setItems(res.items ?? []))
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-teal-50 text-teal-600">
          <ArrowRightLeft className="h-5 w-5" />
        </div>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">交接快照</h1>
          <p className="mt-1 text-sm text-gray-500">Agent 间交接记录</p>
        </div>
      </div>

      {loading && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">加载中…</div>
      )}
      {!loading && error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">{error}</div>
      )}
      {!loading && !error && items.length === 0 && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">暂无交接记录</div>
      )}
      {!loading && !error && items.length > 0 && (
        <div className="space-y-3">
          {items.map((h) => (
            <div key={h.id} className="rounded-lg border border-gray-200 bg-white">
              <button
                type="button"
                onClick={() => setExpanded(expanded === h.id ? null : h.id)}
                className="flex w-full items-center justify-between px-5 py-4 text-left"
              >
                <div>
                  <span className="text-sm font-medium text-gray-900">
                    From <span className="font-mono">{h.from_agent_id.slice(0, 8)}</span>
                    {h.to_agent_id && <> → <span className="font-mono">{h.to_agent_id.slice(0, 8)}</span></>}
                  </span>
                  <span className="ml-3 text-xs text-gray-500">任务 {h.task_id.slice(0, 8)}</span>
                </div>
                <span className="text-xs text-gray-400">{formatTime(h.created_at)}</span>
              </button>
              {expanded === h.id && (
                <div className="space-y-3 border-t border-gray-100 px-5 py-4 text-sm">
                  {h.context_summary && (
                    <div><span className="text-xs font-medium text-gray-500">上下文摘要</span><p className="mt-1 text-gray-700">{h.context_summary}</p></div>
                  )}
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                    <div><span className="text-xs font-medium text-gray-500">已完成</span><p className="mt-1 text-gray-700">{renderList(h.completed_items)}</p></div>
                    <div><span className="text-xs font-medium text-gray-500">待完成</span><p className="mt-1 text-gray-700">{renderList(h.pending_items)}</p></div>
                    <div><span className="text-xs font-medium text-gray-500">风险</span><p className="mt-1 text-gray-700">{renderList(h.risks)}</p></div>
                    <div><span className="text-xs font-medium text-gray-500">下一步</span><p className="mt-1 text-gray-700">{renderList(h.next_steps)}</p></div>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
