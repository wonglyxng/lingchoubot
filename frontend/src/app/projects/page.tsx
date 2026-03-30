"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { FolderKanban, Plus } from "lucide-react";
import { api } from "@/lib/api";
import type { Project } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { FormModal, FormField, inputClass, textareaClass } from "@/components/FormModal";
import { formatTime, getProjectStatus } from "@/lib/utils";

function truncate(text: string, max: number): string {
  const t = text?.trim() || "";
  if (t.length <= max) return t;
  return `${t.slice(0, max)}…`;
}

export default function ProjectsPage() {
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<Project[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form, setForm] = useState({ name: "", description: "" });

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api.projects
      .list()
      .then((res) => setItems(res.items))
      .catch((e: Error) => setError(e.message || "加载失败"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    if (!form.name.trim()) return;
    setSubmitting(true);
    try {
      await api.projects.create({ name: form.name.trim(), description: form.description.trim() });
      setShowCreate(false);
      setForm({ name: "", description: "" });
      load();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "创建失败";
      alert(msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="p-6">
      <div className="mb-6 flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600">
            <FolderKanban className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">项目</h1>
            <p className="mt-1 text-sm text-gray-500">查看与管理项目列表</p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => setShowCreate(true)}
          className="inline-flex items-center gap-2 rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700"
        >
          <Plus className="h-4 w-4" />
          新建项目
        </button>
      </div>

      <FormModal
        open={showCreate}
        onClose={() => setShowCreate(false)}
        title="新建项目"
        onSubmit={handleCreate}
        submitting={submitting}
      >
        <FormField label="项目名称" required>
          <input
            className={inputClass}
            value={form.name}
            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            placeholder="例如：灵筹 MVP"
            maxLength={200}
            required
          />
        </FormField>
        <FormField label="描述">
          <textarea
            className={textareaClass}
            rows={3}
            value={form.description}
            onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
            placeholder="项目简要描述"
            maxLength={2000}
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

      {!loading && !error && items.length === 0 && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">
          暂无项目
        </div>
      )}

      {!loading && !error && items.length > 0 && (
        <ul className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {items.map((p) => {
            const st = getProjectStatus(p.status);
            return (
              <li key={p.id}>
                <Link
                  href={`/projects/${p.id}`}
                  className="block h-full rounded-lg border border-gray-200 bg-white p-4 shadow-sm transition-colors hover:border-blue-200 hover:bg-blue-50/40"
                >
                  <div className="flex items-start justify-between gap-2">
                    <h2 className="text-base font-semibold text-gray-900">
                      {p.name}
                    </h2>
                    <StatusBadge label={st.label} variant={st.variant} />
                  </div>
                  <p className="mt-2 line-clamp-2 text-sm text-gray-500">
                    {truncate(p.description, 160) || "—"}
                  </p>
                  <p className="mt-3 text-xs text-gray-400">
                    创建于 {formatTime(p.created_at)}
                  </p>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
