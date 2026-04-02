UPDATE llm_provider
SET sort_order = CASE key
    WHEN 'deepseek' THEN 1
    WHEN 'openai' THEN 2
    ELSE sort_order
END,
updated_at = now()
WHERE key IN ('deepseek', 'openai');

UPDATE llm_model
SET is_default = CASE
        WHEN provider_id = (SELECT id FROM llm_provider WHERE key = 'openai')
             AND model_id = 'gpt-4.1-mini' THEN false
        WHEN provider_id = (SELECT id FROM llm_provider WHERE key = 'deepseek')
             AND model_id = 'deepseek-chat' THEN true
        ELSE is_default
    END,
    sort_order = CASE
        WHEN provider_id = (SELECT id FROM llm_provider WHERE key = 'deepseek')
             AND model_id = 'deepseek-chat' THEN 1
        WHEN provider_id = (SELECT id FROM llm_provider WHERE key = 'openai')
             AND model_id = 'gpt-4.1-mini' THEN 2
        ELSE sort_order
    END,
    updated_at = now()
WHERE (provider_id = (SELECT id FROM llm_provider WHERE key = 'openai') AND model_id = 'gpt-4.1-mini')
   OR (provider_id = (SELECT id FROM llm_provider WHERE key = 'deepseek') AND model_id = 'deepseek-chat');

UPDATE agent
SET metadata = jsonb_set(metadata::jsonb, '{llm}', '{"provider":"deepseek","model":"deepseek-chat"}'::jsonb, true),
    updated_at = now()
WHERE agent_type = 'llm'
  AND metadata::jsonb -> 'llm' = '{"provider":"openai","model":"gpt-4.1-mini"}'::jsonb;