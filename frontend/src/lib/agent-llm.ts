export interface AgentLLMConfig {
  provider: string;
  model: string;
}

export interface AgentLLMModelOption {
  value: string;
  label: string;
}

export interface AgentLLMProviderOption {
  value: string;
  label: string;
  models: AgentLLMModelOption[];
}

export const DEFAULT_AGENT_LLM_PROVIDER = "openai";
export const DEFAULT_AGENT_LLM_MODEL = "gpt-4.1-mini";

export const agentLLMProviderOptions: AgentLLMProviderOption[] = [
  {
    value: "openai",
    label: "OpenAI",
    models: [
      { value: "gpt-4.1", label: "GPT-4.1" },
      { value: "gpt-4.1-mini", label: "GPT-4.1 Mini" },
      { value: "gpt-4o", label: "GPT-4o" },
      { value: "gpt-4o-mini", label: "GPT-4o Mini" },
    ],
  },
  {
    value: "deepseek",
    label: "DeepSeek",
    models: [
      { value: "deepseek-chat", label: "DeepSeek Chat" },
      { value: "deepseek-reasoner", label: "DeepSeek Reasoner" },
    ],
  },
  {
    value: "qwen",
    label: "通义千问 Qwen",
    models: [
      { value: "qwen-max", label: "Qwen Max" },
      { value: "qwen-plus", label: "Qwen Plus" },
      { value: "qwen-turbo", label: "Qwen Turbo" },
      { value: "qwen2.5-coder-32b-instruct", label: "Qwen2.5 Coder 32B" },
    ],
  },
  {
    value: "moonshot",
    label: "Moonshot Kimi",
    models: [
      { value: "moonshot-v1-8k", label: "Moonshot V1 8K" },
      { value: "moonshot-v1-32k", label: "Moonshot V1 32K" },
      { value: "moonshot-v1-128k", label: "Moonshot V1 128K" },
      { value: "kimi-k2-0711-preview", label: "Kimi K2 Preview" },
    ],
  },
  {
    value: "zhipu",
    label: "智谱 GLM",
    models: [
      { value: "glm-4-plus", label: "GLM-4 Plus" },
      { value: "glm-4-air", label: "GLM-4 Air" },
      { value: "glm-4-flash", label: "GLM-4 Flash" },
    ],
  },
  {
    value: "siliconflow",
    label: "SiliconFlow",
    models: [
      { value: "Qwen/Qwen2.5-72B-Instruct", label: "Qwen2.5 72B Instruct" },
      { value: "deepseek-ai/DeepSeek-V3", label: "DeepSeek V3" },
      { value: "deepseek-ai/DeepSeek-R1", label: "DeepSeek R1" },
    ],
  },
  {
    value: "openrouter",
    label: "OpenRouter",
    models: [
      { value: "openai/gpt-4.1-mini", label: "OpenAI GPT-4.1 Mini" },
      { value: "anthropic/claude-3.7-sonnet", label: "Claude 3.7 Sonnet" },
      { value: "google/gemini-2.5-flash", label: "Gemini 2.5 Flash" },
    ],
  },
  {
    value: "ollama",
    label: "Ollama",
    models: [
      { value: "qwen2.5:14b", label: "Qwen2.5 14B" },
      { value: "deepseek-r1:14b", label: "DeepSeek R1 14B" },
      { value: "llama3.1:8b", label: "Llama 3.1 8B" },
    ],
  },
];

export function getAgentLLMProviderLabel(provider?: string): string {
  return agentLLMProviderOptions.find((item) => item.value === provider)?.label || provider || "未知提供商";
}

export function getAgentLLMModelOptions(provider?: string): AgentLLMModelOption[] {
  return agentLLMProviderOptions.find((item) => item.value === provider)?.models || [];
}

export function getDefaultAgentLLMModel(provider?: string): string {
  return getAgentLLMModelOptions(provider)[0]?.value || DEFAULT_AGENT_LLM_MODEL;
}

export function isPresetAgentLLMModel(provider: string, model: string): boolean {
  return getAgentLLMModelOptions(provider).some((item) => item.value === model);
}

export function readAgentLLMConfig(metadata?: Record<string, unknown>): AgentLLMConfig {
  const llm = metadata && typeof metadata.llm === "object" && metadata.llm !== null
    ? metadata.llm as Record<string, unknown>
    : null;

  const provider = typeof llm?.provider === "string" && llm.provider.trim()
    ? llm.provider.trim()
    : DEFAULT_AGENT_LLM_PROVIDER;
  const model = typeof llm?.model === "string" && llm.model.trim()
    ? llm.model.trim()
    : getDefaultAgentLLMModel(provider);

  return { provider, model };
}

export function mergeAgentLLMMetadata(
  metadata: Record<string, unknown> | undefined,
  agentType: string,
  llmConfig: AgentLLMConfig,
): Record<string, unknown> {
  const nextMetadata = { ...(metadata || {}) };
  if (agentType === "llm") {
    nextMetadata.llm = {
      provider: llmConfig.provider || DEFAULT_AGENT_LLM_PROVIDER,
      model: llmConfig.model || getDefaultAgentLLMModel(llmConfig.provider),
    };
  } else {
    delete nextMetadata.llm;
  }
  return nextMetadata;
}