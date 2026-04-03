"use client";

import { useEffect, useState, useCallback } from "react";
import { FileSearch } from "lucide-react";
import { api } from "@/lib/api";
import type {
  ReviewHardGateResult,
  ReviewReport,
  ReviewReworkBrief,
  ReviewScoreItemResult,
  ReviewScorecardMetadata,
} from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { asRecord, asStringArray, formatTime, metadataNumber, truncateText } from "@/lib/utils";

type ReviewArtifact = {
  artifactId: string;
  name: string;
  artifactType: string;
  versionUri: string;
  contentPreview: string;
};

type ReviewScorecardView = {
  templateKey: string;
  taskCategory: string;
  passThreshold?: number;
  totalScore?: number;
  hardGateResults: ReviewHardGateResult[];
  scoreItems: ReviewScoreItemResult[];
  mustFixItems: string[];
  suggestions: string[];
  reworkBrief?: ReviewReworkBrief;
};

function verdictBadge(verdict: string) {
  switch (verdict) {
    case "approved":
      return { label: "通过", variant: "success" as const };
    case "rejected":
      return { label: "拒绝", variant: "error" as const };
    case "needs_revision":
      return { label: "需修订", variant: "warning" as const };
    default:
      return { label: verdict, variant: "default" as const };
  }
}

function getReviewArtifacts(report: ReviewReport): ReviewArtifact[] {
  const metadata = asRecord(report.metadata);
  const rawArtifacts = metadata.artifacts;
  if (!Array.isArray(rawArtifacts)) {
    return [];
  }
  return rawArtifacts
    .map((item) => {
      const record = asRecord(item);
      return {
        artifactId: typeof record.artifact_id === "string" ? record.artifact_id : "",
        name: typeof record.name === "string" ? record.name : "未命名工件",
        artifactType: typeof record.artifact_type === "string" ? record.artifact_type : "other",
        versionUri: typeof record.version_uri === "string" ? record.version_uri : "",
        contentPreview:
          typeof record.content_preview === "string" ? record.content_preview : "",
      };
    })
    .filter((item) => item.artifactId || item.versionUri || item.name);
}

function asNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function parseHardGateResults(value: unknown): ReviewHardGateResult[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value
    .map((item) => {
      const record = asRecord(item);
      return {
        key: typeof record.key === "string" ? record.key : "",
        passed: record.passed === true,
        reason: typeof record.reason === "string" ? record.reason : "",
      };
    })
    .filter((item) => item.key);
}

function parseScoreItems(value: unknown): ReviewScoreItemResult[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value
    .map((item) => {
      const record = asRecord(item);
      return {
        key: typeof record.key === "string" ? record.key : "",
        name: typeof record.name === "string" ? record.name : "未命名评分项",
        weight: asNumber(record.weight) ?? 0,
        score: asNumber(record.score) ?? 0,
        max_score: asNumber(record.max_score) ?? 0,
        reason: typeof record.reason === "string" ? record.reason : "",
      };
    })
    .filter((item) => item.key);
}

function getReviewScorecard(report: ReviewReport): ReviewScorecardView | null {
  const metadata = asRecord(report.metadata) as ReviewScorecardMetadata & Record<string, unknown>;
  const templateKey = typeof metadata.template_key === "string" ? metadata.template_key : "";
  const taskCategory = typeof metadata.task_category === "string" ? metadata.task_category : "";
  const hardGateResults = parseHardGateResults(metadata.hard_gate_results);
  const scoreItems = parseScoreItems(metadata.score_items);
  const mustFixItems = asStringArray(metadata.must_fix_items);
  const suggestions = asStringArray(metadata.suggestions);
  const reworkBriefRecord = asRecord(metadata.rework_brief);
  const reworkBrief =
    Object.keys(reworkBriefRecord).length > 0
      ? {
          attempt: metadataNumber(reworkBriefRecord, "attempt") ?? 0,
          failed_hard_gate_keys: asStringArray(reworkBriefRecord.failed_hard_gate_keys),
          low_score_item_keys: asStringArray(reworkBriefRecord.low_score_item_keys),
          must_fix_items: asStringArray(reworkBriefRecord.must_fix_items),
          suggestions: asStringArray(reworkBriefRecord.suggestions),
          requires_clarification: reworkBriefRecord.requires_clarification === true,
        }
      : undefined;

  if (
    !templateKey &&
    !taskCategory &&
    hardGateResults.length === 0 &&
    scoreItems.length === 0 &&
    mustFixItems.length === 0 &&
    !reworkBrief
  ) {
    return null;
  }

  return {
    templateKey,
    taskCategory,
    passThreshold: metadataNumber(metadata, "pass_threshold") ?? undefined,
    totalScore: metadataNumber(metadata, "total_score") ?? undefined,
    hardGateResults,
    scoreItems,
    mustFixItems,
    suggestions,
    reworkBrief,
  };
}

