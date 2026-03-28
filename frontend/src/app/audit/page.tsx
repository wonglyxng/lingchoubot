import { ScrollText } from "lucide-react";

export default function AuditPage() {
  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">审计</h1>
        <p className="mt-1 text-sm text-gray-500">
          关键操作时间线
        </p>
      </div>

      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 bg-white py-16 text-center">
        <ScrollText className="mb-3 h-10 w-10 text-gray-300" />
        <p className="text-sm text-gray-500">暂无审计记录</p>
        <p className="mt-1 max-w-sm text-xs text-gray-400">
          审计查询 API 接入后按时间展示事件流。
        </p>
      </div>
    </div>
  );
}
