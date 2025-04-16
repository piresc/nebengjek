-- User service database schema

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    msisdn VARCHAR(20) UNIQUE NOT NULL,
    fullname VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL, -- 'driver' or 'passenger'
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    rating FLOAT NOT NULL DEFAULT 0
);

-- Drivers table (additional info for users who are drivers)
CREATE TABLE IF NOT EXISTS drivers (
    user_id VARCHAR(36) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    vehicle_type VARCHAR(50) NOT NULL,
    vehicle_plate VARCHAR(20) NOT NULL,
);