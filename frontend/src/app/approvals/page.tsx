"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { ShieldCheck, Wifi, WifiOff } from "lucide-react";
import { api } from "@/lib/api";
import type { ApprovalRequest } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { formatTime, getApprovalStatus, relativeTime } from "@/lib/utils";
import { useEventStream, type SSEEvent } from "@/lib/useEventStream";

type FilterTab = "all" | "pending" | "approved" | "rejected";

const TABS: { key: FilterTab; label: string }[] = [
  { key: "all", label: "全部" },
  { key: "pending", label: "待审批" },
  { key: "approved", label: "已批准" },
  { key: "rejected", label: "已拒绝" },
];

export default function ApprovalsPage() {
  const [tab, setTab] = useState<FilterTab>("all");
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<ApprovalRequest[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [decidingId, setDecidingId] = useState<string | null>(null);

  const fetchList = useCallback(() => {
    const params = tab === "all" ? undefined : { status: tab };
    return api.approvals.list(params).then((res) => setItems(res.items));
  }, [tab]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    fetchList()
      .catch((e: Error) => {
        if (!cancelled) setError(e.message || "加载失败");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [fetchList]);

  // SSE real-time updates
  const topics = useMemo(() => ["approval"], []);
  const onEvent = useCallback((_evt: SSEEvent) => {
    fetchList().catch(() => {});
  }, [fetchList]);

  const { connected, mode } = useEventStream({
    topics,
    onEvent,
    onPoll: () => { fetchList().catch(() => {}); },
    pollInterval: 5000,
  });

  async function handleDecide(id: string, status: "approved" | "rejected") {
    const msg = status === "approved" ? "确认批准该审批？" : "确认拒绝该审批？";
    if (!window.confirm(msg)) return;
    setDecidingId(id);
    setError(null);
    try {
      await api.approvals.decide(id, status, "");
      const params = tab === "all" ? undefined : { status: tab };
      const res = await api.approvals.list(params);
      setItems(res.items);
    } catch (e) {
      setError(e instanceof Error ? e.message : "操作失败");
    } finally {
      setDecidingId(null);
    }
  }

  return (
    <div className="min-h-full bg-gray-50 p-6">
      <div className="mb-6 flex items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-700">
          <ShieldCheck className="h-5 w-5" />
        </div>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">审批中心</h1>
          <p className="mt-1 text-sm text-gray-500">
            待办与已处理审批
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
      </div>

      <div className="mb-4 flex flex-wrap gap-2 border-b border-gray-200 pb-3">
        {TABS.map((t) => (
          <button
            key={t.key}
            type="button"
            onClick={() => setTab(t.key)}
            className={`rounded-md border px-3 py-1.5 text-sm font-medium transition-colors ${
              tab === t.key
                ? "border-gray-300 bg-white text-gray-900 shadow-sm"
                : "border-transparent bg-transparent text-gray-600 hover:bg-gray-100"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {loading && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">
          加载中…
        </div>
      )}

      {!loading && error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-white px-5 py-4 text-sm text-red-700">
          {error}
        </div>
      )}

      {!loading && !error && items.length === 0 && (
        <div className="flex flex-col items-center justify-center rounded-lg border border-gray-200 bg-white py-16 text-center">
          <ShieldCheck className="mb-3 h-10 w-10 text-gray-300" />
          <p className="text-sm text-gray-500">暂无审批记录</p>
        </div>
      )}

      {!loading && items.length > 0 && (
        <ul className="space-y-3">
          {items.map((row) => {
            const st = getApprovalStatus(row.status);
            const pending = row.status === "pending";
            return (
              <li
                key={row.id}
                className="flex flex-col gap-3 rounded-lg border border-gray-200 bg-white p-4 sm:flex-row sm:items-start sm:justify-between"
              >
                <div className="min-w-0 flex-1 space-y-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="text-base font-semibold text-gray-900">
                      {row.title}
                    </h2>
                    <StatusBadge label={st.label} variant={st.variant} />
                  </div>
                  <p className="text-sm text-gray-600">
                    {row.description?.trim() ? row.description : "—"}
                  </p>
                  <p className="text-xs text-gray-500">
                    <span className="text-gray-600">{formatTime(row.created_at)}</span>
                    <span className="mx-2 text-gray-300">·</span>
                    <span>{relativeTime(row.created_at)}</span>
                  </p>
                </div>
                {pending && (
                  <div className="flex shrink-0 gap-2 sm:pt-0.5">
                    <button
                      type="button"
                      disabled={decidingId === row.id}
                      onClick={() => handleDecide(row.id, "approved")}
                      className="rounded-md border border-gray-200 bg-white px-3 py-1.5 text-sm font-medium text-green-700 hover:bg-green-50 disabled:opacity-50"
                    >
                      批准
                    </button>
                    <button
                      type="button"
                      disabled={decidingId === row.id}
                      onClick={() => handleDecide(row.id, "rejected")}
                      className="rounded-md border border-gray-200 bg-white px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-50 disabled:opacity-50"
                    >
                      拒绝
                    </button>
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
