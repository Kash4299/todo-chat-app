-- Drop new schema (reverse order of creation)
DROP TABLE IF EXISTS notifications   CASCADE;
DROP TABLE IF EXISTS reactions        CASCADE;
DROP TABLE IF EXISTS messages         CASCADE;
DROP TABLE IF EXISTS task_activities  CASCADE;
DROP TABLE IF EXISTS tasks            CASCADE;
DROP TABLE IF EXISTS channel_members  CASCADE;
DROP TABLE IF EXISTS channels         CASCADE;
DROP TABLE IF EXISTS workspace_members CASCADE;
DROP TABLE IF EXISTS workspaces       CASCADE;
DROP TABLE IF EXISTS refresh_tokens   CASCADE;
DROP TABLE IF EXISTS user_identities  CASCADE;
DROP TABLE IF EXISTS users            CASCADE;

DROP TRIGGER IF EXISTS trg_messages_updated_at   ON messages;
DROP TRIGGER IF EXISTS trg_tasks_updated_at      ON tasks;
DROP TRIGGER IF EXISTS trg_workspaces_updated_at ON workspaces;
DROP TRIGGER IF EXISTS trg_users_updated_at      ON users;

DROP FUNCTION IF EXISTS trigger_set_updated_at();
DROP FUNCTION IF EXISTS immutable_unaccent(text);
DROP EXTENSION IF EXISTS "unaccent";

-- ==========================================
-- Restore migration 001 schema
-- ==========================================
CREATE TABLE users (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    auth0_id     VARCHAR(255) UNIQUE NOT NULL,
    email        VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    avatar_url   VARCHAR(500),
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tasks (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    status      VARCHAR(50)  DEFAULT 'TODO',
    created_by  UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tasks_created_by ON tasks(created_by);
CREATE INDEX idx_tasks_status     ON tasks(status);

CREATE TABLE task_members (
    task_id   UUID        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      VARCHAR(50) DEFAULT 'MEMBER',
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (task_id, user_id)
);

CREATE INDEX idx_task_members_user_id ON task_members(user_id);

CREATE TABLE messages (
    id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id      UUID        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_type VARCHAR(20) DEFAULT 'TEXT',
    content      TEXT,
    attachments  JSONB       DEFAULT '[]'::jsonb,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    search_vector tsvector GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(content, ''))
    ) STORED
);

CREATE INDEX idx_messages_task_id_created_at ON messages(task_id, created_at DESC);
CREATE INDEX idx_messages_search             ON messages USING GIN(search_vector);
CREATE INDEX idx_messages_attachments        ON messages USING GIN(attachments);
