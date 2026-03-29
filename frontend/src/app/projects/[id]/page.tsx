"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { api } from "@/lib/api";
import type { Artifact, AuditLog, Phase, Project, Task } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import {
  formatTime,
  getPhaseStatus,
  getProjectStatus,
  getTaskStatus,
} from "@/lib/utils";

type TabKey = "phases" | "tasks" | "artifacts" | "timeline";

export default function ProjectDetailPage() {
  const params = useParams();
  const id = typeof params.id === "string" ? params.id : "";

  const [tab, setTab] = useState<TabKey>("phases");
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [project, setProject] = useState<Project | null>(null);
  const [phases, setPhases] = useState<Phase[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [timeline, setTimeline] = useState<AuditLog[]>([]);

  const sortedPhases = useMemo(
    () => [...phases].sort((a, b) => a.sort_order - b.sort_order),
    [phases],
  );

  useEffect(() => {
    if (!id) {
      setLoading(false);
      setLoadError("无效的项目 ID");
      return;
    }

    let cancelled = false;
    setLoading(true);
    setLoadError(null);

    Promise.all([
      api.projects.get(id),
      api.phases.listByProject(id),
      api.tasks.list({ project_id: id, limit: 10, offset: 0 }),
      api.artifacts.list({ project_id: id, limit: 10, offset: 0 }),
      api.audit.projectTimeline(id, 20, 0),
    ])
      .then(([proj, ph, taskRes, artRes, auditRes]) => {
        if (cancelled) return;
        setProject(proj);
        setPhases(ph);
        setTasks(taskRes.items);
        setArtifacts(artRes.items);
        setTimeline(auditRes.items);
      })
      .catch((e: Error) => {
        if (!cancelled) setLoadError(e.message || "加载失败");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [id]);

  const tabs: { key: TabKey; label: string }[] = [
    { key: "phases", label: "阶段" },
    { key: "tasks", label: "最近任务" },
    { key: "artifacts", label: "最近工件" },
    { key: "timeline", label: "项目时间线" },
  ];

  return (
    <div className="p-6">
      <div className="mb-6">
        <Link
          href="/projects"
          className="mb-4 inline-flex items-center gap-1.5 text-sm font-medium text-gray-600 transition-colors hover:text-blue-700"
        >
          <ArrowLeft className="h-4 w-4" />
          返回项目列表
        </Link>

        {loading && (
          <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">
            加载中…
          </div>
        )}

        {!loading && loadError && (
          <div className="rounded-lg border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
            {loadError}
          </div>
        )}

        {!loading && !loadError && project && (
          <div className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h1 className="text-2xl font-semibold text-gray-900">
                  {project.name}
                </h1>
                <p className="mt-1 text-sm text-gray-500">
                  创建于 {formatTime(project.created_at)}
                </p>
              </div>
              {(() => {
                const st = getProjectStatus(project.status);
                return <StatusBadge label={st.label} variant={st.variant} />;
              })()}
            </div>
          </div>
        )}
      </div>

      {!loading && !loadError && project && (
        <>
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
            {tab === "phases" && (
              <ul className="divide-y divide-gray-100">
                {sortedPhases.length === 0 ? (
                  <li className="py-6 text-center text-sm text-gray-500">
                    暂无阶段
                  </li>
                ) : (
                  sortedPhases.map((ph) => {
                    const st = getPhaseStatus(ph.status);
                    return (
                      <li key={ph.id} className="py-4 first:pt-0 last:pb-0">
                        <div className="flex flex-wrap items-start justify-between gap-2">
                          <span className="font-medium text-gray-900">
                            {ph.name}
                          </span>
                          <StatusBadge label={st.label} variant={st.variant} />
                        </div>
                        {ph.description ? (
                          <p className="mt-1 text-sm text-gray-500">
                            {ph.description}
                          </p>
                        ) : null}
                      </li>
                    );
                  })
                )}
              </ul>
            )}

            {tab === "tasks" && (
              <ul className="divide-y divide-gray-100">
                {tasks.length === 0 ? (
                  <li className="py-6 text-center text-sm text-gray-500">
                    暂无任务
                  </li>
                ) : (
                  tasks.map((task) => {
                    const st = getTaskStatus(task.status);
                    return (
                      <li key={task.id} className="py-4 first:pt-0 last:pb-0">
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <span className="font-medium text-gray-900">
                            {task.title}
                          </span>
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-gray-500">
                              优先级 {task.priority}
                            </span>
                            <StatusBadge label={st.label} variant={st.variant} />
                          </div>
                        </div>
                      </li>
                    );
                  })
                )}
              </ul>
            )}

            {tab === "artifacts" && (
              <ul className="divide-y divide-gray-100">
                {artifacts.length === 0 ? (
                  <li className="py-6 text-center text-sm text-gray-500">
                    暂无工件
                  </li>
                ) : (
                  artifacts.map((a) => (
                    <li key={a.id} className="py-4 first:pt-0 last:pb-0">
                      <div className="flex flex-wrap items-start justify-between gap-2">
                        <span className="font-medium text-gray-900">
                          {a.name}
                        </span>
                        <span className="text-xs font-medium uppercase tracking-wide text-gray-500">
                          {a.artifact_type}
                        </span>
                      </div>
                      <p className="mt-1 text-xs text-gray-400">
                        {formatTime(a.created_at)}
                      </p>
                    </li>
                  ))
                )}
              </ul>
            )}

            {tab === "timeline" && (
              <ul className="divide-y divide-gray-100">
                {timeline.length === 0 ? (
                  <li className="py-6 text-center text-sm text-gray-500">
                    暂无审计记录
                  </li>
                ) : (
                  timeline.map((log) => (
                    <li key={log.id} className="py-4 first:pt-0 last:pb-0">
                      <div className="text-xs font-medium uppercase tracking-wide text-gray-500">
                        {log.event_type}
                      </div>
                      <p className="mt-1 text-sm text-gray-900">
                        {log.event_summary || "—"}
                      </p>
                      <p className="mt-1 text-xs text-gray-400">
                        {formatTime(log.created_at)}
                      </p>
                    </li>
                  ))
                )}
              </ul>
            )}
          </div>
        </>
      )}
    </div>
  );
}
