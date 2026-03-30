"use client";

import { useEffect, useState, useCallback } from "react";
import { FileText } from "lucide-react";
import { api } from "@/lib/api";
import type { Task, TaskContract } from "@/lib/types";
import { formatTime } from "@/lib/utils";

export default function ContractsPage() {
  const [loading, setLoading] = useState(true);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [contracts, setContracts] = useState<Map<string, TaskContract[]>>(new Map());
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.tasks.list({ limit: 100 });
      const taskList = res.items ?? [];
      setTasks(taskList);
      // Load contracts for each task (best effort)
      const map = new Map<string, TaskContract[]>();
      const results = await Promise.allSettled(
        taskList.map((t) => api.taskContracts.list(t.id).then((list) => ({ taskId: t.id, list })))
      );
      for (const r of results) {
        if (r.status === "fulfilled") {
          const arr = Array.isArray(r.value.list) ? r.value.list : [];
          if (arr.length > 0) map.set(r.value.taskId, arr);
        }
      }
      setContracts(map);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const tasksWithContracts = tasks.filter((t) => contracts.has(t.id));

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-cyan-50 text-cyan-600">
          <FileText className="h-5 w-5" />
        </div>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">任务契约</h1>
          <p className="mt-1 text-sm text-gray-500">任务范围、完成定义和验收标准</p>
        </div>
      </div>

      {loading && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">加载中…</div>
      )}
      {!loading && error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">{error}</div>
      )}
      {!loading && !error && tasksWithContracts.length === 0 && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">
          暂无任务契约（可在任务详情页中查看）
        </div>
      )}
      {!loading && !error && tasksWithContracts.length > 0 && (
        <div className="space-y-3">
          {tasksWithContracts.map((t) => {
            const cList = contracts.get(t.id) ?? [];
            return (
              <div key={t.id} className="rounded-lg border border-gray-200 bg-white">
                <button
                  type="button"
                  onClick={() => setExpanded(expanded === t.id ? null : t.id)}
                  className="flex w-full items-center justify-between px-5 py-4 text-left"
                >
                  <div>
                    <span className="text-sm font-medium text-gray-900">{t.title}</span>
                    <span className="ml-3 text-xs text-gray-500">{cList.length} 个版本</span>
                  </div>
                  <span className="text-xs text-gray-400">{formatTime(t.created_at)}</span>
                </button>
                {expanded === t.id && (
                  <div className="border-t border-gray-100 px-5 py-4">
                    <ul className="space-y-4">
                      {cList.map((c) => (
                        <li key={c.id} className="rounded border border-gray-100 bg-gray-50 p-3">
                          <div className="flex items-center justify-between">
                            <span className="text-xs font-semibold text-gray-700">v{c.version}</span>
                            <span className="text-xs text-gray-400">{formatTime(c.created_at)}</span>
                          </div>
                          <dl className="mt-2 space-y-2 text-sm">
                            <div><dt className="text-xs font-medium text-gray-500">范围</dt><dd className="text-gray-700">{c.scope || "—"}</dd></div>
                            {c.non_goals && <div><dt className="text-xs font-medium text-gray-500">非目标</dt><dd className="text-gray-700">{c.non_goals}</dd></div>}
                            {c.completion_definition && <div><dt className="text-xs font-medium text-gray-500">完成定义</dt><dd className="text-gray-700">{c.completion_definition}</dd></div>}
                          </dl>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
