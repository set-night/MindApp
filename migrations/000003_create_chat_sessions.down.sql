DROP TABLE IF EXISTS message_files;
DROP TABLE IF EXISTS session_messages;
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_active_session;
DROP TABLE IF EXISTS chat_sessions;
