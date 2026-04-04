"use client";

import Link from "next/link";
import { useEffect, useState, useCallback, useRef, useMemo } from "react";
import { Activity, AlertTriangle, ChevronDown, ChevronRight, Play, RefreshCw, Wifi, WifiOff } from "lucide-react";
import { api } from "@/lib/api";
import type { WorkflowRun, WorkflowStep, Project } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { FormModal, FormField, selectClass } from "@/components/FormModal";
import { getWorkflowStatus, formatTime, relativeTime } from "@/lib/utils";
import { useEventStream, type SSEEvent } from "@/lib/useEventStream";

function stepsEqual(left: WorkflowStep[], right: WorkflowStep[]) {
  if (left.length !== right.length) {
    return false;
  }

  return left.every((step, index) => {
    const other = right[index];
    return other
      && step.id === other.id
      && step.status === other.status
      && step.summary === other.summary
      && step.error === other.error
      && step.sort_order === other.sort_order
      && step.started_at === other.started_at
      && step.completed_at === other.completed_at;
  });
}

function mergeRunList(existing: WorkflowRun[], updated: WorkflowRun) {
  let found = false;
  const next = existing.map((run) => {
    if (run.id !== updated.id) {
      return run;
    }

    found = true;
    return { ...run, ...updated };
  });

  return found ? next : [updated, ...existing];
}

function extractRunID(event: SSEEvent) {
  if (typeof event.target_id === "string" && event.target_id.length > 0) {
    return event.target_id;
  }

  if (!event.data || typeof event.data !== "object") {
    return null;
  }

  const maybeAudit = event.data as { after_state?: unknown };
  let afterState = maybeAudit.after_state;

  if (typeof afterState === "string") {
    try {
      afterState = JSON.parse(afterState) as unknown;
    } catch {
      return null;
    }
  }

  if (!afterState || typeof afterState !== "object") {
    return null;
  }

  const runID = (afterState as Record<string, unknown>).run_id;
  return typeof runID === "string" && runID.length > 0 ? runID : null;
}

function stepStatusVariant(status: string) {
  switch (status) {
    case "completed": return "success" as const;
    case "running": return "info" as const;
    case "failed": return "error" as const;
    case "skipped": return "muted" as const;
    default: return "default" as const;
  }
}

type ManualInterventionInfo = {
  reasonCode: string;
  reason: string;
  agentRole?: string;
  taskId?: string;
  taskTitle?: string;
  availableActions: string[];
};

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is string => typeof item === "string" && item.length > 0);
}

function getManualIntervention(run: WorkflowRun): ManualInterventionInfo | null {
  const metadata = asRecord(run.metadata);
  if (!metadata) {
    return null;
  }
  const intervention = asRecord(metadata.manual_intervention);
  if (!intervention) {
    return null;
  }
  const reasonCode = typeof intervention.reason_code === "string" ? intervention.reason_code : "";
  if (!reasonCode) {
    return null;
  }
  return {
    reasonCode,
    reason: typeof intervention.reason === "string" ? intervention.reason : run.error,
    agentRole: typeof intervention.agent_role === "string" ? intervention.agent_role : undefined,
    taskId: typeof intervention.task_id === "string" ? intervention.task_id : undefined,
    taskTitle: typeof intervention.task_title === "string" ? intervention.task_title : undefined,
    availableActions: asStringArray(intervention.available_actions),
  };
}

