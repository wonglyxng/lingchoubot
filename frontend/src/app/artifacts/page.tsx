"use client";

import { useEffect, useState } from "react";
import { Package } from "lucide-react";
import { api } from "@/lib/api";
import type { Artifact } from "@/lib/types";
import { formatTime } from "@/lib/utils";

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
        <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 text-left text-sm">
              <thead className="bg-gray-50 text-xs font-medium uppercase tracking-wide text-gray-500">
                <tr>
                  <th className="px-4 py-3">名称</th>
                  <th className="px-4 py-3">类型</th>
                  <th className="px-4 py-3">描述</th>
                  <th className="px-4 py-3 whitespace-nowrap">创建时间</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 bg-white text-gray-900">
                {items.map((row) => (
                  <tr key={row.id} className="hover:bg-gray-50/80">
                    <td className="px-4 py-3 font-medium">{row.name}</td>
                    <td className="px-4 py-3 text-gray-600">
                      {artifactTypeLabel(row.artifact_type)}
                    </td>
                    <td className="max-w-md px-4 py-3 text-gray-600">
                      {row.description?.trim() ? row.description : "—"}
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-gray-600">
                      {formatTime(row.created_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
