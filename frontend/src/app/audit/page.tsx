"use client";

import { useEffect, useState, useCallback, useMemo, useRef } from "react";
import { ScrollText, Wifi, WifiOff } from "lucide-react";
import { api } from "@/lib/api";
import type { AuditLog } from "@/lib/types";
import { formatTime, relativeTime } from "@/lib/utils";
import { useEventStream, type SSEEvent } from "@/lib/useEventStream";

function mergeAuditItems(existing: AuditLog[], incoming: AuditLog[]) {
  const byID = new Map(existing.map((item) => [item.id, item]));
  for (const item of incoming) {
    byID.set(item.id, item);
  }
  return Array.from(byID.values())
    .sort((left, right) => new Date(right.created_at).getTime() - new Date(left.created_at).getTime())
    .slice(0, 50);
}

function parseAuditEvent(event: SSEEvent) {
  if (!event.data || typeof event.data !== "object") {
    return null;
  }

  const row = event.data as Partial<AuditLog>;
  if (typeof row.id !== "string" || typeof row.event_type !== "string" || typeof row.created_at !== "string") {
    return null;
  }

  return row as AuditLog;
}

export default function AuditPage() {
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<AuditLog[]>([]);
  const [error, setError] = useState<string | null>(null);
  const refreshRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchList = useCallback(async (showLoading: boolean) => {
    if (showLoading) {
      setLoading(true);
      setError(null);
    }

    try {
      const res = await api.audit.list();
      setItems(res.items);
    } catch (e: unknown) {
      if (showLoading) {
        setError(e instanceof Error ? e.message : "加载失败");
      }
    } finally {
      if (showLoading) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    void fetchList(true);
  }, [fetchList]);

  useEffect(() => () => {
    if (refreshRef.current) {
      clearTimeout(refreshRef.current);
      refreshRef.current = null;
    }
  }, []);

  const scheduleRefresh = useCallback(() => {
    if (refreshRef.current) {
      return;
    }

    refreshRef.current = setTimeout(() => {
      refreshRef.current = null;
      void fetchList(false);
    }, 300);
  }, [fetchList]);

  // SSE real-time updates — audit receives all topics
  const topics = useMemo(() => ["audit", "workflow", "approval", "tool_call"], []);
  const onEvent = useCallback((event: SSEEvent) => {
    const row = parseAuditEvent(event);
    if (!row) {
      scheduleRefresh();
      return;
    }

    setError(null);
    setLoading(false);
    setItems((current) => mergeAuditItems(current, [row]));
  }, [scheduleRefresh]);

  const { connected, mode } = useEventStream({
    topics,
    onEvent,
    onPoll: () => {
      void fetchList(false);
    },
    pollInterval: 10000,
  });

  return (
    <div className="min-h-full bg-gray-50 p-6">
      <div className="mb-6 flex items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-700">
          <ScrollText className="h-5 w-5" />
        </div>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">审计</h1>
          <p className="mt-1 text-sm text-gray-500">
            关键操作时间线
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

      {loading && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">
          加载中…
        </div>
      )}

      {!loading && error && (
        <div className="rounded-lg border border-red-200 bg-white px-5 py-4 text-sm text-red-700">
          {error}
        </div>
      )}

      {!loading && !error && items.length === 0 && (
        <div className="flex flex-col items-center justify-center rounded-lg border border-gray-200 bg-white py-16 text-center">
          <ScrollText className="mb-3 h-10 w-10 text-gray-300" />
          <p className="text-sm text-gray-500">暂无审计记录</p>
        </div>
      )}

      {!loading && !error && items.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white p-6">
          <div className="relative pl-8">
            <div
              className="absolute bottom-2 left-[7px] top-2 w-px bg-gray-200"
              aria-hidden
            />
            <ul className="space-y-0">
              {items.map((row) => (
                <li key={row.id} className="relative pb-10 last:pb-0">
                  <div
                    className="absolute left-0 top-1.5 h-3 w-3 rounded-full border-2 border-white bg-gray-400 ring-1 ring-gray-200"
                    aria-hidden
                  />
                  <div className="space-y-2 pl-4">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="inline-flex rounded-md bg-gray-100 px-2 py-0.5 text-[11px] font-medium uppercase tracking-wide text-gray-700">
                        {row.event_type}
                      </span>
                    </div>
                    <p className="text-sm font-medium text-gray-900">
                      {row.event_summary}
                    </p>
                    <p className="text-xs text-gray-500">
                      <span className="font-mono text-gray-600">
                        {row.actor_type}:{row.actor_id}
                      </span>
                      <span className="mx-2 text-gray-300">→</span>
                      <span className="font-mono text-gray-600">
                        {row.target_type}:{row.target_id}
                      </span>
                    </p>
                    <p className="text-xs text-gray-500">
                      <span className="text-gray-600">{formatTime(row.created_at)}</span>
                      <span className="mx-2 text-gray-300">·</span>
                      <span>{relativeTime(row.created_at)}</span>
                    </p>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </div>
  );
}
