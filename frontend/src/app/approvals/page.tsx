import { ShieldCheck } from "lucide-react";

export default function ApprovalsPage() {
  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">审批中心</h1>
        <p className="mt-1 text-sm text-gray-500">
          待办与已处理审批
        </p>
      </div>

      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 bg-white py-16 text-center">
        <ShieldCheck className="mb-3 h-10 w-10 text-gray-300" />
        <p className="text-sm text-gray-500">暂无审批请求</p>
        <p className="mt-1 max-w-sm text-xs text-gray-400">
          审批 API 接入后可在此批准或驳回并填写意见。
        </p>
      </div>
    </div>
  );
}
