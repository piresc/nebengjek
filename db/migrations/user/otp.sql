-- OTP service database schema

-- OTP table for storing one-time passwords
CREATE TABLE IF NOT EXISTS otps (
    id VARCHAR(36) PRIMARY KEY,
    msisdn VARCHAR(20) NOT NULL,
    code VARCHAR(6) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    INDEX (msisdn, expires_at)
);

-- Update users table to use MSISDN as primary identifier
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS msisdn VARCHAR(20) UNIQUE,

-- Add indexes for MSISDN lookups
CREATE INDEX IF NOT EXISTS idx_users_msisdn ON users(msisdn);