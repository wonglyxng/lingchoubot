"use client";

import { useEffect, useState, useCallback } from "react";
import { Activity, ChevronDown, ChevronRight, Play } from "lucide-react";
import { api } from "@/lib/api";
import type { WorkflowRun, WorkflowStep, Project } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { FormModal, FormField, selectClass } from "@/components/FormModal";
import { getWorkflowStatus, formatTime, relativeTime } from "@/lib/utils";

function stepStatusVariant(status: string) {
  switch (status) {
    case "completed": return "success" as const;
    case "running": return "info" as const;
    case "failed": return "error" as const;
    case "skipped": return "muted" as const;
    default: return "default" as const;
  }
}

function StepRow({ step }: { step: WorkflowStep }) {
  const v = stepStatusVariant(step.status);
  return (
    <div className="flex items-center gap-3 rounded border border-gray-100 bg-gray-50/50 px-3 py-2 text-sm">
      <span className="w-6 shrink-0 text-center text-xs text-gray-400">
        #{step.sort_order}
      </span>
      <span className="min-w-0 flex-1 font-medium text-gray-800">
        {step.step_name}
      </span>
      <StatusBadge label={step.status} variant={v} />
      {step.duration_ms != null && step.duration_ms > 0 && (
        <span className="text-xs text-gray-500">
          {step.duration_ms >= 1000
            ? `${(step.duration_ms / 1000).toFixed(1)}s`
            : `${step.duration_ms}ms`}
        </span>
      )}
      {step.error && (
        <span className="max-w-[200px] truncate text-xs text-red-500" title={step.error}>
          {step.error}
        </span>
      )}
    </div>
  );
}

function RunCard({
  run,
  expanded,
  onToggle,
}: {
  run: WorkflowRun;
  expanded: boolean;
  onToggle: () => void;
}) {
  const [steps, setSteps] = useState<WorkflowStep[]>([]);
  const [loadingSteps, setLoadingSteps] = useState(false);
  const st = getWorkflowStatus(run.status);

  useEffect(() => {
    if (!expanded) return;
    setLoadingSteps(true);
    api.workflows
      .steps(run.id)
      .then(setSteps)
      .catch(() => {})
      .finally(() => setLoadingSteps(false));
  }, [expanded, run.id]);

  return (
    <div className="rounded-lg border border-gray-200 bg-white">
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-gray-50"
      >
        {expanded ? (
          <ChevronDown className="h-4 w-4 shrink-0 text-gray-400" />
        ) : (
          <ChevronRight className="h-4 w-4 shrink-0 text-gray-400" />
        )}
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-semibold text-gray-900">
              运行 #{run.id.slice(0, 8)}
            </span>
            <StatusBadge label={st.label} variant={st.variant} />
            {run.trigger && (
              <span className="text-xs text-gray-500">触发: {run.trigger}</span>
            )}
          </div>
          <p className="mt-0.5 text-xs text-gray-500">
            {formatTime(run.created_at)} · {relativeTime(run.created_at)}
          </p>
        </div>
        {run.result_summary && (
          <span className="max-w-[300px] truncate text-xs text-gray-600">
            {run.result_summary}
          </span>
        )}
      </button>

      {expanded && (
        <div className="border-t border-gray-100 px-4 py-3">
          {loadingSteps && (
            <p className="text-sm text-gray-500">加载步骤中…</p>
          )}
          {!loadingSteps && steps.length === 0 && (
            <p className="text-sm text-gray-500">暂无步骤</p>
          )}
          {!loadingSteps && steps.length > 0 && (
            <div className="space-y-2">
              {steps
                .sort((a, b) => a.sort_order - b.sort_order)
                .map((s) => (
                  <StepRow key={s.id} step={s} />
                ))}
            </div>
          )}
          {run.error && (
            <div className="mt-2 rounded bg-red-50 px-3 py-2 text-sm text-red-700">
              错误: {run.error}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function WorkflowsPage() {
  const [loading, setLoading] = useState(true);
  const [runs, setRuns] = useState<WorkflowRun[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [showStart, setShowStart] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedProject, setSelectedProject] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api.workflows
      .list({ limit: 50 })
      .then((res) => setRuns(res.items))
      .catch((e: Error) => setError(e.message || "加载失败"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const openStart = async () => {
    try {
      const res = await api.projects.list(200, 0);
      setProjects(res.items);
    } catch { /* ignore */ }
    setShowStart(true);
  };

  const handleStart = async () => {
    if (!selectedProject) return;
    setSubmitting(true);
    try {
      await api.workflows.start(selectedProject);
      setShowStart(false);
      setSelectedProject("");
      load();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "启动失败";
      alert(msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-full bg-gray-50 p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-700">
            <Activity className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">工作流运行</h1>
            <p className="mt-1 text-sm text-gray-500">查看编排引擎运行记录与步骤</p>
          </div>
        </div>
        <button
          type="button"
          onClick={openStart}
          className="inline-flex items-center gap-2 rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700"
        >
          <Play className="h-4 w-4" />
          启动运行
        </button>
      </div>

      <FormModal
        open={showStart}
        onClose={() => setShowStart(false)}
        title="启动工作流运行"
        onSubmit={handleStart}
        submitting={submitting}
        submitLabel="启动"
      >
        <FormField label="选择项目" required>
          <select
            className={selectClass}
            value={selectedProject}
            onChange={(e) => setSelectedProject(e.target.value)}
            required
          >
            <option value="">选择项目</option>
            {projects.map((p) => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </select>
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

      {!loading && !error && runs.length === 0 && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">
          暂无运行记录
        </div>
      )}

      {!loading && !error && runs.length > 0 && (
        <div className="space-y-3">
          {runs.map((r) => (
            <RunCard
              key={r.id}
              run={r}
              expanded={expandedId === r.id}
              onToggle={() =>
                setExpandedId((prev) => (prev === r.id ? null : r.id))
              }
            />
          ))}
        </div>
      )}
    </div>
  );
}
