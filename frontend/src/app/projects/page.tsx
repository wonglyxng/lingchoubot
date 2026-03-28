import { FolderKanban } from "lucide-react";

export default function ProjectsPage() {
  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">项目</h1>
        <p className="mt-1 text-sm text-gray-500">
          管理项目与阶段，接入 API 后在此列表展示
        </p>
      </div>

      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 bg-white py-16 text-center">
        <FolderKanban className="mb-3 h-10 w-10 text-gray-300" />
        <p className="text-sm text-gray-500">暂无项目数据</p>
        <p className="mt-1 max-w-sm text-xs text-gray-400">
          项目 CRUD 接入后，将显示列表与详情入口。
        </p>
      </div>
    </div>
  );
}
