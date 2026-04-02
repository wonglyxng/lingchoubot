"use client";

import { useEffect, useMemo, useState, useCallback } from "react";
import { Bot, ChevronRight, Plus, Pencil, Trash2 } from "lucide-react";
import { api } from "@/lib/api";
import type { Agent } from "@/lib/types";
import { getAgentRole, getAgentType, getAgentSpec } from "@/lib/utils";
import {
  type AgentLLMProviderOption,
  agentLLMProviderOptions as fallbackProviderOptions,
  DEFAULT_AGENT_LLM_PROVIDER,
  DEFAULT_AGENT_LLM_MODEL,
  getDefaultModel,
  getModelOptions,
  getProviderLabel,
  isPresetModel,
  mergeAgentLLMMetadata,
  readAgentLLMConfig,
  toProviderOptions,
} from "@/lib/agent-llm";
import { FormModal, FormField, inputClass, textareaClass, selectClass } from "@/components/FormModal";

type AgentFormState = {
  name: string;
  role: string;
  role_code: string;
  description: string;
  agent_type: string;
  specialization: string;
  reports_to: string;
  llm_provider: string;
  llm_model: string;
  metadata: Record<string, unknown>;
};

function deriveRoleCode(role: string, specialization: string): string {
  switch (role) {
    case "pm":
      return "PM_SUPERVISOR";
    case "supervisor":
      return specialization === "qa" ? "QA_SUPERVISOR" : "DEVELOPMENT_SUPERVISOR";
    case "worker":
      switch (specialization) {
        case "frontend":
          return "FRONTEND_DEV_WORKER";
        case "qa":
          return "QA_WORKER";
        default:
          return "BACKEND_DEV_WORKER";
      }
    case "reviewer":
      return "REVIEWER_WORKER";
    default:
      return "";
  }
}

function createEmptyAgentForm(): AgentFormState {
  return {
    name: "",
    role: "worker",
    role_code: deriveRoleCode("worker", "general"),
    description: "",
    agent_type: "llm",
    specialization: "general",
    reports_to: "",
    llm_provider: DEFAULT_AGENT_LLM_PROVIDER,
    llm_model: DEFAULT_AGENT_LLM_MODEL,
    metadata: {},
  };
}

