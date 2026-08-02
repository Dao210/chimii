-- Add `openagent` to the built-in runtime profile protocol whitelist.
-- openagent is the in-process Open Agent SDK backend (no CLI subprocess);
-- the daemon runs the agent loop directly via server/runtime SDK.
-- Kept in lockstep with agent.SupportedTypes and agent.New().
-- NOT VALID preserves the historical-row tolerance used by prior family additions.
ALTER TABLE runtime_profile DROP CONSTRAINT IF EXISTS runtime_profile_protocol_family_check;

ALTER TABLE runtime_profile ADD CONSTRAINT runtime_profile_protocol_family_check
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
        'qwen',
        'openagent'
    )) NOT VALID;
