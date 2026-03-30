"use client";

import { useEffect, useMemo, useState, useCallback } from "react";
import { ListChecks, Plus } from "lucide-react";
import { api } from "@/lib/api";
import type { Task, Project } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { FormModal, FormField, inputClass, textareaClass, selectClass } from "@/components/FormModal";
import { getTaskStatus, relativeTime, type StatusVariant } from "@/lib/utils";

const MAIN_STATUSES = [
  "pending",
  "assigned",
  "in_progress",
  "in_review",
  "completed",
] as const;

const OTHER_STATUSES = [
  "revision_required",
  "failed",
  "cancelled",
  "blocked",
] as const;

function priorityVariant(priority: number): StatusVariant {
  if (priority <= 2) return "error";
  if (priority <= 5) return "warning";
  return "muted";
}

function TaskCard({ task }: { task: Task }) {
  const pv = priorityVariant(task.priority);
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-3 shadow-sm">
      <p className="text-sm font-medium text-gray-900 line-clamp-2">
        {task.title}
      </p>
      <div className="mt-2 flex flex-wrap items-center gap-2">
        <StatusBadge label={`P${task.priority}`} variant={pv} />
      </div>
      <p className="mt-2 text-xs text-gray-500">
        {relativeTime(task.created_at)}
      </p>
    </div>
  );
}

function Column({
  statusKey,
  tasks,
}: {
  statusKey: string;
  tasks: Task[];
}) {
  const def = getTaskStatus(statusKey);
  return (
    <div className="flex w-[min(100%,280px)] shrink-0 flex-col rounded-lg border border-gray-200 bg-gray-50/80">
      <div className="border-b border-gray-200 bg-white px-3 py-2.5">
        <h2 className="text-sm font-semibold text-gray-900">
          {def.label}
          <span className="ml-2 font-normal text-gray-500">({tasks.length})</span>
        </h2>
      </div>
      <div className="flex flex-1 flex-col gap-2 p-2">
        {tasks.length === 0 ? (
          <p className="px-1 py-4 text-center text-xs text-gray-400">暂无</p>
        ) : (
          tasks.map((t) => <TaskCard key={t.id} task={t} />)
        )}
      </div>
    </div>
  );
}

export default function TasksPage() {
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<Task[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [projects, setProjects] = useState<Project[]>([]);
  const [form, setForm] = useState({ title: "", description: "", project_id: "", priority: 5 });

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api.tasks
      .list({ limit: 500, offset: 0 })
      .then((res) => setItems(res.items))
      .catch((e: Error) => setError(e.message || "加载失败"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const openCreate = async () => {
    try {
      const res = await api.projects.list(200, 0);
      setProjects(res.items);
    } catch { /* ignore */ }
    setShowCreate(true);
  };

  const handleCreate = async () => {
    if (!form.title.trim() || !form.project_id) return;
    setSubmitting(true);
    try {
      await api.tasks.create({
        title: form.title.trim(),
        description: form.description.trim(),
        project_id: form.project_id,
        priority: form.priority,
      });
      setShowCreate(false);
      setForm({ title: "", description: "", project_id: "", priority: 5 });
      load();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "创建失败";
      alert(msg);
    } finally {
      setSubmitting(false);
    }
  };

  const byStatus = useMemo(() => {
    const map = new Map<string, Task[]>();
    for (const s of MAIN_STATUSES) map.set(s, []);
    map.set("__other__", []);
    for (const t of items) {
      if (MAIN_STATUSES.includes(t.status as (typeof MAIN_STATUSES)[number])) {
        map.get(t.status)!.push(t);
      } else if (
        OTHER_STATUSES.includes(t.status as (typeof OTHER_STATUSES)[number])
      ) {
        map.get("__other__")!.push(t);
      } else {
        map.get("__other__")!.push(t);
      }
    }
    return map;
  }, [items]);

  const hasAnyTask = items.length > 0;

  return (
    <div className="min-h-full bg-gray-50 p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-700">
            <ListChecks className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">任务看板</h1>
            <p className="mt-1 text-sm text-gray-500">按状态分列查看任务</p>
          </div>
        </div>
        <button
          type="button"
          onClick={openCreate}
          className="inline-flex items-center gap-2 rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700"
        >
          <Plus className="h-4 w-4" />
          新建任务
        </button>
      </div>

      <FormModal
        open={showCreate}
        onClose={() => setShowCreate(false)}
        title="新建任务"
        onSubmit={handleCreate}
        submitting={submitting}
      >
        <FormField label="所属项目" required>
          <select
            className={selectClass}
            value={form.project_id}
            onChange={(e) => setForm((f) => ({ ...f, project_id: e.target.value }))}
            required
          >
            <option value="">选择项目</option>
            {projects.map((p) => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </select>
        </FormField>
        <FormField label="任务标题" required>
          <input
            className={inputClass}
            value={form.title}
            onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
            placeholder="例如：实现用户认证模块"
            maxLength={200}
            required
          />
        </FormField>
        <FormField label="优先级（1=最高，10=最低）">
          <input
            className={inputClass}
            type="number"
            min={1}
            max={10}
            value={form.priority}
            onChange={(e) => setForm((f) => ({ ...f, priority: parseInt(e.target.value) || 5 }))}
          />
        </FormField>
        <FormField label="描述">
          <textarea
            className={textareaClass}
            rows={3}
            value={form.description}
            onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
            placeholder="任务简要描述"
            maxLength={5000}
          />
        </FormField>
      </FormModal>

      {loading && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">
          加载中…
        </div>
      )}

      {!loading && error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
          {error}
        </div>
      )}

      {!loading && !error && !hasAnyTask && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">
          暂无任务
        </div>
      )}

      {!loading && !error && hasAnyTask && (
        <div className="space-y-6">
          <div className="overflow-x-auto pb-2">
            <div className="flex min-w-min gap-4">
              {MAIN_STATUSES.map((s) => (
                <Column
                  key={s}
                  statusKey={s}
                  tasks={byStatus.get(s) ?? []}
                />
              ))}
            </div>
          </div>

          {(byStatus.get("__other__")?.length ?? 0) > 0 && (
            <div className="rounded-lg border border-gray-200 bg-white">
              <div className="border-b border-gray-200 px-4 py-3">
                <h2 className="text-sm font-semibold text-gray-900">
                  其他
                  <span className="ml-2 font-normal text-gray-500">
                    ({byStatus.get("__other__")!.length})
                  </span>
                </h2>
                <p className="mt-0.5 text-xs text-gray-500">
                  需修订、失败、已取消、阻塞及其他状态
                </p>
              </div>
              <div className="grid gap-2 p-4 sm:grid-cols-2 lg:grid-cols-3">
                {byStatus.get("__other__")!.map((t) => (
                  <div key={t.id} className="flex flex-col gap-1">
                    <TaskCard task={t} />
                    <div className="px-1">
                      <StatusBadge
                        label={getTaskStatus(t.status).label}
                        variant={getTaskStatus(t.status).variant}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
