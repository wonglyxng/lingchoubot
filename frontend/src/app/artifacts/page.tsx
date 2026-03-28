import { Package } from "lucide-react";

export default function ArtifactsPage() {
  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">工件</h1>
        <p className="mt-1 text-sm text-gray-500">
          产物与版本
        </p>
      </div>

      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 bg-white py-16 text-center">
        <Package className="mb-3 h-10 w-10 text-gray-300" />
        <p className="text-sm text-gray-500">暂无工件数据</p>
        <p className="mt-1 max-w-sm text-xs text-gray-400">
          工件与版本 API 接入后在此浏览与追溯。
        </p>
      </div>
    </div>
  );
}