function buildAgentPayload(form: AgentFormState) {
  return {
    name: form.name.trim(),
    role: form.role,
    role_code: form.role_code || deriveRoleCode(form.role, form.specialization),
    description: form.description.trim(),
    agent_type: form.agent_type,
    specialization: form.specialization,
    reports_to: form.reports_to || undefined,
    metadata: mergeAgentLLMMetadata(form.metadata, form.agent_type, {
      provider: form.llm_provider,
      model: form.llm_model.trim(),
    }),
  };
}

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
  const [showCreate, setShowCreate] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form, setForm] = useState<AgentFormState>(createEmptyAgentForm());
  const [editTarget, setEditTarget] = useState<Agent | null>(null);
  const [editForm, setEditForm] = useState<AgentFormState>(createEmptyAgentForm());
  const [providerOptions, setProviderOptions] = useState<AgentLLMProviderOption[]>(fallbackProviderOptions);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    Promise.all([
      api.agents.orgTree(),
      api.llmProviders.list(true).then((res) => {
        const opts = toProviderOptions(Array.isArray(res.items) ? res.items : []);
        if (opts.length > 0) setProviderOptions(opts);
      }).catch(() => { /* keep fallback */ }),
    ])
      .then(([list]) => setItems(Array.isArray(list) ? list : []))
      .catch((e: Error) => setError(e.message || "加载失败"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    if (!form.name.trim()) return;
    setSubmitting(true);
    try {
      await api.agents.create(buildAgentPayload(form));
      setShowCreate(false);
      setForm(createEmptyAgentForm());
      load();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "创建失败";
      alert(msg);
    } finally {
      setSubmitting(false);
    }
  };

  const openEdit = (a: Agent) => {
    const llm = readAgentLLMConfig(a.metadata);
    setEditTarget(a);
    setEditForm({
      name: a.name,
      role: a.role,
      role_code: a.role_code || deriveRoleCode(a.role, a.specialization),
      description: a.description,
      agent_type: a.agent_type || "llm",
      specialization: a.specialization,
      reports_to: a.reports_to || "",
      llm_provider: llm.provider,
      llm_model: llm.model,
      metadata: a.metadata || {},
    });
  };

  const handleEdit = async () => {
    if (!editTarget || !editForm.name.trim()) return;
    setSubmitting(true);
    try {
      await api.agents.update(editTarget.id, buildAgentPayload(editForm));
      setEditTarget(null);
      load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "更新失败");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (a: Agent) => {
    if (!confirm(`确定删除 Agent「${a.name}」？`)) return;
    try {
      await api.agents.delete(a.id);
      load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "删除失败");
    }
  };

  const rows = useMemo(() => buildDisplayOrder(items), [items]);
  const createModelOptions = getModelOptions(providerOptions, form.llm_provider);
  const editModelOptions = getModelOptions(providerOptions, editForm.llm_provider);

  return (
    <div className="min-h-full bg-gray-50 p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-700">
            <Bot className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">Agent 组织树</h1>
            <p className="mt-1 text-sm text-gray-500">按汇报关系展示层级</p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => setShowCreate(true)}
          className="inline-flex items-center gap-2 rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700"
        >
          <Plus className="h-4 w-4" />
          新建 Agent
        </button>
      </div>

      <FormModal
        open={showCreate}
        onClose={() => setShowCreate(false)}
        title="新建 Agent"
        onSubmit={handleCreate}
        submitting={submitting}
      >
        <FormField label="名称" required>
          <input
            className={inputClass}
            value={form.name}
            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            placeholder="例如：后端执行者-01"
            maxLength={100}
            required
          />
        </FormField>
        <div className="grid grid-cols-2 gap-4">
          <FormField label="角色" required>
            <select className={selectClass} value={form.role} onChange={(e) => setForm((f) => {
              const role = e.target.value;
              return { ...f, role, role_code: deriveRoleCode(role, f.specialization) };
            })}>
              <option value="pm">项目经理</option>
              <option value="supervisor">主管</option>
              <option value="worker">执行者</option>
              <option value="reviewer">评审者</option>
            </select>
          </FormField>
          <FormField label="类型">
            <select className={selectClass} value={form.agent_type} onChange={(e) => setForm((f) => ({ ...f, agent_type: e.target.value }))}>
              <option value="llm">LLM</option>
              <option value="human">人工</option>
            </select>
          </FormField>
        </div>
        {form.agent_type === "llm" && (
          <>
            <div className="grid grid-cols-2 gap-4">
              <FormField label="LLM 提供商" required>
                <select
                  className={selectClass}
                  value={form.llm_provider}
                  onChange={(e) => setForm((f) => ({
                    ...f,
                    llm_provider: e.target.value,
                    llm_model: getDefaultModel(providerOptions, e.target.value),
                  }))}
                >
                  {providerOptions.map((option) => (
                    <option key={option.value} value={option.value}>{option.label}</option>
                  ))}
                </select>
              </FormField>
              <FormField label="模型预设">
                <select
                  className={selectClass}
                  value={isPresetModel(providerOptions, form.llm_provider, form.llm_model) ? form.llm_model : "__custom__"}
                  onChange={(e) => setForm((f) => ({
                    ...f,
                    llm_model: e.target.value === "__custom__" ? f.llm_model : e.target.value,
                  }))}
                >
                  {createModelOptions.map((option) => (
                    <option key={option.value} value={option.value}>{option.label}</option>
                  ))}
                  <option value="__custom__">自定义模型</option>
                </select>
              </FormField>
            </div>
            <FormField label="模型 ID" required>
              <input
                className={inputClass}
                value={form.llm_model}
                onChange={(e) => setForm((f) => ({ ...f, llm_model: e.target.value }))}
                placeholder="例如：gpt-4.1-mini"
                maxLength={120}
                required
              />
            </FormField>
          </>
        )}
        <FormField label="专长">
          <select className={selectClass} value={form.specialization} onChange={(e) => setForm((f) => {
            const specialization = e.target.value;
            return { ...f, specialization, role_code: deriveRoleCode(f.role, specialization) };
          })}>
            <option value="general">通用</option>
            <option value="backend">后端</option>
            <option value="frontend">前端</option>
            <option value="qa">测试</option>
            <option value="release">发布</option>
            <option value="devops">运维</option>
            <option value="design">设计</option>
          </select>
        </FormField>
        <FormField label="上级 Agent ID">
          <select className={selectClass} value={form.reports_to} onChange={(e) => setForm((f) => ({ ...f, reports_to: e.target.value }))}>
            <option value="">无（顶层）</option>
            {items.map((a) => (
              <option key={a.id} value={a.id}>{a.name} ({getAgentRole(a.role)})</option>
            ))}
          </select>
        </FormField>
        <FormField label="描述">
          <textarea
            className={textareaClass}
            rows={2}
            value={form.description}
            onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
            placeholder="Agent 简要描述"
            maxLength={1000}
          />
        </FormField>
      </FormModal>

      <FormModal
        open={!!editTarget}
        onClose={() => setEditTarget(null)}
        title="编辑 Agent"
        onSubmit={handleEdit}
        submitting={submitting}
      >
        <FormField label="名称" required>
          <input className={inputClass} value={editForm.name} onChange={(e) => setEditForm((f) => ({ ...f, name: e.target.value }))} maxLength={100} required />
        </FormField>
        <div className="grid grid-cols-2 gap-4">
          <FormField label="角色" required>
            <select className={selectClass} value={editForm.role} onChange={(e) => setEditForm((f) => {
              const role = e.target.value;
              return { ...f, role, role_code: deriveRoleCode(role, f.specialization) };
            })}>
              <option value="pm">项目经理</option><option value="supervisor">主管</option><option value="worker">执行者</option><option value="reviewer">评审者</option>
            </select>
          </FormField>
          <FormField label="类型">
            <select className={selectClass} value={editForm.agent_type} onChange={(e) => setEditForm((f) => ({ ...f, agent_type: e.target.value }))}>
              <option value="llm">LLM</option><option value="human">人工</option>
              {editForm.agent_type === "mock" && <option value="mock">模拟（兼容旧数据）</option>}
            </select>
          </FormField>
        </div>
        {editForm.agent_type === "llm" && (
          <>
            <div className="grid grid-cols-2 gap-4">
              <FormField label="LLM 提供商" required>
                <select
                  className={selectClass}
                  value={editForm.llm_provider}
                  onChange={(e) => setEditForm((f) => ({
                    ...f,
                    llm_provider: e.target.value,
                    llm_model: getDefaultModel(providerOptions, e.target.value),
                  }))}
                >
                  {providerOptions.map((option) => (
                    <option key={option.value} value={option.value}>{option.label}</option>
                  ))}
                </select>
              </FormField>
              <FormField label="模型预设">
                <select
                  className={selectClass}
                  value={isPresetModel(providerOptions, editForm.llm_provider, editForm.llm_model) ? editForm.llm_model : "__custom__"}
                  onChange={(e) => setEditForm((f) => ({
                    ...f,
                    llm_model: e.target.value === "__custom__" ? f.llm_model : e.target.value,
                  }))}
                >
                  {editModelOptions.map((option) => (
                    <option key={option.value} value={option.value}>{option.label}</option>
                  ))}
                  <option value="__custom__">自定义模型</option>
                </select>
              </FormField>
            </div>
            <FormField label="模型 ID" required>
              <input
                className={inputClass}
                value={editForm.llm_model}
                onChange={(e) => setEditForm((f) => ({ ...f, llm_model: e.target.value }))}
                maxLength={120}
                required
              />
            </FormField>
          </>
        )}
        <FormField label="专长">
          <select className={selectClass} value={editForm.specialization} onChange={(e) => setEditForm((f) => {
            const specialization = e.target.value;
            return { ...f, specialization, role_code: deriveRoleCode(f.role, specialization) };
          })}>
            <option value="general">通用</option><option value="backend">后端</option><option value="frontend">前端</option>
            <option value="qa">测试</option><option value="release">发布</option><option value="devops">运维</option><option value="design">设计</option>
          </select>
        </FormField>
        <FormField label="上级">
          <select className={selectClass} value={editForm.reports_to} onChange={(e) => setEditForm((f) => ({ ...f, reports_to: e.target.value }))}>
            <option value="">无（顶层）</option>
            {items.filter((a) => a.id !== editTarget?.id).map((a) => (
              <option key={a.id} value={a.id}>{a.name} ({getAgentRole(a.role)})</option>
            ))}
          </select>
        </FormField>
        <FormField label="描述">
          <textarea className={textareaClass} rows={2} value={editForm.description} onChange={(e) => setEditForm((f) => ({ ...f, description: e.target.value }))} maxLength={1000} />
        </FormField>
      </FormModal>

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
            {rows.map(({ agent, depth }) => {
              const llm = readAgentLLMConfig(agent.metadata);
              return <li key={agent.id}>
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
                      {agent.agent_type && (
                        <span className="inline-flex rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 ring-1 ring-inset ring-gray-200">
                          {getAgentType(agent.agent_type)}
                        </span>
                      )}
                      {agent.agent_type === "llm" && llm.model && (
                        <span className="inline-flex rounded-full bg-sky-50 px-2 py-0.5 text-xs text-sky-700 ring-1 ring-inset ring-sky-200">
                          {getProviderLabel(providerOptions, llm.provider)} / {llm.model}
                        </span>
                      )}
                      {agent.specialization && agent.specialization !== "general" && (
                        <span className="inline-flex rounded-full bg-indigo-50 px-2 py-0.5 text-xs text-indigo-600 ring-1 ring-inset ring-indigo-200">
                          {getAgentSpec(agent.specialization)}
                        </span>
                      )}
                      <span className="text-xs text-gray-500">
                        {agent.status}
                      </span>
                      <span className="ml-auto flex gap-1">
                        <button onClick={() => openEdit(agent)} className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600" title="编辑">
                          <Pencil className="h-3.5 w-3.5" />
                        </button>
                        <button onClick={() => handleDelete(agent)} className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600" title="删除">
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </span>
                    </div>
                    {agent.description ? (
                      <p className="mt-1 text-sm text-gray-600 line-clamp-2">
                        {agent.description}
                      </p>
                    ) : null}
                  </div>
                </div>
              </li>;
            })}
          </ul>
        </div>
      )}
    </div>
  );
}
