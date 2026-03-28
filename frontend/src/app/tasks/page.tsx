import { ListChecks } from "lucide-react";

export default function TasksPage() {
  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">任务看板</h1>
        <p className="mt-1 text-sm text-gray-500">
          按状态查看与筛选任务
        </p>
      </div>

      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 bg-white py-16 text-center">
        <ListChecks className="mb-3 h-10 w-10 text-gray-300" />
        <p className="text-sm text-gray-500">暂无任务数据</p>
        <p className="mt-1 max-w-sm text-xs text-gray-400">
          任务 API 接入后可在此做分栏看板与筛选。
        </p>
      </div>
    </div>
  );
}
