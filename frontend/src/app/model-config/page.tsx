"use client";

import { useEffect, useState, useCallback } from "react";
import { BrainCircuit, Plus, Pencil, Trash2, ChevronDown, ChevronRight, Power, PowerOff } from "lucide-react";
import { api } from "@/lib/api";
import type { LLMProvider, LLMModel } from "@/lib/types";
import { FormModal, FormField, inputClass, selectClass } from "@/components/FormModal";

/* ---- Provider form state ---- */
interface ProviderForm {
  key: string;
  name: string;
  base_url: string;
  api_key: string;
  is_enabled: boolean;
  sort_order: number;
}

function emptyProviderForm(): ProviderForm {
  return { key: "", name: "", base_url: "", api_key: "", is_enabled: true, sort_order: 0 };
}

/* ---- Model form state ---- */
interface ModelForm {
  model_id: string;
  name: string;
  is_default: boolean;
  sort_order: number;
}

function emptyModelForm(): ModelForm {
  return { model_id: "", name: "", is_default: false, sort_order: 0 };
}

export default function ModelConfigPage() {
  const [loading, setLoading] = useState(true);
  const [providers, setProviders] = useState<LLMProvider[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  /* Provider CRUD state */
  const [showCreate, setShowCreate] = useState(false);
  const [editTarget, setEditTarget] = useState<LLMProvider | null>(null);
  const [providerForm, setProviderForm] = useState<ProviderForm>(emptyProviderForm());
  const [submitting, setSubmitting] = useState(false);

  /* Model CRUD state */
  const [showModelCreate, setShowModelCreate] = useState<string | null>(null); // provider id
  const [editModelTarget, setEditModelTarget] = useState<LLMModel | null>(null);
  const [modelForm, setModelForm] = useState<ModelForm>(emptyModelForm());

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api.llmProviders
      .list()
      .then((res) => setProviders(Array.isArray(res.items) ? res.items : []))
      .catch((e: Error) => setError(e.message || "加载失败"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  /* ---- Provider handlers ---- */
  const handleCreateProvider = async () => {
    if (!providerForm.key.trim() || !providerForm.name.trim() || !providerForm.base_url.trim()) return;
    setSubmitting(true);
    try {
      await api.llmProviders.create(providerForm);
      setShowCreate(false);
      setProviderForm(emptyProviderForm());
      load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "创建失败");
    } finally {
      setSubmitting(false);
    }
  };

  const openEditProvider = (p: LLMProvider) => {
    setEditTarget(p);
    setProviderForm({
      key: p.key,
      name: p.name,
      base_url: p.base_url,
      api_key: p.api_key || "",
      is_enabled: p.is_enabled,
      sort_order: p.sort_order,
    });
  };

  const handleEditProvider = async () => {
    if (!editTarget) return;
    setSubmitting(true);
    try {
      await api.llmProviders.update(editTarget.id, providerForm);
      setEditTarget(null);
      load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "更新失败");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteProvider = async (p: LLMProvider) => {
    if (!confirm(`确定删除供应商「${p.name}」及其所有模型预设？`)) return;
    try {
      await api.llmProviders.delete(p.id);
      load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "删除失败");
    }
  };

  /* ---- Model handlers ---- */
  const handleCreateModel = async () => {
    if (!showModelCreate || !modelForm.model_id.trim() || !modelForm.name.trim()) return;
    setSubmitting(true);
    try {
      await api.llmProviders.createModel(showModelCreate, modelForm);
      setShowModelCreate(null);
      setModelForm(emptyModelForm());
      load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "创建失败");
    } finally {
      setSubmitting(false);
    }
  };

  const openEditModel = (m: LLMModel) => {
    setEditModelTarget(m);
    setModelForm({
      model_id: m.model_id,
      name: m.name,
      is_default: m.is_default,
      sort_order: m.sort_order,
    });
  };

  const handleEditModel = async () => {
    if (!editModelTarget) return;
    setSubmitting(true);
    try {
      await api.llmProviders.updateModel(editModelTarget.id, {
        ...modelForm,
        provider_id: editModelTarget.provider_id,
      });
      setEditModelTarget(null);
      load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "更新失败");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteModel = async (m: LLMModel) => {
    if (!confirm(`确定删除模型预设「${m.name}」？`)) return;
    try {
      await api.llmProviders.deleteModel(m.id);
      load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "删除失败");
    }
  };

  /* ---- Render helpers ---- */
  const providerFormFields = (
    form: ProviderForm,
    setForm: React.Dispatch<React.SetStateAction<ProviderForm>>,
    isEdit: boolean,
  ) => (
    <>
      <div className="grid grid-cols-2 gap-4">
        <FormField label="标识 Key" required>
          <input
            className={inputClass}
            value={form.key}
            onChange={(e) => setForm((f) => ({ ...f, key: e.target.value }))}
            placeholder="例如：openai"
            maxLength={50}
            required
            disabled={isEdit}
          />
        </FormField>
        <FormField label="显示名称" required>
          <input
            className={inputClass}
            value={form.name}
            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            placeholder="例如：OpenAI"
            maxLength={100}
            required
          />
        </FormField>
      </div>
      <FormField label="Base URL" required>
        <input
          className={inputClass}
          value={form.base_url}
          onChange={(e) => setForm((f) => ({ ...f, base_url: e.target.value }))}
          placeholder="例如：https://api.openai.com/v1"
          maxLength={500}
          required
        />
      </FormField>
      <FormField label="API Key">
        <input
          className={inputClass}
          value={form.api_key}
          onChange={(e) => setForm((f) => ({ ...f, api_key: e.target.value }))}
          placeholder={isEdit ? "留空保持不变，或输入新密钥" : "可选"}
          maxLength={500}
          type="password"
          autoComplete="off"
        />
      </FormField>
      <div className="grid grid-cols-2 gap-4">
        <FormField label="排序">
          <input
            className={inputClass}
            type="number"
            value={form.sort_order}
            onChange={(e) => setForm((f) => ({ ...f, sort_order: parseInt(e.target.value) || 0 }))}
          />
        </FormField>
        <FormField label="状态">
          <select
            className={selectClass}
            value={form.is_enabled ? "true" : "false"}
            onChange={(e) => setForm((f) => ({ ...f, is_enabled: e.target.value === "true" }))}
          >
            <option value="true">启用</option>
            <option value="false">禁用</option>
          </select>
        </FormField>
      </div>
    </>
  );

  const modelFormFields = (
    form: ModelForm,
    setForm: React.Dispatch<React.SetStateAction<ModelForm>>,
  ) => (
    <>
      <div className="grid grid-cols-2 gap-4">
        <FormField label="模型 ID" required>
          <input
            className={inputClass}
            value={form.model_id}
            onChange={(e) => setForm((f) => ({ ...f, model_id: e.target.value }))}
            placeholder="例如：gpt-4.1-mini"
            maxLength={120}
            required
          />
        </FormField>
        <FormField label="显示名称" required>
          <input
            className={inputClass}
            value={form.name}
            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            placeholder="例如：GPT-4.1 Mini"
            maxLength={100}
            required
          />
        </FormField>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <FormField label="排序">
          <input
            className={inputClass}
            type="number"
            value={form.sort_order}
            onChange={(e) => setForm((f) => ({ ...f, sort_order: parseInt(e.target.value) || 0 }))}
          />
        </FormField>
        <FormField label="默认模型">
          <select
            className={selectClass}
            value={form.is_default ? "true" : "false"}
            onChange={(e) => setForm((f) => ({ ...f, is_default: e.target.value === "true" }))}
          >
            <option value="false">否</option>
            <option value="true">是</option>
          </select>
        </FormField>
      </div>
    </>
  );

  return (
    <div className="min-h-full bg-gray-50 p-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-700">
            <BrainCircuit className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">模型配置</h1>
            <p className="mt-1 text-sm text-gray-500">管理 LLM 供应商和模型预设</p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => { setProviderForm(emptyProviderForm()); setShowCreate(true); }}
          className="inline-flex items-center gap-2 rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700"
        >
          <Plus className="h-4 w-4" />
          添加供应商
        </button>
      </div>

      {/* Create Provider Modal */}
      <FormModal open={showCreate} onClose={() => setShowCreate(false)} title="添加供应商" onSubmit={handleCreateProvider} submitting={submitting}>
        {providerFormFields(providerForm, setProviderForm, false)}
      </FormModal>

      {/* Edit Provider Modal */}
      <FormModal open={!!editTarget} onClose={() => setEditTarget(null)} title="编辑供应商" onSubmit={handleEditProvider} submitting={submitting}>
        {providerFormFields(providerForm, setProviderForm, true)}
      </FormModal>

      {/* Create Model Modal */}
      <FormModal open={!!showModelCreate} onClose={() => setShowModelCreate(null)} title="添加模型预设" onSubmit={handleCreateModel} submitting={submitting}>
        {modelFormFields(modelForm, setModelForm)}
      </FormModal>

      {/* Edit Model Modal */}
      <FormModal open={!!editModelTarget} onClose={() => setEditModelTarget(null)} title="编辑模型预设" onSubmit={handleEditModel} submitting={submitting}>
        {modelFormFields(modelForm, setModelForm)}
      </FormModal>

      {/* Content */}
      {loading && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">加载中…</div>
      )}

      {!loading && error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">{error}</div>
      )}

      {!loading && !error && providers.length === 0 && (
        <div className="rounded-lg border border-gray-200 bg-white px-5 py-12 text-center text-sm text-gray-500">暂无供应商配置</div>
      )}

      {!loading && !error && providers.length > 0 && (
        <div className="space-y-3">
          {providers.map((p) => {
            const expanded = expandedId === p.id;
            const models = p.models || [];
            return (
              <div key={p.id} className="rounded-lg border border-gray-200 bg-white">
                {/* Provider header row */}
                <div
                  className="flex cursor-pointer items-center gap-3 px-4 py-3 hover:bg-gray-50"
                  onClick={() => setExpandedId(expanded ? null : p.id)}
                >
                  {expanded
                    ? <ChevronDown className="h-4 w-4 text-gray-400" />
                    : <ChevronRight className="h-4 w-4 text-gray-400" />}
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium text-gray-900">{p.name}</span>
                      <span className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-500 font-mono">{p.key}</span>
                      {p.is_builtin && (
                        <span className="rounded bg-blue-50 px-1.5 py-0.5 text-xs text-blue-600 ring-1 ring-inset ring-blue-200">内置</span>
                      )}
                      {p.is_enabled
                        ? <span className="inline-flex items-center gap-1 text-xs text-green-600"><Power className="h-3 w-3" />启用</span>
                        : <span className="inline-flex items-center gap-1 text-xs text-gray-400"><PowerOff className="h-3 w-3" />禁用</span>}
                      <span className="text-xs text-gray-400">{models.length} 个模型</span>
                    </div>
                    <div className="mt-1 text-xs text-gray-400 truncate">{p.base_url}</div>
                  </div>
                  <div className="flex shrink-0 items-center gap-1" onClick={(e) => e.stopPropagation()}>
                    <button onClick={() => openEditProvider(p)} className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600" title="编辑">
                      <Pencil className="h-3.5 w-3.5" />
                    </button>
                    <button onClick={() => handleDeleteProvider(p)} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600" title="删除">
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>

                {/* Expanded: model list */}
                {expanded && (
                  <div className="border-t border-gray-100 px-4 pb-3 pt-2">
                    <div className="mb-2 flex items-center justify-between">
                      <span className="text-xs font-medium text-gray-500">模型预设</span>
                      <button
                        onClick={() => { setModelForm(emptyModelForm()); setShowModelCreate(p.id); }}
                        className="inline-flex items-center gap-1 rounded bg-gray-100 px-2 py-1 text-xs text-gray-600 hover:bg-gray-200"
                      >
                        <Plus className="h-3 w-3" />
                        添加
                      </button>
                    </div>
                    {models.length === 0 ? (
                      <div className="py-3 text-center text-xs text-gray-400">暂无模型预设</div>
                    ) : (
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="border-b border-gray-100 text-left text-xs text-gray-400">
                            <th className="pb-1 font-medium">模型 ID</th>
                            <th className="pb-1 font-medium">显示名称</th>
                            <th className="pb-1 font-medium text-center">默认</th>
                            <th className="pb-1 font-medium text-center">排序</th>
                            <th className="pb-1 font-medium text-right">操作</th>
                          </tr>
                        </thead>
                        <tbody>
                          {models.map((m) => (
                            <tr key={m.id} className="border-b border-gray-50 last:border-0">
                              <td className="py-1.5 font-mono text-xs text-gray-700">{m.model_id}</td>
                              <td className="py-1.5 text-gray-600">{m.name}</td>
                              <td className="py-1.5 text-center">
                                {m.is_default && <span className="text-green-600 text-xs">✓</span>}
                              </td>
                              <td className="py-1.5 text-center text-gray-400">{m.sort_order}</td>
                              <td className="py-1.5 text-right">
                                <button onClick={() => openEditModel(m)} className="mr-1 rounded p-1 text-gray-400 hover:text-gray-600" title="编辑">
                                  <Pencil className="h-3 w-3" />
                                </button>
                                <button onClick={() => handleDeleteModel(m)} className="rounded p-1 text-gray-400 hover:text-red-600" title="删除">
                                  <Trash2 className="h-3 w-3" />
                                </button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    )}

                    {/* API Key info */}
                    {p.api_key && (
                      <div className="mt-2 text-xs text-gray-400">
                        API Key: <span className="font-mono">{p.api_key}</span>
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