export default function ReviewsPage() {
  const [runId, setRunId] = useState("");
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<ReviewReport[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    setRunId(new URLSearchParams(window.location.search).get("run_id") ?? "");
  }, []);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api.reviews
      .list(runId ? { run_id: runId } : undefined)
      .then((res) => setItems(res.items ?? []))
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [runId]);

  useEffect(() => { load(); }, [load]);

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-orange-50 text-orange-600">
          <FileSearch className="h-5 w-5" />
        </div>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">评审报告</h1>
          <p className="mt-1 text-sm text-gray-500">
            {runId ? `运行 #${runId.slice(0, 8)} 的独立评审记录` : "全局独立评审记录"}
          </p>
        </div>
      </div>

      {loading && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">加载中…</div>
      )}
      {!loading && error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">{error}</div>
      )}
      {!loading && !error && items.length === 0 && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">暂无评审报告</div>
      )}
      {!loading && !error && items.length > 0 && (
        <ul className="space-y-4">
          {items.map((r) => {
            const vb = verdictBadge(r.verdict);
            const findings = asStringArray(r.findings);
            const recommendations = asStringArray(r.recommendations);
            const metadata = asRecord(r.metadata);
            const artifacts = getReviewArtifacts(r);
            const artifactCount = metadataNumber(metadata, "artifact_count") ?? artifacts.length;
            const scorecard = getReviewScorecard(r);
            const passedHardGateCount = scorecard?.hardGateResults.filter((item) => item.passed).length ?? 0;

            return (
              <li key={r.id} className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
                <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                  <div className="space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <h2 className="text-lg font-semibold text-gray-900">{r.summary || "未填写评审摘要"}</h2>
                      <StatusBadge label={vb.label} variant={vb.variant} />
                    </div>
                    <p className="text-sm text-gray-500">报告 ID: {r.id.slice(0, 8)} · 运行: {r.run_id ? r.run_id.slice(0, 8) : "—"}</p>
                  </div>
                  <div className="text-xs text-gray-500">{formatTime(r.created_at)}</div>
                </div>

                <div className="mt-4 grid gap-3 rounded-lg border border-gray-100 bg-gray-50 p-4 text-sm text-gray-700 md:grid-cols-3">
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">任务</p>
                    <p className="mt-1 font-mono text-xs text-gray-700">{r.task_id.slice(0, 8)}</p>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">评审者</p>
                    <p className="mt-1 font-mono text-xs text-gray-700">{r.reviewer_id.slice(0, 8)}</p>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">关联交付物</p>
                    <p className="mt-1 font-medium text-gray-900">{artifactCount} 个</p>
                  </div>
                </div>

                {scorecard && (
                  <section className="mt-4 rounded-lg border border-amber-200 bg-amber-50/40 p-4">
                    <div className="grid gap-3 md:grid-cols-4">
                      <div>
                        <p className="text-xs uppercase tracking-wide text-amber-700">评分模板</p>
                        <p className="mt-1 text-sm font-semibold text-gray-900">
                          {scorecard.templateKey || "未记录"}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs uppercase tracking-wide text-amber-700">总分 / 阈值</p>
                        <p className="mt-1 text-sm font-semibold text-gray-900">
                          {scorecard.totalScore ?? "—"} / {scorecard.passThreshold ?? "—"}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs uppercase tracking-wide text-amber-700">硬门槛</p>
                        <p className="mt-1 text-sm font-semibold text-gray-900">
                          {scorecard.hardGateResults.length > 0
                            ? `${passedHardGateCount}/${scorecard.hardGateResults.length} 通过`
                            : "未记录"}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs uppercase tracking-wide text-amber-700">任务类别</p>
                        <p className="mt-1 text-sm font-semibold text-gray-900">
                          {scorecard.taskCategory || "未记录"}
                        </p>
                      </div>
                    </div>

                    {scorecard.mustFixItems.length > 0 && (
                      <div className="mt-4 rounded-lg border border-red-200 bg-white p-3">
                        <p className="text-xs font-medium uppercase tracking-wide text-red-700">必改项</p>
                        <ul className="mt-2 space-y-2 text-sm text-red-800">
                          {scorecard.mustFixItems.map((item, index) => (
                            <li key={`${r.id}-must-fix-${index}`} className="rounded-md bg-red-50 px-3 py-2">
                              {item}
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}

                    {scorecard.reworkBrief && (
                      <div className="mt-4 rounded-lg border border-gray-200 bg-white p-3">
                        <div className="flex flex-wrap items-center gap-3 text-xs text-gray-500">
                          <span>返工摘要</span>
                          <span>第 {scorecard.reworkBrief.attempt} 次返工</span>
                          {scorecard.reworkBrief.requires_clarification && <span>需要额外澄清</span>}
                        </div>
                        <div className="mt-3 grid gap-3 md:grid-cols-3">
                          <div>
                            <p className="text-xs font-medium uppercase tracking-wide text-gray-500">失败硬门槛</p>
                            <p className="mt-2 text-sm text-gray-700">
                              {scorecard.reworkBrief.failed_hard_gate_keys.length > 0
                                ? scorecard.reworkBrief.failed_hard_gate_keys.join("、")
                                : "无"}
                            </p>
                          </div>
                          <div>
                            <p className="text-xs font-medium uppercase tracking-wide text-gray-500">主要失分项</p>
                            <p className="mt-2 text-sm text-gray-700">
                              {scorecard.reworkBrief.low_score_item_keys.length > 0
                                ? scorecard.reworkBrief.low_score_item_keys.join("、")
                                : "无"}
                            </p>
                          </div>
                          <div>
                            <p className="text-xs font-medium uppercase tracking-wide text-gray-500">建议项</p>
                            <p className="mt-2 text-sm text-gray-700">
                              {scorecard.reworkBrief.suggestions.length > 0
                                ? scorecard.reworkBrief.suggestions.join("；")
                                : "无"}
                            </p>
                          </div>
                        </div>
                      </div>
                    )}

                    {scorecard.hardGateResults.length > 0 && (
                      <div className="mt-4">
                        <p className="text-sm font-semibold text-gray-900">硬门槛结果</p>
                        <div className="mt-3 grid gap-3 md:grid-cols-2">
                          {scorecard.hardGateResults.map((item) => (
                            <div
                              key={`${r.id}-hard-gate-${item.key}`}
                              className={`rounded-lg border p-3 ${
                                item.passed ? "border-green-200 bg-white" : "border-red-200 bg-white"
                              }`}
                            >
                              <div className="flex items-center justify-between gap-3">
                                <p className="text-sm font-medium text-gray-900">{item.key}</p>
                                <span
                                  className={`rounded-full px-2 py-0.5 text-xs ${
                                    item.passed
                                      ? "bg-green-50 text-green-700"
                                      : "bg-red-50 text-red-700"
                                  }`}
                                >
                                  {item.passed ? "通过" : "失败"}
                                </span>
                              </div>
                              <p className="mt-2 text-sm text-gray-600">{item.reason || "未记录原因"}</p>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {scorecard.scoreItems.length > 0 && (
                      <div className="mt-4">
                        <p className="text-sm font-semibold text-gray-900">分项得分</p>
                        <div className="mt-3 grid gap-3 md:grid-cols-2">
                          {scorecard.scoreItems.map((item) => (
                            <div key={`${r.id}-score-item-${item.key}`} className="rounded-lg border border-gray-200 bg-white p-3">
                              <div className="flex items-center justify-between gap-3">
                                <p className="text-sm font-medium text-gray-900">{item.name}</p>
                                <span className="text-xs text-gray-500">
                                  {item.score}/{item.max_score} · 权重 {item.weight}
                                </span>
                              </div>
                              <p className="mt-2 text-sm text-gray-600">{item.reason || "未记录评分理由"}</p>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </section>
                )}

                <div className="mt-4 grid gap-4 md:grid-cols-2">
                  <section className="rounded-lg border border-gray-200 p-4">
                    <p className="text-sm font-semibold text-gray-900">主要发现</p>
                    {findings.length > 0 ? (
                      <ul className="mt-3 space-y-2 text-sm text-gray-700">
                        {findings.map((item, index) => (
                          <li key={`${r.id}-finding-${index}`} className="rounded-md bg-gray-50 px-3 py-2">
                            {item}
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p className="mt-3 text-sm text-gray-500">未记录问题项。</p>
                    )}
                  </section>

                  <section className="rounded-lg border border-gray-200 p-4">
                    <p className="text-sm font-semibold text-gray-900">整改建议</p>
                    {recommendations.length > 0 ? (
                      <ul className="mt-3 space-y-2 text-sm text-gray-700">
                        {recommendations.map((item, index) => (
                          <li key={`${r.id}-recommendation-${index}`} className="rounded-md bg-gray-50 px-3 py-2">
                            {item}
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p className="mt-3 text-sm text-gray-500">未记录建议项。</p>
                    )}
                  </section>
                </div>

                <section className="mt-4 rounded-lg border border-gray-200 p-4">
                  <p className="text-sm font-semibold text-gray-900">评审所见交付物</p>
                  {artifacts.length > 0 ? (
                    <div className="mt-3 space-y-3">
                      {artifacts.map((artifact, index) => (
                        <div key={`${r.id}-artifact-${index}`} className="rounded-lg border border-gray-100 bg-gray-50 p-3">
                          <div className="flex flex-wrap items-center gap-2 text-sm">
                            <span className="font-medium text-gray-900">{artifact.name}</span>
                            <span className="rounded-full bg-white px-2 py-0.5 text-xs text-gray-600">
                              {artifact.artifactType}
                            </span>
                          </div>
                          <p className="mt-2 break-all text-xs text-gray-500">{artifact.versionUri || "未记录版本 URI"}</p>
                          {artifact.contentPreview && (
                            <pre className="mt-3 whitespace-pre-wrap break-words rounded-md bg-gray-950 px-3 py-2 text-xs leading-6 text-gray-100">
                              {truncateText(artifact.contentPreview, 600)}
                            </pre>
                          )}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="mt-3 text-sm text-gray-500">本次评审未记录交付物上下文。</p>
                  )}
                </section>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