function StepRow({ step }: { step: WorkflowStep }) {
  const v = stepStatusVariant(step.status);
  const durationMs = step.started_at && step.completed_at
    ? new Date(step.completed_at).getTime() - new Date(step.started_at).getTime()
    : 0;
  return (
    <div className="flex items-center gap-3 rounded border border-gray-100 bg-gray-50/50 px-3 py-2 text-sm">
      <span className="w-6 shrink-0 text-center text-xs text-gray-400">
        #{step.sort_order}
      </span>
      <span className="min-w-0 flex-1 font-medium text-gray-800">
        {step.name}
      </span>
      <StatusBadge label={step.status} variant={v} />
      {durationMs > 0 && (
        <span className="text-xs text-gray-500">
          {durationMs >= 1000
            ? `${(durationMs / 1000).toFixed(1)}s`
            : `${durationMs}ms`}
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
  onRunUpdated,
  liveUpdatesEnabled,
}: {
  run: WorkflowRun;
  expanded: boolean;
  onToggle: () => void;
  onRunUpdated?: (run: WorkflowRun) => void;
  liveUpdatesEnabled: boolean;
}) {
  const [steps, setSteps] = useState<WorkflowStep[]>([]);
  const [loadingSteps, setLoadingSteps] = useState(false);
  const [resuming, setResuming] = useState(false);
  const [manualActing, setManualActing] = useState(false);
  const [manualNote, setManualNote] = useState("");
  const st = getWorkflowStatus(run.status);
  const isRunning = run.status === "running" || run.status === "pending";
  const shouldPoll = !liveUpdatesEnabled && isRunning;
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const canResume = run.status === "waiting_manual_intervention" || run.status === "waiting_approval";
  const manualIntervention = useMemo(() => getManualIntervention(run), [run]);
  const canEscalateToApproval = run.status === "waiting_manual_intervention"
    && manualIntervention?.availableActions.includes("escalate_to_approval");

  useEffect(() => {
    const nextSteps = Array.isArray(run.steps) ? run.steps : [];
    if (nextSteps.length === 0) {
      return;
    }

    setSteps((current) => (stepsEqual(current, nextSteps) ? current : nextSteps));
  }, [run.steps]);

  const handleResume = async () => {
    setResuming(true);
    try {
      await api.workflows.resume(run.id);
      const updated = await api.workflows.get(run.id);
      onRunUpdated?.(updated);
      setSteps(Array.isArray(updated.steps) ? updated.steps : []);
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "恢复执行失败");
    } finally {
      setResuming(false);
    }
  };

  const handleEscalateToApproval = async () => {
    setManualActing(true);
    try {
      await api.workflows.resolveManualIntervention(run.id, {
        action: "escalate_to_approval",
        note: manualNote.trim(),
      });
      const updated = await api.workflows.get(run.id);
      onRunUpdated?.(updated);
      setSteps(Array.isArray(updated.steps) ? updated.steps : []);
      setManualNote("");
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "人工放行失败");
    } finally {
      setManualActing(false);
    }
  };

  // Load steps + poll if running
  useEffect(() => {
    if (!expanded) return;

    const fetchData = async (showLoading: boolean) => {
      if (showLoading) {
        setLoadingSteps(true);
      }

      try {
        const updated = await api.workflows.get(run.id);
        const nextSteps = Array.isArray(updated.steps) ? updated.steps : [];

        setSteps((current) => (stepsEqual(current, nextSteps) ? current : nextSteps));
        if (updated) {
          if (
            updated.status !== run.status
            || updated.summary !== run.summary
            || updated.error !== run.error
          ) {
            onRunUpdated?.(updated);
          }
        }
      } catch {
        if (showLoading) {
          setSteps([]);
        }
      } finally {
        if (showLoading) {
          setLoadingSteps(false);
        }
      }
    };

    void fetchData(true);

    if (shouldPoll) {
      pollRef.current = setInterval(() => {
        void fetchData(false);
      }, 2000);
    }

    return () => {
      if (pollRef.current) {
        clearInterval(pollRef.current);
        pollRef.current = null;
      }
    };
  }, [expanded, run.id, run.status, run.summary, run.error, shouldPoll, onRunUpdated]);

  return (
    <div className="rounded-lg border border-gray-200 bg-white">
      <div className="flex items-center gap-3 px-4 py-3 hover:bg-gray-50">
        <button
          type="button"
          onClick={onToggle}
          className="flex min-w-0 flex-1 items-center gap-3 text-left"
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
              {isRunning && (
                <RefreshCw className="h-3.5 w-3.5 animate-spin text-blue-500" />
              )}
            </div>
            <p className="mt-0.5 text-xs text-gray-500">
              {formatTime(run.created_at)} · {relativeTime(run.created_at)}
            </p>
          </div>
          {run.summary && (
            <span className="max-w-[300px] truncate text-xs text-gray-600">
              {run.summary}
            </span>
          )}
        </button>
        {canResume && (
          <button
            type="button"
            onClick={handleResume}
            disabled={resuming || manualActing}
            className="inline-flex shrink-0 items-center gap-1.5 rounded-md border border-blue-200 bg-blue-50 px-3 py-1.5 text-xs font-medium text-blue-700 hover:bg-blue-100 disabled:cursor-not-allowed disabled:opacity-60"
          >
            <Play className="h-3.5 w-3.5" />
            {resuming ? "恢复中..." : "恢复执行"}
          </button>
        )}
      </div>

      {expanded && (
        <div className="border-t border-gray-100 px-4 py-3">
          <div className="mb-3 flex items-center justify-end">
            <Link
              href={`/reviews?run_id=${encodeURIComponent(run.id)}`}
              className="text-xs font-medium text-blue-600 hover:text-blue-700"
            >
              查看本次评审
            </Link>
          </div>
          {loadingSteps && (
            <p className="text-sm text-gray-500">加载步骤中…</p>
          )}
          {!loadingSteps && steps.length === 0 && (
            <p className="text-sm text-gray-500">暂无步骤</p>
          )}
          {!loadingSteps && steps.length > 0 && (
            <div className="space-y-2">
              {[...steps]
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
          {run.status === "waiting_approval" && (
            <div className="mt-2 rounded bg-amber-50 px-3 py-2 text-sm text-amber-800">
              当前运行已在审批关口暂停。同一阶段内任务可并行待批；跨阶段会等待本阶段审批收口后再继续推进。
            </div>
          )}
          {run.status === "waiting_manual_intervention" && (
            manualIntervention?.reasonCode === "rework_limit_reached" ? (
              <div className="mt-2 rounded border border-amber-200 bg-amber-50 px-3 py-3 text-sm text-amber-900">
                <div className="flex items-start gap-2">
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-600" />
                  <div className="min-w-0 flex-1">
                    <div className="font-medium">当前运行因任务返工次数达到上限而暂停，等待人工处理。</div>
                    <div className="mt-1 text-amber-800">
                      {manualIntervention.taskTitle
                        ? `阻塞任务：${manualIntervention.taskTitle}。`
                        : "当前阻塞任务已达到最大返工次数。"} 这不是供应商故障；直接恢复执行只会重跑当前链路，不会修改当前评审结论。
                    </div>
                    {canEscalateToApproval && (
                      <div className="mt-3 space-y-2">
                        <textarea
                          value={manualNote}
                          onChange={(e) => setManualNote(e.target.value)}
                          placeholder="可选：填写人工放行说明，写入审批上下文与审计日志"
                          rows={3}
                          className="w-full rounded-md border border-amber-200 bg-white px-3 py-2 text-sm text-gray-800 outline-none ring-0 placeholder:text-gray-400 focus:border-amber-400"
                        />
                        <div className="flex flex-wrap items-center gap-3">
                          <button
                            type="button"
                            onClick={handleEscalateToApproval}
                            disabled={manualActing || resuming}
                            className="inline-flex items-center gap-1.5 rounded-md border border-amber-300 bg-white px-3 py-1.5 text-xs font-medium text-amber-900 hover:bg-amber-100 disabled:cursor-not-allowed disabled:opacity-60"
                          >
                            <Play className="h-3.5 w-3.5" />
                            {manualActing ? "处理中..." : "人工放行到审批"}
                          </button>
                          <span className="text-xs text-amber-800">
                            会把当前任务送入审批关口，并继续推进工作流。
                          </span>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ) : (
              <div className="mt-2 rounded border border-red-200 bg-red-50 px-3 py-3 text-sm text-red-800">
                <div className="flex items-start gap-2">
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-red-600" />
                  <div>
                    <div className="font-medium">当前运行因 LLM 执行失败而暂停，等待人工介入。</div>
                    <div className="mt-1 text-red-700">修复供应商配置、模型权限或上下文问题后，可点击“恢复执行”继续。</div>
                  </div>
                </div>
              </div>
            )
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
  const eventRefreshRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const loadRuns = useCallback(async (showLoading: boolean) => {
    if (showLoading) {
      setLoading(true);
      setError(null);
    }

    api.workflows
      .list({ limit: 50 })
      .then((res) => setRuns(res.items))
      .catch((e: Error) => {
        if (showLoading) {
          setError(e.message || "加载失败");
        }
      })
      .finally(() => {
        if (showLoading) {
          setLoading(false);
        }
      });
  }, []);

  useEffect(() => {
    void loadRuns(true);
  }, [loadRuns]);

  useEffect(() => () => {
    if (eventRefreshRef.current) {
      clearTimeout(eventRefreshRef.current);
      eventRefreshRef.current = null;
    }
  }, []);

  const scheduleListRefresh = useCallback(() => {
    if (eventRefreshRef.current) {
      return;
    }

    eventRefreshRef.current = setTimeout(() => {
      eventRefreshRef.current = null;
      void loadRuns(false);
    }, 300);
  }, [loadRuns]);

  const refreshSingleRun = useCallback((runID: string) => {
    api.workflows
      .get(runID)
      .then((updated) => {
        setRuns((current) => mergeRunList(current, updated));
      })
      .catch(() => {
        scheduleListRefresh();
      });
  }, [scheduleListRefresh]);

  // SSE real-time updates: refresh list on any workflow event
  const topics = useMemo(() => ["workflow"], []);
  const onEvent = useCallback((event: SSEEvent) => {
    const runID = extractRunID(event);
    if (runID) {
      refreshSingleRun(runID);
      return;
    }

    scheduleListRefresh();
  }, [refreshSingleRun, scheduleListRefresh]);

  const { connected, mode } = useEventStream({
    topics,
    onEvent,
    onPoll: () => {
      void loadRuns(false);
    },
    pollInterval: 5000,
  });

  const openStart = async () => {
    try {
      const res = await api.projects.list(200, 0);
      setProjects(res.items ?? []);
    } catch {
      setProjects([]);
    }
    setShowStart(true);
  };

  const handleStart = async () => {
    if (!selectedProject) return;
    setSubmitting(true);
    try {
      const newRun = await api.workflows.start(selectedProject);
      setShowStart(false);
      setSelectedProject("");
      // Insert the new run at the top and auto-expand it
      setRuns((prev) => [newRun, ...prev]);
      setExpandedId(newRun.id);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "启动失败";
      alert(msg);
    } finally {
      setSubmitting(false);
    }
  };

  const handleRunUpdated = useCallback((updated: WorkflowRun) => {
    setRuns((prev) => mergeRunList(prev, updated));
  }, []);

  return (
    <div className="min-h-full bg-gray-50 p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-700">
            <Activity className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">工作流运行</h1>
            <p className="mt-1 text-sm text-gray-500">
              查看编排引擎运行记录与步骤
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
              liveUpdatesEnabled={connected && mode === "sse"}
              onToggle={() =>
                setExpandedId((prev) => (prev === r.id ? null : r.id))
              }
              onRunUpdated={handleRunUpdated}
            />
          ))}
        </div>
      )}
    </div>
  );
}
