-- LLM Provider 供应商表
CREATE TABLE llm_provider (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key         TEXT NOT NULL,
    name        TEXT NOT NULL,
    base_url    TEXT NOT NULL,
    api_key     TEXT NOT NULL DEFAULT '',
    is_builtin  BOOLEAN NOT NULL DEFAULT false,
    is_enabled  BOOLEAN NOT NULL DEFAULT true,
    sort_order  INT NOT NULL DEFAULT 0,
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_llm_provider_key ON llm_provider (key);

-- LLM Model 模型预设表
CREATE TABLE llm_model (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES llm_provider(id) ON DELETE CASCADE,
    model_id    TEXT NOT NULL,
    name        TEXT NOT NULL,
    is_default  BOOLEAN NOT NULL DEFAULT false,
    sort_order  INT NOT NULL DEFAULT 0,
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_llm_model_provider_model ON llm_model (provider_id, model_id);

-- 内置种子数据：供应商
INSERT INTO llm_provider (key, name, base_url, is_builtin, sort_order) VALUES
    ('openai',      'OpenAI',           'https://api.openai.com/v1',                        true, 1),
    ('deepseek',    'DeepSeek',         'https://api.deepseek.com/v1',                      true, 2),
    ('qwen',        '通义千问 Qwen',     'https://dashscope.aliyuncs.com/compatible-mode/v1', true, 3),
    ('moonshot',    'Moonshot Kimi',    'https://api.moonshot.cn/v1',                       true, 4),
    ('zhipu',       '智谱 GLM',         'https://open.bigmodel.cn/api/paas/v4',             true, 5),
    ('siliconflow', 'SiliconFlow',      'https://api.siliconflow.cn/v1',                    true, 6),
    ('openrouter',  'OpenRouter',       'https://openrouter.ai/api/v1',                     true, 7),
    ('ollama',      'Ollama',           'http://localhost:11434/v1',                         true, 8);

-- 内置种子数据：模型
INSERT INTO llm_model (provider_id, model_id, name, is_default, sort_order)
SELECT p.id, v.model_id, v.name, v.is_default, v.sort_order
FROM llm_provider p
JOIN (VALUES
    ('openai', 'gpt-4.1',                     'GPT-4.1',               false, 1),
    ('openai', 'gpt-4.1-mini',                'GPT-4.1 Mini',          true,  2),
    ('openai', 'gpt-4o',                      'GPT-4o',                false, 3),
    ('openai', 'gpt-4o-mini',                 'GPT-4o Mini',           false, 4),
    ('deepseek', 'deepseek-chat',             'DeepSeek Chat',         true,  1),
    ('deepseek', 'deepseek-reasoner',         'DeepSeek Reasoner',     false, 2),
    ('qwen', 'qwen-max',                      'Qwen Max',              true,  1),
    ('qwen', 'qwen-plus',                     'Qwen Plus',             false, 2),
    ('qwen', 'qwen-turbo',                    'Qwen Turbo',            false, 3),
    ('qwen', 'qwen2.5-coder-32b-instruct',   'Qwen2.5 Coder 32B',    false, 4),
    ('moonshot', 'moonshot-v1-8k',            'Moonshot V1 8K',        true,  1),
    ('moonshot', 'moonshot-v1-32k',           'Moonshot V1 32K',       false, 2),
    ('moonshot', 'moonshot-v1-128k',          'Moonshot V1 128K',      false, 3),
    ('moonshot', 'kimi-k2-0711-preview',      'Kimi K2 Preview',       false, 4),
    ('zhipu', 'glm-4-plus',                   'GLM-4 Plus',            true,  1),
    ('zhipu', 'glm-4-air',                    'GLM-4 Air',             false, 2),
    ('zhipu', 'glm-4-flash',                  'GLM-4 Flash',           false, 3),
    ('siliconflow', 'Qwen/Qwen2.5-72B-Instruct', 'Qwen2.5 72B Instruct', true,  1),
    ('siliconflow', 'deepseek-ai/DeepSeek-V3',    'DeepSeek V3',          false, 2),
    ('siliconflow', 'deepseek-ai/DeepSeek-R1',    'DeepSeek R1',          false, 3),
    ('openrouter', 'openai/gpt-4.1-mini',         'OpenAI GPT-4.1 Mini',  true,  1),
    ('openrouter', 'anthropic/claude-3.7-sonnet',  'Claude 3.7 Sonnet',    false, 2),
    ('openrouter', 'google/gemini-2.5-flash',      'Gemini 2.5 Flash',     false, 3),
    ('ollama', 'qwen2.5:14b',                 'Qwen2.5 14B',          true,  1),
    ('ollama', 'deepseek-r1:14b',             'DeepSeek R1 14B',       false, 2),
    ('ollama', 'llama3.1:8b',                 'Llama 3.1 8B',          false, 3)
) AS v(provider_key, model_id, name, is_default, sort_order) ON p.key = v.provider_key;
