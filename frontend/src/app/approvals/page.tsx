"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { ShieldCheck, Wifi, WifiOff } from "lucide-react";
import { api } from "@/lib/api";
import type { ApprovalRequest, ApprovalScoreBreakdownItem, ApprovalScoreSummary } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { asRecord, asStringArray, formatTime, getApprovalStatus, metadataNumber, relativeTime, truncateText } from "@/lib/utils";
import { useEventStream } from "@/lib/useEventStream";

type FilterTab = "all" | "pending" | "approved" | "rejected";

const TABS: { key: FilterTab; label: string }[] = [
  { key: "all", label: "全部" },
  { key: "pending", label: "待审批" },
  { key: "approved", label: "已批准" },
  { key: "rejected", label: "已拒绝" },
];

type ApprovalArtifact = {
  name: string;
  artifactType: string;
  versionUri: string;
  contentPreview: string;
};

type ApprovalScoreSummaryView = {
  templateKey: string;
  passThreshold?: number;
  totalScore?: number;
  hardGatePassedCount?: number;
  hardGateTotalCount?: number;
  scoreBreakdownSummary: ApprovalScoreBreakdownItem[];
  mustFixItems: string[];
};

function getApprovalArtifacts(metadata: Record<string, unknown>): ApprovalArtifact[] {
  const rawArtifacts = metadata.artifacts;
  if (!Array.isArray(rawArtifacts)) {
    return [];
  }
  return rawArtifacts
    .map((item) => {
      const record = asRecord(item);
      return {
        name: typeof record.name === "string" ? record.name : "未命名工件",
        artifactType: typeof record.artifact_type === "string" ? record.artifact_type : "other",
        versionUri: typeof record.version_uri === "string" ? record.version_uri : "",
        contentPreview:
          typeof record.content_preview === "string" ? record.content_preview : "",
      };
    })
    .filter((item) => item.name || item.versionUri);
}

function asNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function getApprovalScoreSummary(metadata: Record<string, unknown>): ApprovalScoreSummaryView | null {
  const scoreMeta = metadata as ApprovalScoreSummary & Record<string, unknown>;
  const templateKey = typeof scoreMeta.template_key === "string" ? scoreMeta.template_key : "";
  const scoreBreakdownSummary = Array.isArray(scoreMeta.score_breakdown_summary)
    ? scoreMeta.score_breakdown_summary
        .map((item) => {
          const record = asRecord(item);
          return {
            key: typeof record.key === "string" ? record.key : "",
            name: typeof record.name === "string" ? record.name : "未命名评分项",
            weight: asNumber(record.weight) ?? 0,
            score: asNumber(record.score) ?? 0,
            max_score: asNumber(record.max_score) ?? 0,
          };
        })
        .filter((item) => item.key)
    : [];
  const mustFixItems = asStringArray(scoreMeta.must_fix_items);

  if (!templateKey && scoreBreakdownSummary.length === 0 && mustFixItems.length === 0) {
    return null;
  }

  return {
    templateKey,
    passThreshold: metadataNumber(scoreMeta, "pass_threshold") ?? undefined,
    totalScore: metadataNumber(scoreMeta, "total_score") ?? undefined,
    hardGatePassedCount: metadataNumber(scoreMeta, "hard_gate_passed_count") ?? undefined,
    hardGateTotalCount: metadataNumber(scoreMeta, "hard_gate_total_count") ?? undefined,
    scoreBreakdownSummary,
    mustFixItems,
  };
}

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
  const onEvent = useCallback(() => {
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
            const metadata = asRecord(row.metadata);
            const reviewSummary = typeof metadata.review_summary === "string" ? metadata.review_summary : "";
            const reviewId = typeof metadata.review_id === "string" ? metadata.review_id : "";
            const taskTitle = typeof metadata.task_title === "string" ? metadata.task_title : "";
            const findings = asStringArray(metadata.findings);
            const recommendations = asStringArray(metadata.recommendations);
            const artifacts = getApprovalArtifacts(metadata);
            const artifactCount = metadataNumber(metadata, "artifact_count") ?? artifacts.length;
            const scoreSummary = getApprovalScoreSummary(metadata);
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
                  {(reviewSummary || findings.length > 0 || recommendations.length > 0 || artifacts.length > 0) && (
                    <section className="rounded-lg border border-gray-100 bg-gray-50 p-4">
                      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500">
                        <span>审批依据</span>
                        {reviewId && <span>评审报告: {reviewId.slice(0, 8)}</span>}
                        {taskTitle && <span>任务: {taskTitle}</span>}
                        <span>交付物: {artifactCount} 个</span>
                      </div>

                      {reviewSummary && (
                        <p className="mt-3 text-sm leading-6 text-gray-700">{reviewSummary}</p>
                      )}

                      {scoreSummary && (
                        <div className="mt-3 rounded-lg border border-amber-200 bg-white p-3">
                          <div className="grid gap-3 md:grid-cols-4">
                            <div>
                              <p className="text-xs font-medium uppercase tracking-wide text-amber-700">评分模板</p>
                              <p className="mt-1 text-sm font-semibold text-gray-900">
                                {scoreSummary.templateKey || "未记录"}
                              </p>
                            </div>
                            <div>
                              <p className="text-xs font-medium uppercase tracking-wide text-amber-700">总分 / 阈值</p>
                              <p className="mt-1 text-sm font-semibold text-gray-900">
                                {scoreSummary.totalScore ?? "—"} / {scoreSummary.passThreshold ?? "—"}
                              </p>
                            </div>
                            <div>
                              <p className="text-xs font-medium uppercase tracking-wide text-amber-700">硬门槛</p>
                              <p className="mt-1 text-sm font-semibold text-gray-900">
                                {typeof scoreSummary.hardGatePassedCount === "number" &&
                                typeof scoreSummary.hardGateTotalCount === "number"
                                  ? `${scoreSummary.hardGatePassedCount}/${scoreSummary.hardGateTotalCount} 通过`
                                  : "未记录"}
                              </p>
                            </div>
                            <div>
                              <p className="text-xs font-medium uppercase tracking-wide text-amber-700">主要分项</p>
                              <p className="mt-1 text-sm font-semibold text-gray-900">
                                {scoreSummary.scoreBreakdownSummary.length} 项
                              </p>
                            </div>
                          </div>

                          {scoreSummary.mustFixItems.length > 0 && (
                            <div className="mt-3 rounded-md border border-red-200 bg-red-50 px-3 py-2">
                              <p className="text-xs font-medium uppercase tracking-wide text-red-700">必改项</p>
                              <p className="mt-2 text-sm text-red-800">{scoreSummary.mustFixItems.join("；")}</p>
                            </div>
                          )}

                          {scoreSummary.scoreBreakdownSummary.length > 0 && (
                            <div className="mt-3 grid gap-2 md:grid-cols-2">
                              {scoreSummary.scoreBreakdownSummary.map((item) => (
                                <div key={`${row.id}-score-${item.key}`} className="rounded-md border border-gray-200 bg-gray-50 px-3 py-2">
                                  <div className="flex items-center justify-between gap-3">
                                    <p className="text-sm font-medium text-gray-900">{item.name}</p>
                                    <span className="text-xs text-gray-500">
                                      {item.score}/{item.max_score} · 权重 {item.weight}
                                    </span>
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      )}

                      <div className="mt-3 grid gap-3 md:grid-cols-2">
                        <div>
                          <p className="text-xs font-medium uppercase tracking-wide text-gray-500">主要发现</p>
                          {findings.length > 0 ? (
                            <ul className="mt-2 space-y-2 text-sm text-gray-700">
                              {findings.map((item, index) => (
                                <li key={`${row.id}-finding-${index}`} className="rounded-md bg-white px-3 py-2">
                                  {item}
                                </li>
                              ))}
                            </ul>
                          ) : (
                            <p className="mt-2 text-sm text-gray-500">无</p>
                          )}
                        </div>
                        <div>
                          <p className="text-xs font-medium uppercase tracking-wide text-gray-500">整改建议</p>
                          {recommendations.length > 0 ? (
                            <ul className="mt-2 space-y-2 text-sm text-gray-700">
                              {recommendations.map((item, index) => (
                                <li key={`${row.id}-recommendation-${index}`} className="rounded-md bg-white px-3 py-2">
                                  {item}
                                </li>
                              ))}
                            </ul>
                          ) : (
                            <p className="mt-2 text-sm text-gray-500">无</p>
                          )}
                        </div>
                      </div>

                      {artifacts.length > 0 && (
                        <div className="mt-3 space-y-2">
                          {artifacts.map((artifact, index) => (
                            <div key={`${row.id}-artifact-${index}`} className="rounded-md border border-gray-200 bg-white p-3">
                              <div className="flex flex-wrap items-center gap-2 text-sm text-gray-900">
                                <span className="font-medium">{artifact.name}</span>
                                <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
                                  {artifact.artifactType}
                                </span>
                              </div>
                              <p className="mt-2 break-all text-xs text-gray-500">{artifact.versionUri || "未记录版本 URI"}</p>
                              {artifact.contentPreview && (
                                <pre className="mt-3 whitespace-pre-wrap break-words rounded-md bg-gray-950 px-3 py-2 text-xs leading-6 text-gray-100">
                                  {truncateText(artifact.contentPreview, 500)}
                                </pre>
                              )}
                            </div>
                          ))}
                        </div>
                      )}
                    </section>
                  )}
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
