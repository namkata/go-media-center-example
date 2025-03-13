-- Drop indexes
DROP INDEX IF EXISTS idx_media_tags_tag_id;
DROP INDEX IF EXISTS idx_media_tags_media_id;
DROP INDEX IF EXISTS idx_tags_name;
DROP INDEX IF EXISTS idx_media_deleted_at;
DROP INDEX IF EXISTS idx_media_folder_id;
DROP INDEX IF EXISTS idx_media_user_id;
DROP INDEX IF EXISTS idx_folders_parent_id;
DROP INDEX IF EXISTS idx_folders_user_id;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;

-- Drop tables
DROP TABLE IF EXISTS media_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS media;
DROP TABLE IF EXISTS folders;
DROP TABLE IF EXISTS users; 