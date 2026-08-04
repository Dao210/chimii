-- Open Agent SDK is a server-side Cloud runtime, not a daemon CLI protocol.
-- Preserve historical profiles for operator inspection, but make them inert
-- before narrowing the accepted protocol families for all new writes.
UPDATE runtime_profile
SET enabled = false, updated_at = now()
WHERE protocol_family = 'openagent' AND enabled = true;

ALTER TABLE runtime_profile
    DROP CONSTRAINT IF EXISTS runtime_profile_protocol_family_check;

ALTER TABLE runtime_profile
    ADD CONSTRAINT runtime_profile_protocol_family_check
    CHECK (protocol_family IN (
        'claude',
        'codebuddy',
        'codex',
        'copilot',
        'opencode',
        'openclaw',
        'hermes',
        'pi',
        'cursor',
        'kimi',
        'kiro',
        'antigravity',
        'qoder',
        'traecli',
        'deveco',
        'grok',
        'qwen'
    )) NOT VALID;
