"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  FolderKanban,
  ListChecks,
  Bot,
  Package,
  ShieldCheck,
  ScrollText,
  Activity,
  FileSearch,
  ArrowRightLeft,
  Wrench,
  FileText,
  BrainCircuit,
} from "lucide-react";

const navItems = [
  { href: "/", label: "概览", icon: LayoutDashboard },
  { href: "/projects", label: "项目", icon: FolderKanban },
  { href: "/tasks", label: "任务", icon: ListChecks },
  { href: "/agents", label: "Agent", icon: Bot },
  { href: "/workflows", label: "工作流", icon: Activity },
  { href: "/artifacts", label: "工件", icon: Package },
  { href: "/reviews", label: "评审", icon: FileSearch },
  { href: "/handoffs", label: "交接", icon: ArrowRightLeft },
  { href: "/tool-calls", label: "工具调用", icon: Wrench },
  { href: "/contracts", label: "任务契约", icon: FileText },
  { href: "/approvals", label: "审批", icon: ShieldCheck },
  { href: "/model-config", label: "模型配置", icon: BrainCircuit },
  { href: "/audit", label: "审计", icon: ScrollText },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="flex w-56 shrink-0 flex-col border-r border-gray-200 bg-white">
      <div className="flex h-14 items-center gap-2 border-b border-gray-200 px-4">
        <LayoutDashboard className="h-6 w-6 text-blue-600" />
        <span className="text-lg font-semibold text-gray-900">灵筹</span>
        <span className="ml-1 rounded bg-blue-100 px-1.5 py-0.5 text-[10px] font-medium text-blue-700">
          MVP
        </span>
      </div>

      <nav className="flex-1 space-y-1 px-2 py-3">
        {navItems.map((item) => {
          const active =
            item.href === "/"
              ? pathname === "/"
              : pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                active
                  ? "bg-blue-50 text-blue-700"
                  : "text-gray-600 hover:bg-gray-50 hover:text-gray-900"
              }`}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              {item.label}
            </Link>
          );
        })}
      </nav>

      <div className="border-t border-gray-200 px-4 py-3 text-xs text-gray-500">
        v0.1.0 · 控制台
      </div>
    </aside>
  );
}
