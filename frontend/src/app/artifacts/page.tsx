"use client";

import { useEffect, useState } from "react";
import { FileText, Package } from "lucide-react";
import { api } from "@/lib/api";
import type { Artifact, ArtifactVersion } from "@/lib/types";
import { asRecord, formatTime, metadataString, truncateText } from "@/lib/utils";

const ARTIFACT_TYPE_LABELS: Record<string, string> = {
  prd: "需求文档",
  design: "设计文档",
  api_spec: "API规范",
  schema_sql: "数据库Schema",
  source_code: "源代码",
  test_report: "测试报告",
  deployment_plan: "部署计划",
  release_note: "发布说明",
  other: "其他",
};

function artifactTypeLabel(type: string): string {
  return ARTIFACT_TYPE_LABELS[type] ?? type;
}

export default function ArtifactsPage() {
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<Artifact[]>([]);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [versionsByArtifact, setVersionsByArtifact] = useState<Record<string, ArtifactVersion[]>>({});
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    api.artifacts
      .list()
      .then((res) => {
        if (!cancelled) setItems(res.items);
      })
      .catch((e: Error) => {
        if (!cancelled) setError(e.message || "加载失败");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (items.length === 0) {
      setVersionsByArtifact({});
      return;
    }
    let cancelled = false;
    setVersionsLoading(true);
    Promise.all(
      items.map(async (artifact) => {
        const versions = await api.artifacts.versions(artifact.id);
        return [artifact.id, versions] as const;
      }),
    )
      .then((entries) => {
        if (cancelled) return;
        setVersionsByArtifact(Object.fromEntries(entries));
      })
      .catch((e: Error) => {
        if (!cancelled) setError(e.message || "加载工件版本失败");
      })
      .finally(() => {
        if (!cancelled) setVersionsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [items]);

  return (
    <div className="min-h-full bg-gray-50 p-6">
      <div className="mb-6 flex items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-700">
          <Package className="h-5 w-5" />
        </div>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">工件</h1>
          <p className="mt-1 text-sm text-gray-500">按项目产出的工件列表</p>
        </div>
      </div>

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
        <div className="flex flex-col items-center justify-center rounded-lg border border-gray-200 bg-white py-16 text-center">
          <Package className="mb-3 h-10 w-10 text-gray-300" />
          <p className="text-sm text-gray-500">暂无工件</p>
        </div>
      )}

      {!loading && !error && items.length > 0 && (
        <div className="space-y-4">
          {versionsLoading && (
            <div className="rounded-lg border border-gray-200 bg-white px-5 py-3 text-sm text-gray-500">
              正在加载工件版本与正文预览…
            </div>
          )}
          {items.map((row) => {
            const versions = versionsByArtifact[row.id] ?? [];
            const latest = versions[0];
            const versionMeta = asRecord(latest?.metadata);
            const preview = metadataString(versionMeta, "inline_content");
            const sourceName = metadataString(versionMeta, "source_name");
            const storedIn = metadataString(versionMeta, "stored_in");

            return (
              <article key={row.id} className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
                <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <h2 className="text-lg font-semibold text-gray-900">{row.name}</h2>
                      <span className="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600">
                        {artifactTypeLabel(row.artifact_type)}
                      </span>
                    </div>
                    <p className="mt-2 text-sm leading-6 text-gray-600">
                      {row.description?.trim() ? row.description : "未填写工件说明。"}
                    </p>
                  </div>
                  <div className="shrink-0 text-xs text-gray-500">{formatTime(row.created_at)}</div>
                </div>

                <div className="mt-4 grid gap-3 rounded-lg border border-gray-100 bg-gray-50 p-4 text-sm text-gray-700 md:grid-cols-2">
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">最新版本</p>
                    <p className="mt-1 font-medium text-gray-900">
                      {latest ? `v${latest.version}` : "暂无版本"}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">内容来源</p>
                    <p className="mt-1 font-medium text-gray-900">
                      {storedIn ? storedIn.toUpperCase() : "未记录"}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">存储 URI</p>
                    <p className="mt-1 break-all text-gray-700">{latest?.uri || "—"}</p>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">源文件名</p>
                    <p className="mt-1 break-all text-gray-700">{sourceName || "—"}</p>
                  </div>
                </div>

                <div className="mt-4 rounded-lg border border-gray-200 bg-gray-950 p-4 text-gray-100">
                  <div className="mb-3 flex items-center gap-2 text-sm font-medium text-gray-200">
                    <FileText className="h-4 w-4" />
                    工件内容预览
                  </div>
                  <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words text-xs leading-6 text-gray-200">
                    {preview ? truncateText(preview, 2400) : "当前版本未提供可直接预览的正文内容。"}
                  </pre>
                </div>
              </article>
            );
          })}
        </div>
      )}
    </div>
  );
}
