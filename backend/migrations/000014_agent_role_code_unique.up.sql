-- 000014: enforce unique non-empty role_code for agent

DO $$
DECLARE
	duplicate_codes TEXT;
BEGIN
	SELECT string_agg(role_code, ', ' ORDER BY role_code)
	INTO duplicate_codes
	FROM (
		SELECT role_code
		FROM agent
		WHERE role_code <> ''
		GROUP BY role_code
		HAVING COUNT(*) > 1
	) duplicates;

	IF duplicate_codes IS NOT NULL THEN
		RAISE EXCEPTION 'cannot enforce unique agent.role_code; duplicate role_code values exist: %', duplicate_codes;
	END IF;
END $$;

DROP INDEX IF EXISTS idx_agent_role_code;

CREATE UNIQUE INDEX IF NOT EXISTS uq_agent_role_code_non_empty
	ON agent(role_code)
	WHERE role_code <> '';