DROP TABLE IF EXISTS group_context_messages;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS fk_transactions_group;
DROP TABLE IF EXISTS groups;
