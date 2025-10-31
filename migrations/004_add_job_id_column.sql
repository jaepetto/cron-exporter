-- Migration: Add ID column to jobs table and update primary key
-- This migration adds an auto-incrementing ID column and changes the primary key
-- from (name, host) composite key to just ID for better referencing

-- Create new table with ID as primary key
CREATE TABLE jobs_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    host TEXT NOT NULL,
    api_key TEXT,
    automatic_failure_threshold INTEGER NOT NULL DEFAULT 3600,
    labels TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    last_reported_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, host) -- Keep name+host combination unique
);

-- Copy data from old table to new table (if it exists)
INSERT INTO jobs_new (name, host, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at)
SELECT name, host, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at
FROM jobs
WHERE EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name='jobs');

-- Drop old table
DROP TABLE IF EXISTS jobs;

-- Rename new table
ALTER TABLE jobs_new RENAME TO jobs;

-- Create indexes
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_last_reported ON jobs(last_reported_at);
CREATE INDEX idx_jobs_name_host ON jobs(name, host);

-- Update job_results table to reference job by ID instead of name+host
-- First, add job_id column to job_results table
ALTER TABLE job_results ADD COLUMN job_id INTEGER REFERENCES jobs(id);

-- Create index on job_id for better performance
CREATE INDEX idx_job_results_job_id ON job_results(job_id);