DROP INDEX IF EXISTS idx_messages_attachments;
DROP INDEX IF EXISTS idx_messages_search;
DROP INDEX IF EXISTS idx_messages_task_id_created_at;
DROP TABLE IF EXISTS messages;

DROP INDEX IF EXISTS idx_task_members_user_id;
DROP TABLE IF EXISTS task_members;

DROP INDEX IF EXISTS idx_tasks_status;
DROP INDEX IF EXISTS idx_tasks_created_by;
DROP TABLE IF EXISTS tasks;

DROP TABLE IF EXISTS users;
