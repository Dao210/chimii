-- Parent-owned child profiles and high-entropy restricted child sessions.
-- PIN hashes use Argon2id with a per-profile salt. Raw session tokens are
-- never stored. Indexes are added CONCURRENTLY in migrations 242-245.
CREATE TABLE child_profile (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    parent_user_id UUID NOT NULL,
    display_name TEXT NOT NULL,
    avatar_seed TEXT NOT NULL DEFAULT '',
    pin_hash TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE child_session (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    token_hash TEXT NOT NULL,
    profile_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    parent_user_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    unlock_failures INTEGER NOT NULL DEFAULT 0,
    unlock_locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
