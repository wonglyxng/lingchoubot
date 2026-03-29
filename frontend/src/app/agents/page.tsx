"use client";

import { useEffect, useMemo, useState } from "react";
import { Bot, ChevronRight } from "lucide-react";
import { api } from "@/lib/api";
import type { Agent } from "@/lib/types";
import { getAgentRole } from "@/lib/utils";

function roleColorClass(role: string): string {
  switch (role) {
    case "pm":
      return "text-blue-700 bg-blue-50 ring-blue-200";
    case "supervisor":
      return "text-purple-700 bg-purple-50 ring-purple-200";
    case "worker":
      return "text-green-700 bg-green-50 ring-green-200";
    case "reviewer":
      return "text-orange-700 bg-orange-50 ring-orange-200";
    default:
      return "text-gray-700 bg-gray-100 ring-gray-200";
  }
}

function buildDisplayOrder(agents: Agent[]): { agent: Agent; depth: number }[] {
  const byId = new Map(agents.map((a) => [a.id, a]));
  const children = new Map<string, Agent[]>();

  for (const a of agents) {
    const pid = a.reports_to;
    if (pid && byId.has(pid)) {
      if (!children.has(pid)) children.set(pid, []);
      children.get(pid)!.push(a);
    }
  }

  for (const [, arr] of children) {
    arr.sort((x, y) => x.name.localeCompare(y.name, "zh-CN"));
  }

  const roots = agents.filter((a) => !a.reports_to || !byId.has(a.reports_to));
  roots.sort((x, y) => x.name.localeCompare(y.name, "zh-CN"));

  const depthMemo = new Map<string, number>();

  function depthOf(id: string, stack: Set<string>): number {
    if (depthMemo.has(id)) return depthMemo.get(id)!;
    if (stack.has(id)) {
      depthMemo.set(id, 0);
      return 0;
    }
    const a = byId.get(id);
    if (!a || !a.reports_to || !byId.has(a.reports_to)) {
      depthMemo.set(id, 0);
      return 0;
    }
    stack.add(id);
    const d = 1 + depthOf(a.reports_to, stack);
    stack.delete(id);
    depthMemo.set(id, d);
    return d;
  }

  for (const a of agents) {
    depthOf(a.id, new Set());
  }

  const ordered: { agent: Agent; depth: number }[] = [];
  const seen = new Set<string>();

  function walk(node: Agent, depth: number) {
    if (seen.has(node.id)) return;
    seen.add(node.id);
    ordered.push({ agent: node, depth });
    const ch = children.get(node.id) || [];
    for (const c of ch) walk(c, depth + 1);
  }

  for (const r of roots) walk(r, 0);

  for (const a of agents) {
    if (!seen.has(a.id)) {
      const d = depthMemo.get(a.id) ?? 0;
      ordered.push({ agent: a, depth: d });
    }
  }

  return ordered;
}

export default function AgentsPage() {
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<Agent[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    api.agents
      .orgTree()
      .then((list) => {
        if (!cancelled) setItems(list);
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

  const rows = useMemo(() => buildDisplayOrder(items), [items]);

  return (
    <div className="min-h-full bg-gray-50 p-6">
      <div className="mb-6 flex items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-700">
          <Bot className="h-5 w-5" />
        </div>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Agent 组织树</h1>
          <p className="mt-1 text-sm text-gray-500">按汇报关系展示层级</p>
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
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">
          暂无 Agent
        </div>
      )}

      {!loading && !error && items.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <ul className="space-y-1">
            {rows.map(({ agent, depth }) => (
              <li key={agent.id}>
                <div
                  className="flex gap-2 rounded-md border border-gray-200 bg-gray-50/50 py-2.5 pr-3"
                  style={{ paddingLeft: `${12 + depth * 20}px` }}
                >
                  {depth > 0 ? (
                    <span className="flex shrink-0 items-center text-gray-400">
                      <ChevronRight className="h-4 w-4" aria-hidden />
                    </span>
                  ) : null}
                  <span className="flex shrink-0 items-center text-gray-500">
                    <Bot className="h-4 w-4" aria-hidden />
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium text-gray-900">
                        {agent.name}
                      </span>
                      <span
                        className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${roleColorClass(agent.role)}`}
                      >
                        {getAgentRole(agent.role)}
                      </span>
                      <span className="text-xs text-gray-500">
                        {agent.status}
                      </span>
                    </div>
                    {agent.description ? (
                      <p className="mt-1 text-sm text-gray-600 line-clamp-2">
                        {agent.description}
                      </p>
                    ) : null}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
