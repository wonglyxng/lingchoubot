"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, Pencil, Trash2 } from "lucide-react";
import { api } from "@/lib/api";
import type { Task, TaskContract, TaskAssignment, HandoffSnapshot } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { FormModal, FormField, inputClass, textareaClass, selectClass } from "@/components/FormModal";
import { formatTime, getTaskStatus } from "@/lib/utils";

function shortId(value?: string): string {
  return value ? value.slice(0, 8) : "—";
}

const allStatuses = [
  "pending", "assigned", "in_progress", "in_review",
  "revision_required", "completed", "failed", "cancelled", "blocked",
];

type TabKey = "info" | "contracts" | "assignments" | "handoffs";

export default function TaskDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = typeof params.id === "string" ? params.id : "";

  const [task, setTask] = useState<Task | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<TabKey>("info");

  const [contracts, setContracts] = useState<TaskContract[]>([]);
  const [assignments, setAssignments] = useState<TaskAssignment[]>([]);
  const [handoffs, setHandoffs] = useState<HandoffSnapshot[]>([]);

  const [showEdit, setShowEdit] = useState(false);
  const [editForm, setEditForm] = useState({ title: "", description: "", priority: 3 });
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(() => {
    if (!id) return;
    setLoading(true);
    setError(null);
    api.tasks.get(id)
      .then((t) => {
        setTask(t);
        setEditForm({ title: t.title, description: t.description, priority: t.priority });
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(() => { load(); }, [load]);

  useEffect(() => {
    if (!id) return;
    if (tab === "contracts") {
      api.taskContracts.list(id)
        .then((list) => setContracts(Array.isArray(list) ? list : []))
        .catch(() => setContracts([]));
    } else if (tab === "assignments") {
      api.taskAssignments.list({ task_id: id })
        .then((res) => setAssignments(res.items ?? []))
        .catch(() => setAssignments([]));
    } else if (tab === "handoffs") {
      api.handoffs.list({ task_id: id })
        .then((res) => setHandoffs(res.items ?? []))
        .catch(() => setHandoffs([]));
    }
  }, [id, tab]);

  const handleTransition = async (status: string) => {
    if (!task) return;
    try {
      const updated = await api.tasks.transition(task.id, status);
      setTask(updated);
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "状态流转失败");
    }
  };

  const handleEdit = async () => {
    if (!task || !editForm.title.trim()) return;
    setSubmitting(true);
    try {
      const updated = await api.tasks.update(task.id, {
        title: editForm.title.trim(),
        description: editForm.description.trim(),
        priority: editForm.priority,
      });
      setTask(updated);
      setShowEdit(false);
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "更新失败");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!task || !confirm("确定删除此任务？此操作不可撤销。")) return;
    try {
      await api.tasks.delete(task.id);
      router.push("/tasks");
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "删除失败");
    }
  };

  const tabs: { key: TabKey; label: string }[] = [
    { key: "info", label: "基本信息" },
    { key: "contracts", label: "契约" },
    { key: "assignments", label: "分派" },
    { key: "handoffs", label: "交接" },
  ];

  if (loading) {
    return (
      <div className="p-6">
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">
          加载中…
        </div>
      </div>
    );
  }

  if (error || !task) {
    return (
      <div className="p-6">
        <Link href="/tasks" className="mb-4 inline-flex items-center gap-1.5 text-sm font-medium text-gray-600 hover:text-blue-700">
          <ArrowLeft className="h-4 w-4" /> 返回任务列表
        </Link>
        <div className="rounded-lg border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
          {error || "未找到任务"}
        </div>
      </div>
    );
  }

  const st = getTaskStatus(task.status);

  return (
    <div className="p-6">
      <Link href="/tasks" className="mb-4 inline-flex items-center gap-1.5 text-sm font-medium text-gray-600 hover:text-blue-700">
        <ArrowLeft className="h-4 w-4" /> 返回任务列表
      </Link>

      {/* Header */}
      <div className="mb-6 rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex-1">
            <h1 className="text-2xl font-semibold text-gray-900">{task.title}</h1>
            {task.description && (
              <p className="mt-2 text-sm text-gray-600">{task.description}</p>
            )}
            <p className="mt-2 text-xs text-gray-400">
              创建于 {formatTime(task.created_at)} · 优先级 {task.priority}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <StatusBadge label={st.label} variant={st.variant} />
            <button onClick={() => setShowEdit(true)} className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600" title="编辑">
              <Pencil className="h-4 w-4" />
            </button>
            <button onClick={handleDelete} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600" title="删除">
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* Status transition buttons */}
        <div className="mt-4 flex flex-wrap gap-2 border-t border-gray-100 pt-4">
          <span className="self-center text-xs font-medium text-gray-500">流转到：</span>
          {allStatuses.filter((s) => s !== task.status).map((s) => {
            const def = getTaskStatus(s);
            return (
              <button
                key={s}
                onClick={() => handleTransition(s)}
                className="rounded-md border border-gray-200 px-2.5 py-1 text-xs font-medium text-gray-700 transition-colors hover:border-blue-300 hover:bg-blue-50 hover:text-blue-700"
              >
                {def.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Edit Modal */}
      <FormModal open={showEdit} onClose={() => setShowEdit(false)} title="编辑任务" onSubmit={handleEdit} submitting={submitting}>
        <FormField label="标题" required>
          <input className={inputClass} value={editForm.title} onChange={(e) => setEditForm((f) => ({ ...f, title: e.target.value }))} maxLength={300} required />
        </FormField>
        <FormField label="描述">
          <textarea className={textareaClass} rows={3} value={editForm.description} onChange={(e) => setEditForm((f) => ({ ...f, description: e.target.value }))} maxLength={5000} />
        </FormField>
        <FormField label="优先级">
          <select className={selectClass} value={editForm.priority} onChange={(e) => setEditForm((f) => ({ ...f, priority: Number(e.target.value) }))}>
            {[1, 2, 3, 4, 5].map((p) => <option key={p} value={p}>{p}</option>)}
          </select>
        </FormField>
      </FormModal>

      {/* Tabs */}
      <div className="mb-4 flex flex-wrap gap-2 border-b border-gray-200 pb-px">
        {tabs.map((t) => (
          <button
            key={t.key}
            type="button"
            onClick={() => setTab(t.key)}
            className={`relative -mb-px rounded-t-md border px-3 py-2 text-sm font-medium transition-colors ${
              tab === t.key
                ? "border-gray-200 border-b-white bg-white text-blue-700"
                : "border-transparent text-gray-500 hover:text-gray-900"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
        {tab === "info" && (
          <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div><dt className="text-xs font-medium text-gray-500">ID</dt><dd className="mt-1 text-sm font-mono text-gray-900">{task.id}</dd></div>
            <div><dt className="text-xs font-medium text-gray-500">项目 ID</dt><dd className="mt-1 text-sm font-mono text-gray-900">{task.project_id}</dd></div>
            {task.phase_id && <div><dt className="text-xs font-medium text-gray-500">阶段 ID</dt><dd className="mt-1 text-sm font-mono text-gray-900">{task.phase_id}</dd></div>}
            {task.assignee_id && <div><dt className="text-xs font-medium text-gray-500">执行者 ID</dt><dd className="mt-1 text-sm font-mono text-gray-900">{task.assignee_id}</dd></div>}
            <div><dt className="text-xs font-medium text-gray-500">更新于</dt><dd className="mt-1 text-sm text-gray-900">{formatTime(task.updated_at)}</dd></div>
          </dl>
        )}

        {tab === "contracts" && (
          <ul className="divide-y divide-gray-100">
            {contracts.length === 0 ? (
              <li className="py-6 text-center text-sm text-gray-500">暂无契约</li>
            ) : contracts.map((c) => (
              <li key={c.id} className="py-4 first:pt-0 last:pb-0">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium text-gray-900">版本 {c.version}</span>
                  <span className="text-xs text-gray-400">{formatTime(c.created_at)}</span>
                </div>
                <p className="mt-1 text-sm text-gray-600">{c.scope || "—"}</p>
                {c.completion_definition && (
                  <p className="mt-1 text-xs text-gray-500">完成定义：{c.completion_definition}</p>
                )}
              </li>
            ))}
          </ul>
        )}

        {tab === "assignments" && (
          <ul className="divide-y divide-gray-100">
            {assignments.length === 0 ? (
              <li className="py-6 text-center text-sm text-gray-500">暂无分派记录</li>
            ) : assignments.map((a) => (
              <li key={a.id} className="py-4 first:pt-0 last:pb-0">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm text-gray-900">Agent: <span className="font-mono">{a.agent_id.slice(0, 8)}</span></span>
                  <StatusBadge label={a.status} variant={a.status === "completed" ? "success" : a.status === "active" ? "info" : "default"} />
                </div>
                <p className="mt-1 text-xs text-gray-500">角色: {a.role} · {formatTime(a.created_at)}</p>
              </li>
            ))}
          </ul>
        )}

        {tab === "handoffs" && (
          <ul className="divide-y divide-gray-100">
            {handoffs.length === 0 ? (
              <li className="py-6 text-center text-sm text-gray-500">暂无交接快照</li>
            ) : handoffs.map((h) => (
              <li key={h.id} className="py-4 first:pt-0 last:pb-0">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium text-gray-900">
                    Agent: <span className="font-mono">{shortId(h.agent_id)}</span>
                  </span>
                  <span className="text-xs text-gray-400">{formatTime(h.created_at)}</span>
                </div>
                {h.summary && <p className="mt-1 text-sm text-gray-600">{h.summary}</p>}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
