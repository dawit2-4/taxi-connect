
-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL, -- Hashed password, never plaintext
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    reputation_score DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    report_count INTEGER NOT NULL DEFAULT 0,
    accuracy_rate DOUBLE PRECISION DEFAULT 0.0
    -- Percentage of accurate reports (0.0 to 1.0)
);

-- Add index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email
    ON users(email);

-- Add user_id to queue_reports table (nullable for existing reports)
ALTER TABLE queue_reports
    ADD COLUMN IF NOT EXISTS user_id UUID;

-- Add foreign key constraint only if it does not already exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_queue_reports_user'
          AND table_name = 'queue_reports'
          AND table_schema = 'public'
    ) THEN
        ALTER TABLE queue_reports
            ADD CONSTRAINT fk_queue_reports_user
            FOREIGN KEY (user_id)
            REFERENCES users(id)
            ON DELETE SET NULL;
    END IF;
END $$;

-- Add index on user_id for faster lookups by user
CREATE INDEX IF NOT EXISTS idx_queue_reports_user_id
    ON queue_reports(user_id);
