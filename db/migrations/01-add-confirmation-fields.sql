-- Add confirmation fields to matches table for two-way confirmation system

-- Update the match_status enum to include new confirmation states
ALTER TYPE match_status ADD VALUE 'DRIVER_CONFIRMED';
ALTER TYPE match_status ADD VALUE 'PASSENGER_CONFIRMED';

-- Add confirmation columns to matches table
ALTER TABLE matches ADD COLUMN IF NOT EXISTS driver_confirmed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS passenger_confirmed BOOLEAN NOT NULL DEFAULT FALSE;

-- Create indexes for confirmation fields for better query performance
CREATE INDEX IF NOT EXISTS idx_matches_driver_confirmed ON matches(driver_confirmed);
CREATE INDEX IF NOT EXISTS idx_matches_passenger_confirmed ON matches(passenger_confirmed);
CREATE INDEX IF NOT EXISTS idx_matches_status_confirmations ON matches(status, driver_confirmed, passenger_confirmed);
