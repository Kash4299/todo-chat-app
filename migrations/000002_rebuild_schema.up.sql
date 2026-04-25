-- Drop old schema from migration 001 (cascade handles FK order)
DROP TABLE IF EXISTS messages    CASCADE;
DROP TABLE IF EXISTS task_members CASCADE;
DROP TABLE IF EXISTS tasks        CASCADE;
DROP TABLE IF EXISTS users        CASCADE;

-- ==========================================
-- EXTENSIONS
-- ==========================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "unaccent";

-- ==========================================
-- TRIGGER FUNCTION: auto-update updated_at
-- ==========================================
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ==========================================
-- TABLE: users
-- ==========================================
CREATE TABLE users (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    email        VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    avatar_url   VARCHAR(500),
    status_text  VARCHAR(150) NOT NULL DEFAULT '',
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ==========================================
-- TABLE: user_identities
-- ==========================================
CREATE TABLE user_identities (
    id               UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id          UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         VARCHAR(50)  NOT NULL,
    provider_subject VARCHAR(255) NOT NULL UNIQUE, -- full Auth0 sub, ex: google-oauth2|abc
    email_at_link_time VARCHAR(255) NOT NULL,
    is_primary       BOOLEAN      NOT NULL DEFAULT FALSE,
    linked_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_identities_user_id ON user_identities(user_id);
CREATE UNIQUE INDEX uq_user_identities_primary_per_user ON user_identities(user_id, is_primary) WHERE is_primary = TRUE;
CREATE UNIQUE INDEX uq_user_identities_user_provider ON user_identities(user_id, provider);

-- ==========================================
-- TABLE: refresh_tokens
-- ==========================================
CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash CHAR(64)    NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- ==========================================
-- TABLE: workspaces
-- ==========================================
CREATE TABLE workspaces (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(100) NOT NULL,
    slug       VARCHAR(100) NOT NULL UNIQUE,
    owner_id   UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_workspaces_updated_at
    BEFORE UPDATE ON workspaces
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_workspaces_owner_id ON workspaces(owner_id);

-- ==========================================
-- TABLE: workspace_members
-- ==========================================
CREATE TABLE workspace_members (
    workspace_id UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      UUID        NOT NULL REFERENCES users(id)      ON DELETE CASCADE,
    role         VARCHAR(20) NOT NULL DEFAULT 'MEMBER',
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (workspace_id, user_id),
    CONSTRAINT chk_workspace_member_role CHECK (role IN ('ADMIN', 'MEMBER', 'GUEST'))
);

CREATE INDEX idx_workspace_members_user_id ON workspace_members(user_id);

-- ==========================================
-- TABLE: channels
-- ==========================================
CREATE TABLE channels (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,
    type         VARCHAR(20)  NOT NULL DEFAULT 'PUBLIC',
    created_by   UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_channel_type CHECK (type IN ('PUBLIC', 'PRIVATE', 'DM')),
    CONSTRAINT uq_channel_workspace_name UNIQUE (workspace_id, name)
);

CREATE INDEX idx_channels_workspace_id ON channels(workspace_id);

-- ==========================================
-- TABLE: channel_members
-- ==========================================
CREATE TABLE channel_members (
    channel_id   UUID        NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id      UUID        NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (channel_id, user_id)
);

CREATE INDEX idx_channel_members_user_id ON channel_members(user_id);

-- ==========================================
-- TABLE: tasks
-- ==========================================
CREATE TABLE tasks (
    id             UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id   UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id     UUID                  REFERENCES channels(id)   ON DELETE SET NULL,
    title          VARCHAR(255) NOT NULL,
    description    TEXT         NOT NULL DEFAULT '',
    status         VARCHAR(20)  NOT NULL DEFAULT 'TODO',
    priority       VARCHAR(20)  NOT NULL DEFAULT 'MEDIUM',
    assignee_id    UUID                  REFERENCES users(id) ON DELETE SET NULL,
    created_by     UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    parent_task_id UUID                  REFERENCES tasks(id) ON DELETE CASCADE,
    position       INTEGER      NOT NULL DEFAULT 0,
    due_date       TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_task_status   CHECK (status   IN ('TODO', 'IN_PROGRESS', 'REVIEW', 'DONE')),
    CONSTRAINT chk_task_priority CHECK (priority IN ('LOW', 'MEDIUM', 'HIGH', 'URGENT'))
);

CREATE TRIGGER trg_tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_tasks_workspace_id   ON tasks(workspace_id);
CREATE INDEX idx_tasks_assignee_id    ON tasks(assignee_id);
CREATE INDEX idx_tasks_status         ON tasks(status);
CREATE INDEX idx_tasks_parent_task_id ON tasks(parent_task_id) WHERE parent_task_id IS NOT NULL;

-- ==========================================
-- TABLE: task_activities
-- ==========================================
CREATE TABLE task_activities (
    id            UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id       UUID        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    activity_type VARCHAR(50) NOT NULL,
    old_value     TEXT,
    new_value     TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_activity_type CHECK (activity_type IN (
        'CREATED', 'STATUS_CHANGED', 'PRIORITY_CHANGED',
        'ASSIGNED', 'UNASSIGNED', 'TITLE_CHANGED',
        'DESCRIPTION_CHANGED', 'DUE_DATE_CHANGED'
    ))
);

CREATE INDEX idx_task_activities_task_id ON task_activities(task_id);

-- ==========================================
-- TABLE: messages
-- ==========================================
CREATE TABLE messages (
    id            UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id    UUID        NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id       UUID        NOT NULL REFERENCES users(id)    ON DELETE RESTRICT,
    message_type  VARCHAR(20) NOT NULL DEFAULT 'TEXT',
    content       TEXT        NOT NULL DEFAULT '',
    reply_to_id   UUID                 REFERENCES messages(id) ON DELETE SET NULL,
    is_edited     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- GENERATED: do not write this column directly
    search_vector tsvector GENERATED ALWAYS AS (
        to_tsvector('simple', unaccent(coalesce(content, '')))
    ) STORED,
    CONSTRAINT chk_message_type    CHECK (message_type IN ('TEXT', 'SYSTEM')),
    CONSTRAINT chk_message_content CHECK (
        message_type = 'SYSTEM' OR length(trim(content)) > 0
    )
);

CREATE TRIGGER trg_messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Covering index for chat history (most frequent query)
CREATE INDEX idx_messages_channel_created ON messages(channel_id, created_at DESC);
-- GIN index for full-text search
CREATE INDEX idx_messages_search          ON messages USING GIN(search_vector);
-- Partial index for thread lookups
CREATE INDEX idx_messages_reply_to        ON messages(reply_to_id) WHERE reply_to_id IS NOT NULL;

-- ==========================================
-- TABLE: reactions
-- ==========================================
CREATE TABLE reactions (
    message_id UUID        NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    emoji      VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, emoji)
);

CREATE INDEX idx_reactions_message_id ON reactions(message_id);

-- ==========================================
-- TABLE: notifications
-- ==========================================
CREATE TABLE notifications (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       VARCHAR(50) NOT NULL,
    payload    JSONB       NOT NULL DEFAULT '{}',
    is_read    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_notification_type CHECK (type IN (
        'MENTION', 'TASK_ASSIGNED', 'TASK_DUE', 'CHANNEL_INVITE'
    ))
);

-- Unread badge count query index
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, is_read) WHERE is_read = FALSE;
-- Notification center pagination index
CREATE INDEX idx_notifications_user_time   ON notifications(user_id, created_at DESC);
