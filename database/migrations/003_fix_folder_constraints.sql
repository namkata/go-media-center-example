-- Create folders table if not exists
CREATE TABLE IF NOT EXISTS folders (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    parent_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Drop existing foreign key if exists
ALTER TABLE media DROP CONSTRAINT IF EXISTS media_folder_id_fkey;

-- Add new foreign key with proper reference
ALTER TABLE media
    ADD CONSTRAINT media_folder_id_fkey
    FOREIGN KEY (folder_id)
    REFERENCES folders(id)
    ON DELETE SET NULL;

-- Create index on folder_id
CREATE INDEX IF NOT EXISTS idx_media_folder_id ON media(folder_id);
CREATE INDEX IF NOT EXISTS idx_folders_user_id ON folders(user_id);